package webui

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

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
	Pattern string
	Link    string
}

type Bookmark struct {
	Label string
	Query string
}

type BookmarkGroup struct {
	Group     string
	Bookmarks []Bookmark
}

// Configuration for the web UI.
type WebUiConfig struct {
	Xref      []XrefRule
	Bookmarks []BookmarkGroup
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

	for _, id := range repo.QueryBugs(q) {
		ticket, err := repo.ResolveBugExcerpt(id)
		if err != nil {
			return fmt.Errorf("unable to resolve ticket id: %w", err)
		}
		tickets[ticket.Status] = append(tickets[ticket.Status], ticket)

		key := ""
		switch q.ColorBy {
		case query.ColorByAuthor:
			if id, err := repo.ResolveIdentityExcerpt(ticket.AuthorId); err == nil {
				key = id.DisplayName()
			}
		case query.ColorByAssignee:
			if id, err := repo.ResolveIdentityExcerpt(ticket.AssigneeId); err == nil {
				key = id.DisplayName()
			}
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

		if key != "" {
			if _, ok := colorKey[key]; !ok {
				colorKey[key] = colors[len(colorKey)%len(colors)]
			}
			ticketColors[ticket.Id] = colorKey[key]
		}
	}

	return templates.ExecuteTemplate(w, "index.html", struct {
		Statuses  []bug.Status
		Bookmarks []BookmarkGroup
		Tickets   map[bug.Status][]*cache.BugExcerpt
		Colors    map[entity.Id]string
		ColorKey  map[string]string
		Query     string
	}{bug.AllStatuses(), webUiConfig.Bookmarks, tickets, ticketColors, colorKey, qParam})
}

func handleTicket(repo *cache.RepoCache, w io.Writer, r *http.Request) error {
	id := r.URL.Query().Get("id")
	bug, err := repo.ResolveBugPrefix(id)
	if err != nil {
		return fmt.Errorf("unable to find ticket %s: %w", id, err)
	}
	return templates.ExecuteTemplate(w, "ticket.html", bug.Snapshot())
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
	// Load the config. Ignore, if no config can be found.
	data, _ := config.GetConfig(repo, "webui")
	if err := yaml.Unmarshal(data, &webUiConfig); err != nil {
		return fmt.Errorf("failed to unmarshal web-ui config: %w", err)
	}

	http.HandleFunc("/", withRepoCache(repo, handleIndex))
	http.HandleFunc("/ticket/", withRepoCache(repo, handleTicket))
	http.Handle("/static/", http.FileServer(http.FS(staticFs)))

	fmt.Printf("Running web-ui at http://localhost:%d\n", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
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
	"comment": func(s string) string {
		return strings.ReplaceAll(s, "\n", "<br>")
	},
	"formatTime": func(t time.Time) string {
		return t.Format(time.DateTime)
	},
	"formatTimestamp": func(ts timestamp.Timestamp) string {
		return ts.Time().Format(time.DateTime)
	},
	"identityToName": func(ident identity.Interface) string {
		if ident == nil {
			return ""
		}
		return ident.DisplayName()
	},
	"reviewStatusColor": func(s string) string {
		if strings.Contains(s, string(gitea.ReviewStateApproved)) {
			return "bg-success"
		}
		if strings.Contains(s, string(gitea.ReviewStateRequestChanges)) {
			return "bg-danger"
		}
		return "bg-secondary"
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
		patterns := []string{}
		for _, rule := range webUiConfig.Xref {
			patterns = append(patterns, fmt.Sprintf("(?:%s)", rule.Pattern))
		}
		fullPattern := regexp.MustCompile(strings.Join(patterns, "|"))

		return template.HTML(fullPattern.ReplaceAllStringFunc(s, func(s string) string {
			for _, rule := range webUiConfig.Xref {
				if match := regexp.MustCompile(rule.Pattern).FindStringSubmatch(s); match != nil {
					link := &bytes.Buffer{}
					template.Must(template.New("").Parse(rule.Link)).Execute(link, match)
					return fmt.Sprintf("<a href=\"%s\">%s</a>", link.String(), match[0])
				}
			}
			return s
		}))
	},
}
