package cache

import (
	"regexp"
	"testing"

	"github.com/daedaleanai/git-ticket/query"
	"github.com/stretchr/testify/assert"
)

func TestTitleFilter(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		titleFilter query.TitleFilter
		match       bool
	}{
		{name: "complete match", title: "hello world", titleFilter: query.TitleFilter{Title: &query.LiteralNode{Token: query.Token{TokenType: query.StringToken, Literal: "hello world"}}}, match: true},
		{name: "no match", title: "hello world", titleFilter: query.TitleFilter{Title: &query.LiteralNode{Token: query.Token{TokenType: query.StringToken, Literal: "foo"}}}, match: false},
		{name: "cased title", title: "Hello World", titleFilter: query.TitleFilter{Title: &query.LiteralNode{Token: query.Token{TokenType: query.StringToken, Literal: "hello world"}}}, match: true},
		{name: "cased query", title: "hello world", titleFilter: query.TitleFilter{Title: &query.LiteralNode{Token: query.Token{TokenType: query.StringToken, Literal: "Hello World"}}}, match: true},
		{name: "regex title", title: "hello world", titleFilter: query.TitleFilter{Title: &query.RegexNode{Token: query.Token{TokenType: query.RegexToken, Literal: "^hello.*"}, Regex: *regexp.MustCompile("^hello.*")}}, match: true},
		{name: "regex title", title: "hello world", titleFilter: query.TitleFilter{Title: &query.RegexNode{Token: query.Token{TokenType: query.RegexToken, Literal: "^Hello.*"}, Regex: *regexp.MustCompile("^Hello.*")}}, match: false},

		// Those following tests should work eventually but are left for a future iteration.

		// {name: "cased accents", title: "ÑOÑO", query: "ñoño", match: true},
		// {name: "natural language matching", title: "Århus", query: "Aarhus", match: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excerpt := &BugExcerpt{Title: tt.title}
			assert.Equal(t, tt.match, executeTitleFilter(&tt.titleFilter, nil, excerpt))
		})
	}
}
