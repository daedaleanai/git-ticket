package query

import (
	"testing"
	"time"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	getTime := func(input string) time.Time {
		result, err := time.ParseInLocation("2006-01-02", input, time.Local)
		assert.NoError(t, err)
		return result
	}

	var tests = []struct {
		input  string
		filter AstNode
	}{
		{``, nil},
		{
			`status  (merged, "proposed", inprogress)`,
			&StatusFilter{Statuses: []bug.Status{bug.MergedStatus, bug.ProposedStatus, bug.InProgressStatus}},
		},
		{
			`status(all)`,
			&StatusFilter{Statuses: bug.AllStatuses()},
		},
		{
			`status(ALL)`,
			&StatusFilter{Statuses: bug.AllStatuses()},
		},
		{
			`author("John Doe")`,
			&AuthorFilter{AuthorName: "John Doe"},
		},
		{
			`author(Jane)`,
			&AuthorFilter{AuthorName: "Jane"},
		},
		{
			`assignee(Jane)`,
			&AssigneeFilter{AssigneeName: "Jane"},
		},
		{
			`assignee("Jane Doe")`,
			&AssigneeFilter{AssigneeName: "Jane Doe"},
		},
		{
			`ccb("Jane Doe")`,
			&CcbFilter{CcbName: "Jane Doe"},
		},
		{
			`ccb-pending("Jane Doe")`,
			&CcbPendingFilter{CcbName: "Jane Doe"},
		},
		{
			`actor("Jane Doe")`,
			&ActorFilter{ActorName: "Jane Doe"},
		},
		{
			`participant("Jane Doe")`,
			&ParticipantFilter{ParticipantName: "Jane Doe"},
		},
		{
			`label("a new label")`,
			&LabelFilter{LabelName: "a new label"},
		},
		{
			`"label"("a new label")`,
			&LabelFilter{LabelName: "a new label"},
		},
		{
			`"title"(mytitle)`,
			&TitleFilter{Title: "mytitle"},
		},
		{
			`title("my title")`,
			&TitleFilter{Title: "my title"},
		},
		{
			`not(status("proposed", vetted))`,
			&NotFilter{&StatusFilter{[]bug.Status{bug.ProposedStatus, bug.VettedStatus}}},
		},
		{
			`create-before(2026-05-23)`,
			&CreationDateFilter{Date: getTime("2026-05-23"), Before: true},
		},
		{
			`create-after(2026-05-23)`,
			&CreationDateFilter{Date: getTime("2026-05-23")},
		},
		{
			`edit-before(2026-05-23)`,
			&EditDateFilter{Date: getTime("2026-05-23"), Before: true},
		},
		{
			`edit-after(2026-05-23)`,
			&EditDateFilter{Date: getTime("2026-05-23")},
		},
		{
			`all(status(vetted), label("mylabel"))`,
			&AndFilter{Inner: []FilterNode{
				&StatusFilter{[]bug.Status{bug.VettedStatus}},
				&LabelFilter{LabelName: "mylabel"},
			}},
		},
		{
			`any(status(vetted), label("mylabel"))`,
			&OrFilter{Inner: []FilterNode{
				&StatusFilter{[]bug.Status{bug.VettedStatus}},
				&LabelFilter{LabelName: "mylabel"},
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			parser, err := NewParser(tc.input)
			assert.NoError(t, err)

			ast, err := parser.Parse()
			assert.NoError(t, err)
			if tc.filter != nil {
				assert.Equal(t, ast.FilterNode, tc.filter)
			} else {
				assert.Nil(t, ast.FilterNode)
			}
		})
	}
}
