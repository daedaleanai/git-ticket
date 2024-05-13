package query

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/daedaleanai/git-ticket/bug"
)

// Parse parse a query DSL
//
// Ex: "status:open author:descartes sort:edit-asc"
//
// Supported filter qualifiers and syntax are described in docs/queries.md
func Parse(query string) (*Query, error) {
	tokens, err := tokenize(query)
	if err != nil {
		return nil, err
	}

	q := &Query{
		OrderBy:        OrderByCreation,
		OrderDirection: OrderDescending,
	}
	sortingDone := false
	coloringDone := false

	for _, t := range tokens {
		switch t.qualifier {
		case "status", "state":
			if strings.EqualFold(t.value, "ALL") {
				q.Status = bug.AllStatuses()
				continue
			}
			status, err := bug.StatusFromString(t.value)
			if err != nil {
				return nil, err
			}
			q.Status = append(q.Status, status)
		case "author":
			q.Author = append(q.Author, t.value)
		case "assignee":
			q.Assignee = append(q.Assignee, t.value)
		case "ccb":
			q.Ccb = append(q.Ccb, t.value)
		case "ccb-pending":
			q.CcbPending = append(q.CcbPending, t.value)
		case "actor":
			q.Actor = append(q.Actor, t.value)
		case "participant":
			q.Participant = append(q.Participant, t.value)
		case "label":
			q.Label = append(q.Label, t.value)
		case "title":
			q.Title = append(q.Title, t.value)
		case "no":
			switch t.value {
			case "label":
				q.NoLabel = true
			default:
				return nil, fmt.Errorf("unknown \"no\" filter \"%s\"", t.value)
			}
		case "create-before":
			parsedTime, err := parseTime(t.value)
			if err != nil {
				return nil, err
			}
			q.CreateBefore = parsedTime
		case "create-after":
			parsedTime, err := parseTime(t.value)
			if err != nil {
				return nil, err
			}
			q.CreateAfter = parsedTime
		case "edit-before":
			parsedTime, err := parseTime(t.value)
			if err != nil {
				return nil, err
			}
			q.EditBefore = parsedTime
		case "edit-after":
			parsedTime, err := parseTime(t.value)
			if err != nil {
				return nil, err
			}
			q.EditAfter = parsedTime
		case "sort":
			if sortingDone {
				return nil, fmt.Errorf("multiple sorting")
			}
			err = parseSorting(q, t.value)
			if err != nil {
				return nil, err
			}
			sortingDone = true
		case "color-by":
			if coloringDone {
				return nil, fmt.Errorf("multiple coloring")
			}
			err = parseColoring(q, t.value)
			if err != nil {
				return nil, err
			}
			coloringDone = true

		default:
			return nil, fmt.Errorf("unknown qualifier \"%s\"", t.qualifier)
		}
	}
	return q, nil
}

func parseSorting(q *Query, value string) error {
	switch value {
	// default ASC
	case "id-desc":
		q.OrderBy = OrderById
		q.OrderDirection = OrderDescending
	case "id", "id-asc":
		q.OrderBy = OrderById
		q.OrderDirection = OrderAscending

	// default DESC
	case "creation", "creation-desc":
		q.OrderBy = OrderByCreation
		q.OrderDirection = OrderDescending
	case "creation-asc":
		q.OrderBy = OrderByCreation
		q.OrderDirection = OrderAscending

	// default DESC
	case "edit", "edit-desc":
		q.OrderBy = OrderByEdit
		q.OrderDirection = OrderDescending
	case "edit-asc":
		q.OrderBy = OrderByEdit
		q.OrderDirection = OrderAscending

	default:
		return fmt.Errorf("unknown sorting %s", value)
	}

	return nil
}

func parseTime(input string) (time.Time, error) {
	var formats = []string{"2006-01-02T15:04:05", "2006-01-02"}

	for _, format := range formats {
		t, err := time.ParseInLocation(format, input, time.Local)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("unrecognized time format")
}

func parseColoring(q *Query, value string) error {
	if strings.HasPrefix(value, "ccb-pending-user:") {
		q.ColorBy = ColorByCcbPendingByUser
		q.ColorByCcbUserName = ColorByCcbUserName(strings.TrimPrefix(value, "ccb-pending-user:"))
		return nil
	}

	if strings.HasPrefix(value, "label:") {
		q.ColorBy = ColorByLabel
		q.ColorByLabelPrefix = ColorByLabelPrefix(strings.TrimPrefix(value, "label:"))
		return nil
	}

	if value == "author" {
		q.ColorBy = ColorByAuthor
		return nil
	}

	if value == "assignee" {
		q.ColorBy = ColorByAssignee
		return nil
	}

	return fmt.Errorf("unknown coloring %s", value)
}
