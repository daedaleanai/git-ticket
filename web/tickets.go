package web

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/query"
)

func _ticketsListPage(c *gin.Context, excerpts []*cache.BugExcerpt, title string, diag *diagnostic) {
	c.HTML(http.StatusOK, "tickets.html", gin.H{
		"title":    title,
		"excerpts": excerpts,
		"diag":     diag,
	})
}

func _ticketPage(c *gin.Context, tkt *cache.BugCache) {
	c.HTML(http.StatusOK, "ticket.html", gin.H{
		"title":  fmt.Sprintf("[%s] %s", tkt.Id()[0:8], tkt.Snapshot().Title),
		"ticket": &tkt,
	})
}

func _ticketGetBugExcerpts(backend *cache.RepoCache, ticketIds []entity.Id) []*cache.BugExcerpt {
	bugExcerpts := []*cache.BugExcerpt{}
	for _, id := range ticketIds {
		b, err := backend.ResolveBugExcerpt(id)
		if b == nil || err != nil {
			continue
		}
		bugExcerpts = append(bugExcerpts, b)
	}
	return bugExcerpts
}

func handleTicketsList(c *gin.Context, backend *cache.RepoCache) {
	bugIds := backend.QueryBugs(&query.Query{
		Filters:        query.Filters{},
		OrderBy:        query.OrderByEdit,
		OrderDirection: query.OrderDescending,
	})

	_ticketsListPage(c, _ticketGetBugExcerpts(backend, bugIds), "Recently updated", nil)
}

func handleTicket(c *gin.Context, backend *cache.RepoCache) {
	id := c.Param("id")
	if tkt, err := backend.ResolveBugPrefix(id); err == nil {
		_ticketPage(c, tkt)
	} else if ambig, ok := err.(*entity.ErrMultipleMatch); ok {
		_ticketsListPage(c, _ticketGetBugExcerpts(backend, ambig.Matching), "Ambiguous ticket id", diagWarning("Multiple tickets with the specified prefix found, choose one"))
	} else {
		c.HTML(http.StatusNotFound, "ticket.html", gin.H{
			"title": "Ticket not found",
			"diag":  diagNote("Check the ID format, and maybe try a shorter suffix"),
		})
	}
}
