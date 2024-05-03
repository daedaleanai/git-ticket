package webui

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/config"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/query"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/daedaleanai/git-ticket/util/timestamp"
)

type XrefRule struct {
	Pattern *regexp.Regexp
	Link    *template.Template
}

type Bookmark struct {
	Label string
	Query string
}

type BookmarkGroup struct {
	Group     string
	Bookmarks []Bookmark
}

type Xref struct {
	FullPattern *regexp.Regexp
	Rules       []XrefRule
}

// Configuration for the web UI.
type WebUiConfig struct {
	Xref
	BookmarkGroups []BookmarkGroup
}

type ApiAction struct {
	Action string
	Ticket string
	Status string
}

type HandlerWithRepoCache = func(*cache.RepoCache, io.Writer, *http.Request) error

var (
	//go:embed static
	staticFs embed.FS

	//go:embed templates
	templatesFs embed.FS
	templates   = template.Must(template.New("").Funcs(templateHelpers).ParseFS(templatesFs, "templates/*.html"))

	webUiConfig WebUiConfig

	colors = []string{
		"#e6194b", "#3cb44b", "#ffe119", "#4363d8", "#f58231", "#911eb4", "#42d4f4", "#f032e6", "#bfef45", "#fabed4",
		"#469990", "#dcbeff", "#9A6324", "#fffac8", "#800000", "#aaffc3", "#808000", "#ffd8b1", "#000075", "#a9a9a9",
	}
)

func handleIndex(repo *cache.RepoCache, w io.Writer, r *http.Request) error {
	qParam := r.URL.Query().Get("q")
	q, err := query.Parse(qParam)
	if err != nil {
		return fmt.Errorf("unable to parse query: %w", err)
	}

	tickets := map[bug.Status][]*cache.BugExcerpt{}
	colorKey := map[string]string{}
	ticketColors := map[entity.Id]string{}
	workflows := map[*bug.Workflow]struct{}{}

	for _, id := range repo.QueryBugs(q) {
		ticket, err := repo.ResolveBugExcerpt(id)
		if err != nil {
			return fmt.Errorf("unable to resolve ticket id: %w", err)
		}
		tickets[ticket.Status] = append(tickets[ticket.Status], ticket)

		for _, label := range ticket.Labels {
			if label.IsWorkflow() {
				workflows[bug.FindWorkflow(ticket.Labels)] = struct{}{}
			}
		}

		key, err := getTicketColorKey(repo, q, ticket)
		if err != nil {
			return fmt.Errorf("failed to determine ticket color: %w", err)
		}
		if key != "" {
			if _, ok := colorKey[key]; !ok {
				colorKey[key] = colors[len(colorKey)%len(colors)]
			}
			ticketColors[ticket.Id] = colorKey[key]
		}
	}

	ticketStatuses := determineWorkflowStatuses(workflows)

	return templates.ExecuteTemplate(w, "index.html", struct {
		Statuses       []bug.Status
		BookmarkGroups []BookmarkGroup
		Tickets        map[bug.Status][]*cache.BugExcerpt
		Colors         map[entity.Id]string
		ColorKey       map[string]string
		Query          string
	}{ticketStatuses, webUiConfig.BookmarkGroups, tickets, ticketColors, colorKey, qParam})
}

func handleTicket(repo *cache.RepoCache, w io.Writer, r *http.Request) error {
	id := r.URL.Query().Get("id")
	ticket, err := repo.ResolveBugPrefix(id)
	if err != nil {
		return fmt.Errorf("unable to find ticket %s: %w", id, err)
	}
	return templates.ExecuteTemplate(w, "ticket.html", ticket.Snapshot())
}

func handleApi(repo *cache.RepoCache, w io.Writer, r *http.Request) error {
	action := ApiAction{}
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		return fmt.Errorf("failed to decode body: %w", err)
	}

	ticket, err := repo.ResolveBug(entity.Id(action.Ticket))
	if err != nil {
		return fmt.Errorf("invalid ticket id: %w", err)
	}

	switch action.Action {
	case "setStatus":
		status, err := bug.StatusFromString(action.Status)
		if err != nil {
			return fmt.Errorf("invalid status %s", action.Status)
		}
		_, err = ticket.SetStatus(status)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid action %s", action.Action)
	}

	if err := ticket.CommitAsNeeded(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	fmt.Fprintln(w, "Success.")
	return nil
}

func withRepoCache(repo repository.ClockedRepo, handler HandlerWithRepoCache) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cache, err := cache.NewRepoCache(repo, false)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Error: unable to open git cache: %s", err)
			return
		}
		defer cache.Close()

		buf := &bytes.Buffer{}
		if err := handler(cache, buf, r); err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Error: %s", err)
			return
		}
		io.Copy(w, buf)
	}
}

func Run(repo repository.ClockedRepo, port int) error {
	if err := loadConfig(repo); err != nil {
		return err
	}

	http.Handle("/static/", http.FileServer(http.FS(staticFs)))
	http.HandleFunc("/", withRepoCache(repo, handleIndex))
	http.HandleFunc("/ticket/", withRepoCache(repo, handleTicket))
	http.HandleFunc("/api", withRepoCache(repo, handleApi))

	fmt.Printf("Running web-ui at http://localhost:%d\n", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func loadConfig(repo repository.ClockedRepo) error {
	// Load bookmarks from the local git config
	localGitConfig, err := repo.LocalConfig().ReadAll("git-bug.webui")
	if err != nil {
		return fmt.Errorf("failed to read webui config from local git config: %w", err)
	}

	bookmarks := localGitConfig["git-bug.webui.bookmarks"]
	if len(bookmarks) > 0 {
		if err := json.Unmarshal([]byte(bookmarks), &webUiConfig.BookmarkGroups); err != nil {
			return fmt.Errorf("failed to unmarshal bookmarks from local git config: %w", err)
		}
	}

	// Load bookmarks from the git ticket config
	gitTicketConfig, err := config.GetConfig(repo, "webui")
	if err != nil {
		return fmt.Errorf("failed to read webui config from git ticket config: %w", err)
	}

	xrefConfig := struct {
		Xref []struct {
			Pattern string
			Link    string
		}
	}{}
	if len(gitTicketConfig) > 0 {
		if err := json.Unmarshal(gitTicketConfig, &xrefConfig); err != nil {
			return fmt.Errorf("failed to unmarshal xref rules from git ticket config: %w", err)
		}
	}

	// Verify that all regexp and templates in the xref configuration actually compile.
	patterns := []string{}
	for _, rule := range xrefConfig.Xref {
		pattern, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return fmt.Errorf("failed to compile xref pattern %s: %w", rule.Pattern, err)
		}
		link, err := template.New("").Parse(rule.Link)
		if err != nil {
			return fmt.Errorf("failed to compile xref link template %s: %w", rule.Link, err)
		}

		webUiConfig.Xref.Rules = append(webUiConfig.Xref.Rules, XrefRule{
			Pattern: pattern,
			Link:    link,
		})

		patterns = append(patterns, rule.Pattern)
	}

	fullPattern := strings.Join(patterns, "|")
	webUiConfig.Xref.FullPattern, err = regexp.Compile(fullPattern)
	if err != nil {
		return fmt.Errorf("failed to compile xref patterns %s: %w", fullPattern, err)
	}

	return nil
}

func getTicketColorKey(repo *cache.RepoCache, q *query.Query, ticket *cache.BugExcerpt) (string, error) {
	key := ""
	switch q.ColorBy {
	case query.ColorByAuthor:
		id, err := repo.ResolveIdentityExcerpt(ticket.AuthorId)
		if err != nil {
			return "", fmt.Errorf("failed to resolve identity %s: %w", ticket.AuthorId, err)
		}
		key = id.DisplayName()

	case query.ColorByAssignee:
		if ticket.AssigneeId != "" {
			break
		}
		id, err := repo.ResolveIdentityExcerpt(ticket.AssigneeId)
		if err != nil {
			return "", fmt.Errorf("failed to resolve identity %s: %w", ticket.AssigneeId, err)
		}
		key = id.DisplayName()

	case query.ColorByLabel:
		labels := []string{}
		for _, label := range ticket.Labels {
			if strings.HasPrefix(label.String(), string(q.ColorByLabelPrefix)) {
				labels = append(labels, strings.TrimPrefix(label.String(), string(q.ColorByLabelPrefix)))
			}
		}
		sort.Strings(labels)
		key = strings.Join(labels, " ")
	}

	return key, nil
}

func determineWorkflowStatuses(workflows map[*bug.Workflow]struct{}) []bug.Status {
	statusesMap := map[bug.Status]struct{}{}
	for workflow := range workflows {
		if workflow == nil {
			continue
		}
		for _, s := range workflow.AllStatuses() {
			statusesMap[s] = struct{}{}
		}
	}

	statuses := []bug.Status{}
	for _, s := range bug.AllStatuses() {
		if _, ok := statusesMap[s]; ok {
			statuses = append(statuses, s)
		}
	}

	return statuses
}

var templateHelpers = template.FuncMap{
	"ccbStateColor": func(s bug.CcbState) string {
		switch s {
		case bug.ApprovedCcbState:
			return "bg-success"
		case bug.BlockedCcbState:
			return "bg-danger"
		default:
			return "bg-secondary"
		}
	},
	"checklist": func(s bug.Label) string {
		return strings.TrimPrefix(string(s), "checklist:")
	},
	"checklistStateColor": func(s bug.ChecklistState) string {
		switch s {
		case bug.Passed:
			return "bg-success"
		case bug.Failed:
			return "bg-danger"
		default:
			return "bg-secondary"
		}
	},
	"split": func(s template.HTML) template.HTML {
		return template.HTML(strings.ReplaceAll(string(s), "\n", "<br>"))
	},
	"formatTime": func(t time.Time) string {
		return t.Format("2006-01-02 15:04:05")
	},
	"formatTimestamp": func(ts timestamp.Timestamp) string {
		return ts.Time().Format("2006-01-02 15:04:05")
	},
	"identityToName": func(ident identity.Interface) string {
		if ident == nil {
			return ""
		}
		return ident.DisplayName()
	},
	"reviewStatusColor": func(s string) string {
		switch s {
		case string(gitea.ReviewStateApproved):
			return "bg-success"
		case string(gitea.ReviewStateRequestChanges):
			return "bg-danger"
		default:
			return "bg-secondary"
		}
	},
	"splitTitle": func(s string) [2]string {
		if match := regexp.MustCompile(`^\[([a-zA-Z0-9-]+)\] (.*)$`).FindStringSubmatch(s); match != nil {
			return [2]string{match[1], match[2]}
		}
		return [2]string{"<none>", s}
	},
	"ticketStatusColor": func(s bug.Status) string {
		switch s {
		case bug.VettedStatus:
			return "bg-info"
		case bug.InProgressStatus:
			return "bg-info"
		case bug.InReviewStatus:
			return "bg-info"
		case bug.ReviewedStatus:
			return "bg-info"
		case bug.AcceptedStatus:
			return "bg-success"
		case bug.MergedStatus:
			return "bg-success"
		case bug.DoneStatus:
			return "bg-success"
		case bug.RejectedStatus:
			return "bg-danger"
		default:
			return "bg-secondary"
		}
	},
	"workflow": func(s *bug.Snapshot) string {
		for _, l := range s.Labels {
			if l.IsWorkflow() {
				return strings.TrimPrefix(l.String(), "workflow:")
			}
		}
		return ""
	},
	"xref": func(s string) template.HTML {
		return template.HTML(webUiConfig.Xref.FullPattern.ReplaceAllStringFunc(s, func(s string) string {
			for _, rule := range webUiConfig.Xref.Rules {
				if match := rule.Pattern.FindStringSubmatch(s); match != nil {
					link := &bytes.Buffer{}
					if err := rule.Link.Execute(link, match); err != nil {
						panic(err)
					}
					return fmt.Sprintf("<a href=\"%s\">%s</a>", link.String(), match[0])
				}
			}
			return s
		}))
	},
}
