package webui

import (
	_ "embed"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"text/template"

	"code.gitea.io/sdk/gitea"
	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/query"
)

var (
	//go:embed templates/index.html
	index string
	//go:embed templates/ticket.html
	ticket string

	indexTmpl  = template.Must(template.New("").Parse(index))
	ticketTmpl = template.Must(template.New("").Funcs(template.FuncMap{
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
	}).Parse(ticket))
)

type TicketExcerpt struct {
	Id      string
	ShortId string
	Repo    string
	Title   string
}

type Column struct {
	Status  string
	Tickets []TicketExcerpt
}

type Identity struct {
	Name  string
	Email string
}

type Comment struct {
	Message string
	Author  Identity
}

type Ccb struct {
	Identity Identity
}

type Review struct {
}

type Ticket struct {
	Id           string
	Title        string
	Status       string
	Workflow     string
	Ccb          []Ccb
	Reviews      []Review
	Labels       []bug.Label
	Actors       []Identity
	Participants []Identity
	Comments     []Comment
}

var titleRx = regexp.MustCompile(`^\[([a-zA-Z0-9-]+)\] (.*)$`)

func getRoot(cache *cache.RepoCache) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		columns := map[bug.Status][]TicketExcerpt{}

		query, err := query.Parse(r.URL.Query().Get("q"))
		if err != nil {
			w.Write([]byte("invalid query"))
			return
		}
		for _, id := range cache.QueryBugs(query) {
			t, _ := cache.ResolveBugExcerpt(id)

			ticket := TicketExcerpt{
				Id:      string(t.Id),
				ShortId: string(t.Id[:7]),
				Repo:    "&lt;none&gt;",
				Title:   t.Title,
			}

			if match := titleRx.FindStringSubmatch(t.Title); match != nil {
				ticket.Repo = match[1]
				ticket.Title = match[2]
			}

			columns[t.Status] = append(columns[t.Status], ticket)
		}

		cols := []Column{}
		for _, s := range bug.AllStatuses() {
			if tickets := columns[s]; tickets != nil {
				cols = append(cols, Column{s.String(), tickets})
			}
		}

		indexTmpl.Execute(w, cols)
	}
}

func getTicket(cache *cache.RepoCache) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		bug, _ := cache.ResolveBug(entity.Id(id))
		fmt.Println(ticketTmpl.Execute(w, bug.Snapshot()))
	}
}

func Run(cache *cache.RepoCache) error {
	http.HandleFunc("/", getRoot(cache))
	http.HandleFunc("/ticket/", getTicket(cache))
	return http.ListenAndServe(":3333", nil)
}
