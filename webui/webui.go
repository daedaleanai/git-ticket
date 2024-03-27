package webui

import (
	_ "embed"
	"net/http"
	"regexp"
	"text/template"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/query"
)

var (
	//go:embed templates/index.html
	index string
	tmpl  = template.Must(template.New("").Parse(index))
)

type Ticket struct {
	Id    entity.Id
	Repo  string
	Title string
}

type Column struct {
	Status  string
	Tickets []Ticket
}

var titleRx = regexp.MustCompile(`^\[([a-zA-Z0-9-]+)\] (.*)$`)

func getRoot(cache *cache.RepoCache) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		columns := map[bug.Status][]Ticket{}

		query, err := query.Parse(r.URL.Query().Get("q"))
		if err != nil {
			w.Write([]byte("invalid query"))
			return
		}
		for _, id := range cache.QueryBugs(query) {
			t, _ := cache.ResolveBugExcerpt(id)

			ticket := Ticket{
				Id:    t.Id[:7],
				Repo:  "&lt;none&gt;",
				Title: t.Title,
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

		tmpl.Execute(w, cols)
	}
}

func getTicket(cache *cache.RepoCache) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		w.Write([]byte(id))
	}
}

func Run(cache *cache.RepoCache) error {
	http.HandleFunc("/", getRoot(cache))
	http.HandleFunc("/ticket/", getTicket(cache))
	return http.ListenAndServe(":3333", nil)
}
