package query

import (
	"fmt"
	"time"

	"github.com/daedaleanai/git-ticket/bug"
)

// Query is the intermediary representation of a Bug's query. It is either
// produced by parsing a query string (ex: "status:open author:rene") or created
// manually. This query doesn't do anything by itself and need to be interpreted
// for the specific domain of application.
type Query struct {
	Filters
	OrderBy
	OrderDirection
	ColorBy
	ColorByLabelPrefix
	ColorByCcbUserName
}

type CompiledQuery struct {
	FilterNode FilterNode
	OrderNode  *OrderByNode
	ColorNode  *ColorByNode
}

func (q *CompiledQuery) String() string {
	filter := ""
	if q.FilterNode != nil {
		filter = q.FilterNode.String()
	}

	order := ""
	if q.OrderNode != nil {
		order = q.OrderNode.String()
	}

	color := ""
	if q.ColorNode != nil {
		order = q.ColorNode.String()
	}
	return fmt.Sprintf("%s %s %s", filter, order, color)
}

// NewQuery return an identity query with the default sorting (creation-desc).
func NewQuery() *Query {
	return &Query{
		OrderBy:        OrderByCreation,
		OrderDirection: OrderDescending,
	}
}

// Filters is a collection of Filter that implement a complex filter
type Filters struct {
	Status       []bug.Status
	Author       []string
	Assignee     []string
	Ccb          []string
	CcbPending   []string
	Actor        []string
	Participant  []string
	Label        []string
	Title        []string
	NoLabel      bool
	CreateBefore time.Time
	CreateAfter  time.Time
	EditBefore   time.Time
	EditAfter    time.Time
}

type OrderBy int

const (
	_ OrderBy = iota
	OrderById
	OrderByCreation
	OrderByEdit
)

type OrderDirection int

const (
	_ OrderDirection = iota
	OrderAscending
	OrderDescending
)

type ColorBy int

const (
	_ ColorBy = iota
	ColorByAuthor
	ColorByAssignee
	ColorByLabel
	ColorByCcbPendingByUser
)

type ColorByLabelPrefix string

type ColorByCcbUserName string
