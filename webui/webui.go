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
	"github.com/yuin/goldmark"
	gmast "github.com/yuin/goldmark/ast"
	gmextension "github.com/yuin/goldmark/extension"
	gmparser "github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	gmtext "github.com/yuin/goldmark/text"
	gmutil "github.com/yuin/goldmark/util"
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

type ApiActionSetStatus struct {
	Ticket string
	Status string
}

type ApiActionSubmitComment struct {
	Ticket  string
	Comment string
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

type SideBarData struct {
	SelectedQuery  string
	BookmarkGroups []BookmarkGroup
	ColorKey       map[string]string
}

func handleIndex(repo *cache.RepoCache, w io.Writer, r *http.Request) error {
	if r.URL.Path != "/" {
		return fmt.Errorf("Unknown request path: %s", r.URL.Path)
	}
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
		Statuses []bug.Status
		Tickets  map[bug.Status][]*cache.BugExcerpt
		Colors   map[entity.Id]string
		SideBar  SideBarData
	}{
		ticketStatuses,
		tickets,
		ticketColors,
		SideBarData{
			SelectedQuery:  qParam,
			BookmarkGroups: webUiConfig.BookmarkGroups,
			ColorKey:       colorKey,
		},
	})
}

func handleTicket(repo *cache.RepoCache, w io.Writer, r *http.Request) error {
	id := r.URL.Query().Get("id")
	ticket, err := repo.ResolveBugPrefix(id)
	if err != nil {
		return fmt.Errorf("unable to find ticket %s: %w", id, err)
	}

	return templates.ExecuteTemplate(w, "ticket.html", struct {
		SideBar SideBarData
		Ticket  *bug.Snapshot
	}{
		SideBarData{
			BookmarkGroups: webUiConfig.BookmarkGroups,
			ColorKey:       map[string]string{},
		},
		ticket.Snapshot(),
	})
}

func handleChecklist(repo *cache.RepoCache, w io.Writer, r *http.Request) error {
	id := r.URL.Query().Get("id")

	checklist := bug.Label(r.URL.Query().Get("checklist"))
	if !checklist.IsChecklist() {
		return fmt.Errorf("requested checklist %s is not a checklist", checklist)
	}

	ticket, err := repo.ResolveBugPrefix(id)
	if err != nil {
		return fmt.Errorf("unable to find ticket %s: %w", id, err)
	}

	snap := ticket.Snapshot()
	// only display checklists which are currently associated with the ticket
	clMap, present := snap.Checklists[checklist]
	if !present {
		return fmt.Errorf("Checklist %s is not part of ticket %s", checklist, id)
	}

	type checklistItem struct {
		Ident     identity.Interface
		Checklist bug.ChecklistSnapshot
	}

	clList := []checklistItem{}
	for k, v := range clMap {
		id, err := repo.ResolveIdentity(k)
		if err != nil {
			return err
		}
		clList = append(clList, checklistItem{
			Ident:     id.Identity,
			Checklist: v,
		})
	}

	// Sort them so that they appear in consistent order
	sort.Slice(clList, func(i, j int) bool {
		return clList[i].Ident.Id() < clList[j].Ident.Id()
	})

	return templates.ExecuteTemplate(w, "checklist.html", struct {
		Ticket         *bug.Snapshot
		ChecklistLabel bug.Label
		Checklists     []checklistItem
	}{
		Ticket:         snap,
		ChecklistLabel: checklist,
		Checklists:     clList,
	})
}

func handleApiSetStatus(repo *cache.RepoCache, w io.Writer, r *http.Request) error {
	action := ApiActionSetStatus{}
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		return fmt.Errorf("failed to decode body: %w", err)
	}

	ticket, err := repo.ResolveBug(entity.Id(action.Ticket))
	if err != nil {
		return fmt.Errorf("invalid ticket id: %w", err)
	}

	status, err := bug.StatusFromString(action.Status)
	if err != nil {
		return fmt.Errorf("invalid status %s", action.Status)
	}
	_, err = ticket.SetStatus(status)
	if err != nil {
		return err
	}

	if err := ticket.CommitAsNeeded(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	fmt.Fprintln(w, "Success.")
	return nil
}

func handleApiSubmitComment(repo *cache.RepoCache, w io.Writer, r *http.Request) error {
	action := ApiActionSubmitComment{}
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		return fmt.Errorf("failed to decode body: %w", err)
	}

	ticket, err := repo.ResolveBug(entity.Id(action.Ticket))
	if err != nil {
		return fmt.Errorf("invalid ticket id: %w", err)
	}

	_, err = ticket.AddComment(action.Comment)
	if err != nil {
		return err
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

func Run(repo repository.ClockedRepo, host string, port int) error {
	if err := loadConfig(repo); err != nil {
		return err
	}

	http.Handle("/static/", http.FileServer(http.FS(staticFs)))
	http.HandleFunc("/", withRepoCache(repo, handleIndex))
	http.HandleFunc("/ticket/", withRepoCache(repo, handleTicket))
	http.HandleFunc("/checklist/", withRepoCache(repo, handleChecklist))
	http.HandleFunc("/api/set-status", withRepoCache(repo, handleApiSetStatus))
	http.HandleFunc("/api/submit-comment", withRepoCache(repo, handleApiSubmitComment))

	fmt.Printf("Running web-ui at http://%s:%d\n", host, port)
	return http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), nil)
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
	switch q.ColorBy {
	case query.ColorByAuthor:
		id, err := repo.ResolveIdentityExcerpt(ticket.AuthorId)
		if err != nil {
			return "", fmt.Errorf("failed to resolve identity %s: %w", ticket.AuthorId, err)
		}
		return id.DisplayName(), nil

	case query.ColorByAssignee:
		if ticket.AssigneeId != "" {
			break
		}
		id, err := repo.ResolveIdentityExcerpt(ticket.AssigneeId)
		if err != nil {
			return "", fmt.Errorf("failed to resolve identity %s: %w", ticket.AssigneeId, err)
		}
		return id.DisplayName(), nil

	case query.ColorByLabel:
		labels := []string{}
		for _, label := range ticket.Labels {
			if strings.HasPrefix(label.String(), string(q.ColorByLabelPrefix)) {
				labels = append(labels, strings.TrimPrefix(label.String(), string(q.ColorByLabelPrefix)))
			}
		}
		sort.Strings(labels)
		return strings.Join(labels, " "), nil

	case query.ColorByCcbPendingByUser:
		workflow := bug.FindWorkflow(ticket.Labels)
		if workflow == nil {
			// No workflow assigned
			break
		}

		nextStatuses := workflow.NextStatuses(ticket.Status)

		for _, ccbInfo := range ticket.Ccb {
			identityExcerpt, err := repo.ResolveIdentityExcerpt(ccbInfo.User)
			if err != nil {
				return "", err
			}

			if identityExcerpt.Match(string(q.ColorByCcbUserName)) {
				for _, nextStatus := range nextStatuses {
					if nextStatus == ccbInfo.Status && ccbInfo.State != bug.ApprovedCcbState {
						return string(q.ColorByCcbUserName), nil
					}
				}
			}
		}

	}

	return "", nil
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

type xrefTransformer struct {
}

// Applies the xrefs over the given child text node. Returns the next node that should be handled.
func (t xrefTransformer) applyXrefs(n gmast.Node, child *gmast.Text, text string) gmast.Node {
	segmentStart := child.Segment.Start
	segmentStop := child.Segment.Stop

	for _, rule := range webUiConfig.Xref.Rules {
		if match := rule.Pattern.FindStringSubmatchIndex(text); match != nil {
			stringMatches := []string{}
			for i := 0; i < len(match); i += 2 {
				stringMatches = append(stringMatches, text[match[i]:match[i+1]])
			}

			globalMatchStart := match[0]
			globalMatchStop := match[1]

			link := &bytes.Buffer{}
			if err := rule.Link.Execute(link, stringMatches); err != nil {
				panic(err)
			}

			gmLink := gmast.NewLink()
			gmLink.Destination = []byte(link.Bytes())
			gmLinkText := gmast.NewText()
			gmLinkText.Segment = gmtext.Segment{
				Start: segmentStart + globalMatchStart,
				Stop:  segmentStart + globalMatchStop,
			}

			gmLink.AppendChild(gmLink, gmLinkText)

			if globalMatchStart != 0 {
				child.Segment = child.Segment.WithStop(segmentStart + globalMatchStart)
				n.InsertAfter(n, child, gmLink)
			} else {
				n.ReplaceChild(n, child, gmLink)
			}

			if globalMatchStop != (segmentStop - segmentStart) {
				leftoverText := gmast.NewText()
				leftoverText.Segment = gmtext.Segment{
					Start: segmentStart + globalMatchStop,
					Stop:  segmentStop,
				}
				n.InsertAfter(n, gmLink, leftoverText)
			}

			// We are only interested in content after the link
			return gmLink.NextSibling()
		}
	}
	return child.NextSibling()
}

func (t xrefTransformer) Transform(doc *gmast.Document, reader gmtext.Reader, pc gmparser.Context) {
	_ = gmast.Walk(doc, func(n gmast.Node, entering bool) (gmast.WalkStatus, error) {
		kind := n.Kind()
		if kind == gmast.KindFencedCodeBlock || kind == gmast.KindImage || kind == gmast.KindCodeSpan || kind == gmast.KindLink || kind == gmast.KindAutoLink {
			// Do not apply xrefs under these
			return gmast.WalkSkipChildren, nil
		}

		// We apply the transformation when entering a node (before walking its children). It does not really make a difference
		// for this transformation, though.
		if !entering {
			return gmast.WalkContinue, nil
		}

		// Processing the children here instead of in the walk allows us to control the next sibling
		// and skip any nodes that we just added.
		for child := n.FirstChild(); child != nil; {
			if child.Kind() != gmast.KindText {
				child = child.NextSibling()
				continue
			}

			txt := string(child.Text(reader.Source()))
			childText := child.(*gmast.Text)

			child = t.applyXrefs(n, childText, txt)
		}

		return gmast.WalkContinue, nil
	})
}

var md = goldmark.New(
	goldmark.WithExtensions(gmextension.GFM),
	goldmark.WithRendererOptions(
		gmhtml.WithWriter(gmhtml.DefaultWriter),
	),
	goldmark.WithParserOptions(gmparser.WithASTTransformers(gmutil.PrioritizedValue{
		Value: xrefTransformer{},
	})),
)

func applyXrefs(s string, handleMatch func(match []string, link string) string) string {
	for _, rule := range webUiConfig.Xref.Rules {
		if match := rule.Pattern.FindStringSubmatch(s); match != nil {
			link := &bytes.Buffer{}
			if err := rule.Link.Execute(link, match); err != nil {
				panic(err)
			}
			return handleMatch(match, link.String())
		}
	}
	return s
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
		return strings.TrimPrefix(string(s), bug.ChecklistPrefix)
	},
	"checklistStateColor": func(s config.ChecklistState) string {
		switch s {
		case config.Passed:
			return "bg-success"
		case config.Failed:
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
	"getRepo": func(ticket *cache.BugExcerpt) string {
		for _, label := range ticket.Labels {
			if label.IsRepo() {
				return strings.TrimPrefix(string(label), bug.RepoPrefix)
			}
		}
		return "<none>"
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
				return strings.TrimPrefix(l.String(), bug.WorkflowPrefix)
			}
		}
		return ""
	},
	"xref": func(s string) template.HTML {
		return template.HTML(webUiConfig.Xref.FullPattern.ReplaceAllStringFunc(s, func(s string) string {
			return applyXrefs(s, func(match []string, link string) string {
				return fmt.Sprintf("<a href=\"%s\">%s</a>", link, match[0])
			})
		}))
	},
	"mdToHtml": func(s string) template.HTML {
		w := bytes.Buffer{}
		err := md.Convert([]byte(s), &w)
		if err != nil {
			panic(err)
		}
		return template.HTML(w.Bytes())
	},
	"checklistFieldStateColor": func(s config.ChecklistState) string {
		switch s {
		case config.Passed:
			return "bg-success"
		case config.Failed:
			return "bg-danger"
		case config.NotApplicable:
			return "bg-secondary"
		case config.TBD:
			return "bg-warning"
		default:
			return "bg-danger"
		}
	},
}
