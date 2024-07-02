package query

import (
	"regexp"
	"testing"
	"time"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/stretchr/testify/assert"
)

func TestParseFilters(t *testing.T) {
	getTime := func(input string) time.Time {
		result, err := time.ParseInLocation("2006-01-02", input, time.Local)
		assert.NoError(t, err)
		return result
	}

	var tests = []struct {
		input  string
		filter AstNode
		color  AstNode
		order  AstNode
	}{
		{``, nil, nil, nil},
		{
			`status  (merged, "proposed", inprogress)`,
			&StatusFilter{Statuses: []bug.Status{bug.MergedStatus, bug.ProposedStatus, bug.InProgressStatus}, span: Span{0, 40}},
			nil,
			nil,
		},
		{
			`status(all)`,
			&StatusFilter{Statuses: bug.AllStatuses(), span: Span{0, 11}},
			nil,
			nil,
		},
		{
			`status(ALL)`,
			&StatusFilter{Statuses: bug.AllStatuses(), span: Span{0, 11}},
			nil,
			nil,
		},
		{
			`author("John Doe")`,
			&AuthorFilter{Author: &LiteralNode{Token{StringToken, "John Doe", Span{7, 17}}}, span: Span{0, 18}},
			nil,
			nil,
		},
		{
			`author(Jane)`,
			&AuthorFilter{Author: &LiteralNode{Token{IdentToken, "Jane", Span{7, 11}}}, span: Span{0, 12}},
			nil,
			nil,
		},
		{
			`assignee(Jane)`,
			&AssigneeFilter{Assignee: &LiteralNode{Token{IdentToken, "Jane", Span{9, 13}}}, span: Span{0, 14}},
			nil,
			nil,
		},
		{
			`assignee("Jane Doe")`,
			&AssigneeFilter{Assignee: &LiteralNode{Token{StringToken, "Jane Doe", Span{9, 19}}}, span: Span{0, 20}},
			nil,
			nil,
		},
		{
			`ccb("Jane Doe")`,
			&CcbFilter{Ccb: &LiteralNode{Token{StringToken, "Jane Doe", Span{4, 14}}}, span: Span{0, 15}},
			nil,
			nil,
		},
		{
			`ccb-pending("Jane Doe")`,
			&CcbPendingFilter{Ccb: &LiteralNode{Token{StringToken, "Jane Doe", Span{12, 22}}}, span: Span{0, 23}},
			nil,
			nil,
		},
		{
			`actor("Jane Doe")`,
			&ActorFilter{Actor: &LiteralNode{Token{StringToken, "Jane Doe", Span{6, 16}}}, span: Span{0, 17}},
			nil,
			nil,
		},
		{
			`participant("Jane Doe")`,
			&ParticipantFilter{Participant: &LiteralNode{Token{StringToken, "Jane Doe", Span{12, 22}}}, span: Span{0, 23}},
			nil,
			nil,
		},
		{
			`label("a new label")`,
			&LabelFilter{Label: &LiteralNode{Token{StringToken, "a new label", Span{6, 19}}}, span: Span{0, 20}},
			nil,
			nil,
		},
		{
			`"label"("a new label")`,
			&LabelFilter{Label: &LiteralNode{Token{StringToken, "a new label", Span{8, 21}}}, span: Span{0, 22}},
			nil,
			nil,
		},
		{
			`"title"(mytitle)`,
			&TitleFilter{Title: &LiteralNode{Token{IdentToken, "mytitle", Span{8, 15}}}, span: Span{0, 16}},
			nil,
			nil,
		},
		{
			`title("my title")`,
			&TitleFilter{Title: &LiteralNode{Token{StringToken, "my title", Span{6, 16}}}, span: Span{0, 17}},
			nil,
			nil,
		},
		{
			`title(r"repo:.*")`,
			&TitleFilter{Title: &RegexNode{Token{RegexToken, "repo:.*", Span{6, 16}}, *regexp.MustCompile("repo:.*")}, span: Span{0, 17}},
			nil,
			nil,
		},
		{
			`not(status("proposed", vetted))`,
			&NotFilter{
				&StatusFilter{[]bug.Status{bug.ProposedStatus, bug.VettedStatus}, Span{4, 30}},
				Span{0, 31},
			},
			nil,
			nil,
		},
		{
			`create-before(2026-05-23)`,
			&CreationDateFilter{Date: getTime("2026-05-23"), Before: true, span: Span{0, 25}},
			nil,
			nil,
		},
		{
			`create-after(2026-05-23)`,
			&CreationDateFilter{Date: getTime("2026-05-23"), span: Span{0, 24}},
			nil,
			nil,
		},
		{
			`edit-before(2026-05-23)`,
			&EditDateFilter{Date: getTime("2026-05-23"), Before: true, span: Span{0, 23}},
			nil,
			nil,
		},
		{
			`edit-after(2026-05-23)`,
			&EditDateFilter{Date: getTime("2026-05-23"), span: Span{0, 22}},
			nil,
			nil,
		},
		{
			`edit-after("2026-05-23")`,
			&EditDateFilter{Date: getTime("2026-05-23"), span: Span{0, 24}},
			nil,
			nil,
		},
		{
			`all(status(vetted), label("mylabel"))`,
			&AndFilter{
				Inner: []FilterNode{
					&StatusFilter{[]bug.Status{bug.VettedStatus}, Span{4, 18}},
					&LabelFilter{Label: &LiteralNode{Token{StringToken, "mylabel", Span{26, 35}}}, span: Span{20, 36}},
				},
				span: Span{0, 37},
			},
			nil,
			nil,
		},
		{
			`any(status(vetted), label("mylabel"))`,
			&OrFilter{
				Inner: []FilterNode{
					&StatusFilter{[]bug.Status{bug.VettedStatus}, Span{4, 18}},
					&LabelFilter{Label: &LiteralNode{Token{StringToken, "mylabel", Span{26, 35}}}, span: Span{20, 36}},
				},
				span: Span{0, 37},
			},
			nil,
			nil,
		},
		{
			// Color those tickets that match `Johannes` or `Juan`
			`color-by(author(r"Johannes|Juan"))`,
			nil,
			&ColorByNode{ColorFilter: &AuthorFilter{Author: &RegexNode{Token: Token{RegexToken, "Johannes|Juan", Span{16, 32}}, Regex: *regexp.MustCompile("Johannes|Juan")}, span: Span{9, 33}}, span: Span{0, 34}},
			nil,
		},
		{
			// Color those tickets that match the given assignees
			`color-by(assignee(r"Johannes|Juan"))`,
			nil,
			&ColorByNode{ColorFilter: &AssigneeFilter{Assignee: &RegexNode{Token: Token{RegexToken, "Johannes|Juan", Span{18, 34}}, Regex: *regexp.MustCompile("Johannes|Juan")}, span: Span{9, 35}}, span: Span{0, 36}},
			nil,
		},
		{
			// Color those tickets that match the given labels
			`color-by(label(r"repo:.*"))`,
			nil,
			&ColorByNode{ColorFilter: &LabelFilter{Label: &RegexNode{Token: Token{RegexToken, "repo:.*", Span{15, 25}}, Regex: *regexp.MustCompile("repo:.*")}, span: Span{9, 26}}, span: Span{0, 27}},
			nil,
		},
		{
			// Color those tickets that match the given ccb user
			`color-by(ccb-pending(r"Johannes"))`,
			nil,
			&ColorByNode{ColorFilter: &CcbPendingFilter{Ccb: &RegexNode{Token: Token{RegexToken, "Johannes", Span{21, 32}}, Regex: *regexp.MustCompile("Johannes")}, span: Span{9, 33}}, span: Span{0, 34}},
			nil,
		},
		{
			`color-by(ccb-pending(r"Johannes")) status(vetted)`,
			&StatusFilter{Statuses: []bug.Status{bug.VettedStatus}, span: Span{35, 49}},
			&ColorByNode{ColorFilter: &CcbPendingFilter{Ccb: &RegexNode{Token: Token{RegexToken, "Johannes", Span{21, 32}}, Regex: *regexp.MustCompile("Johannes")}, span: Span{9, 33}}, span: Span{0, 34}},
			nil,
		},
		{
			`color-by(ccb-pending(r"Johannes")) status(vetted) sort(id-asc)`,
			&StatusFilter{Statuses: []bug.Status{bug.VettedStatus}, span: Span{35, 49}},
			&ColorByNode{ColorFilter: &CcbPendingFilter{Ccb: &RegexNode{Token: Token{RegexToken, "Johannes", Span{21, 32}}, Regex: *regexp.MustCompile("Johannes")}, span: Span{9, 33}}, span: Span{0, 34}},
			&OrderByNode{OrderBy: OrderById, OrderDirection: OrderAscending, span: Span{50, 62}},
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
			if tc.color != nil {
				assert.Equal(t, tc.color, ast.ColorNode)
			} else {
				assert.Nil(t, ast.ColorNode)
			}
			if tc.order != nil {
				assert.Equal(t, tc.order, ast.OrderNode)
			} else {
				assert.Nil(t, ast.OrderNode)
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
		{
			`edit-after(r"2026-05-23")`,
			&ParseError{query: "edit-after(r\"2026-05-23\")", span: Span{Begin: 11, End: 24}, message: "Expected Literal expression\n\tWhile parsing Edit Date expression"},
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
