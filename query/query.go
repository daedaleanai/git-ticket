package query

import (
	"fmt"
)

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
