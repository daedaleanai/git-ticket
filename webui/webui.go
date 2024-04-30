package webui

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"text/template"

	"code.gitea.io/sdk/gitea"
	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/query"
	"github.com/daedaleanai/git-ticket/repository"
)

type Column struct {
	Status  string
	Tickets []*cache.BugExcerpt
}

type (
	Handler              = func(http.ResponseWriter, *http.Request)
	HandlerWithRepoCache = func(*cache.RepoCache, http.ResponseWriter, *http.Request)
)

var (
	//go:embed static
	staticFs embed.FS
	//go:embed templates
	templatesFs embed.FS

	templates = template.Must(template.New("").Funcs(templateHelpers).ParseFS(templatesFs, "templates/*.html"))
)

func handleIndex(repo *cache.RepoCache, w http.ResponseWriter, r *http.Request) {
	query, err := query.Parse(r.URL.Query().Get("q"))
	if err != nil {
		w.WriteHeader(401)
		fmt.Fprintf(w, "Unable to parse query: %s", err)
		return
	}

	columns := map[bug.Status][]*cache.BugExcerpt{}
	for _, id := range repo.QueryBugs(query) {
		t, err := repo.ResolveBugExcerpt(id)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Unable to resolve ticket id: %s", err)
			return
		}
		columns[t.Status] = append(columns[t.Status], t)
	}

	executeTemplate(w, "index.html", columns)
}

func handleTicket(repo *cache.RepoCache, w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	bug, err := repo.ResolveBug(entity.Id(id))
	if err != nil {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Unable to find ticket: %s", err)
	}

	executeTemplate(w, "ticket.html", bug.Snapshot())
}

func executeTemplate(w http.ResponseWriter, template string, data interface{}) {
	buf := &bytes.Buffer{}
	if err := templates.ExecuteTemplate(buf, template, data); err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to render ticket %s: %s", template, err)
	}
	io.Copy(w, buf)
}

func withRepoCache(repo repository.ClockedRepo, handler HandlerWithRepoCache) Handler {
	return func(w http.ResponseWriter, r *http.Request) {
		cache, err := cache.NewRepoCache(repo, false)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Unable to open git cache: %s", err)
			return
		}
		defer cache.Close()

		handler(cache, w, r)
	}
}

func Run(repo repository.ClockedRepo, port int) error {
	http.HandleFunc("/", withRepoCache(repo, handleIndex))
	http.HandleFunc("/ticket/", withRepoCache(repo, handleTicket))
	http.Handle("/static/", http.FileServer(http.FS(staticFs)))

	fmt.Printf("Running webui at http://localhost:%d\n", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

var templateHelpers = template.FuncMap{
	"allStatuses": func() []bug.Status {
		return bug.AllStatuses()
	},
	"workflow": func(s *bug.Snapshot) string {
		for _, l := range s.Labels {
			if l.IsWorkflow() {
				return strings.TrimPrefix(l.String(), "workflow:")
			}
		}
		return ""
	},
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
	"reviewStatusColor": func(s string) string {
		if strings.Contains(s, string(gitea.ReviewStateApproved)) {
			return "bg-success"
		}
		if strings.Contains(s, string(gitea.ReviewStateRequestChanges)) {
			return "bg-danger"
		}
		return "bg-secondary"
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
	"checklist": func(s bug.Label) string {
		return strings.TrimPrefix(string(s), "checklist:")
	},
	"comment": func(s string) string {
		return strings.ReplaceAll(s, "\n", "<br>")
	},
	"identityToName": func(ident identity.Interface) string {
		if ident == nil {
			return "None"
		}
		return ident.DisplayName()
	},
	"splitTitle": func(s string) [2]string {
		if match := regexp.MustCompile(`^\[([a-zA-Z0-9-]+)\] (.*)$`).FindStringSubmatch(s); match != nil {
			return [2]string{match[1], match[2]}
		}
		return [2]string{"<none>", s}
	},
}
