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
			&StatusFilter{Statuses: []bug.Status{bug.MergedStatus, bug.ProposedStatus, bug.InProgressStatus}, span: Span{0, 40}},
		},
		{
			`status(all)`,
			&StatusFilter{Statuses: bug.AllStatuses(), span: Span{0, 11}},
		},
		{
			`status(ALL)`,
			&StatusFilter{Statuses: bug.AllStatuses(), span: Span{0, 11}},
		},
		{
			`author("John Doe")`,
			&AuthorFilter{AuthorName: "John Doe", span: Span{0, 18}},
		},
		{
			`author(Jane)`,
			&AuthorFilter{AuthorName: "Jane", span: Span{0, 12}},
		},
		{
			`assignee(Jane)`,
			&AssigneeFilter{AssigneeName: "Jane", span: Span{0, 14}},
		},
		{
			`assignee("Jane Doe")`,
			&AssigneeFilter{AssigneeName: "Jane Doe", span: Span{0, 20}},
		},
		{
			`ccb("Jane Doe")`,
			&CcbFilter{CcbName: "Jane Doe", span: Span{0, 15}},
		},
		{
			`ccb-pending("Jane Doe")`,
			&CcbPendingFilter{CcbName: "Jane Doe", span: Span{0, 23}},
		},
		{
			`actor("Jane Doe")`,
			&ActorFilter{ActorName: "Jane Doe", span: Span{0, 17}},
		},
		{
			`participant("Jane Doe")`,
			&ParticipantFilter{ParticipantName: "Jane Doe", span: Span{0, 23}},
		},
		{
			`label("a new label")`,
			&LabelFilter{LabelName: "a new label", span: Span{0, 20}},
		},
		{
			`"label"("a new label")`,
			&LabelFilter{LabelName: "a new label", span: Span{0, 22}},
		},
		{
			`"title"(mytitle)`,
			&TitleFilter{Title: "mytitle", span: Span{0, 16}},
		},
		{
			`title("my title")`,
			&TitleFilter{Title: "my title", span: Span{0, 17}},
		},
		{
			`not(status("proposed", vetted))`,
			&NotFilter{
				&StatusFilter{[]bug.Status{bug.ProposedStatus, bug.VettedStatus}, Span{4, 30}},
				Span{0, 31},
			},
		},
		{
			`create-before(2026-05-23)`,
			&CreationDateFilter{Date: getTime("2026-05-23"), Before: true, span: Span{0, 25}},
		},
		{
			`create-after(2026-05-23)`,
			&CreationDateFilter{Date: getTime("2026-05-23"), span: Span{0, 24}},
		},
		{
			`edit-before(2026-05-23)`,
			&EditDateFilter{Date: getTime("2026-05-23"), Before: true, span: Span{0, 23}},
		},
		{
			`edit-after(2026-05-23)`,
			&EditDateFilter{Date: getTime("2026-05-23"), span: Span{0, 22}},
		},
		{
			`all(status(vetted), label("mylabel"))`,
			&AndFilter{
				Inner: []FilterNode{
					&StatusFilter{[]bug.Status{bug.VettedStatus}, Span{4, 18}},
					&LabelFilter{LabelName: "mylabel", span: Span{20, 36}},
				},
				span: Span{0, 37},
			},
		},
		{
			`any(status(vetted), label("mylabel"))`,
			&OrFilter{
				Inner: []FilterNode{
					&StatusFilter{[]bug.Status{bug.VettedStatus}, Span{4, 18}},
					&LabelFilter{LabelName: "mylabel", span: Span{20, 36}},
				},
				span: Span{0, 37},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			parser, err := NewParser(tc.input)
			assert.NoError(t, err)

			ast, err := parser.Parse()
			assert.NoError(t, err)
			if tc.filter != nil {
				assert.Equal(t, tc.filter, ast.FilterNode)
			} else {
				assert.Nil(t, ast.FilterNode)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	var tests = []struct {
		input string
		err   error
	}{
		{
			`status(propposed)`,
			&ParseError{`status(propposed)`, Span{7, 16}, "Invalid ticket status\n\tWhile parsing Status expression"},
		},
		{
			`all(status(proposed), not(bleh))`,
			&ParseError{query: "all(status(proposed), not(bleh))", span: Span{Begin: 26, End: 30}, message: "Expected filter expression\n\tWhile parsing Not expression\n\tWhile parsing All expression"},
		},
		{
			`all(sort(id))`,
			&ParseError{query: "all(sort(id))", span: Span{4, 12}, message: "Expected filter expression\n\tWhile parsing All expression"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			parser, err := NewParser(tc.input)
			assert.NoError(t, err)

			_, err = parser.Parse()
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestFormatParserErrrors(t *testing.T) {
	var tests = []struct {
		err      error
		expected string
	}{
		{
			&ParseError{`status(propposed)`, Span{7, 16}, "Invalid ticket status"},
			"Invalid ticket status\nstatus(\x1b[31m\x1b[4mpropposed\x1b[24m\x1b[0m)\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.err.Error())
		})
	}
}
