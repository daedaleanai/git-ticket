package cache

import (
	"fmt"
	"testing"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/stretchr/testify/assert"
)

func TestIdentityExcerpt_Match(t *testing.T) {
	testcases := []struct {
		id    entity.Id
		name  string
		query string
		match bool
	}{
		{entity.Id("abcdef01234"), "John Doe", "john", true},
		{entity.Id("abcdef01234"), "John Doe", "johnd", false},
		{entity.Id("abcdef01234"), "John Doe", "j", true},
		{entity.Id("abcdef01234"), "John Doe", "jo", true},
		{entity.Id("abcdef01234"), "John Doe", "joh", true},
		{entity.Id("abcdef01234"), "John Doe", "john", true},
		{entity.Id("abcdef01234"), "John Doe", "john doe", true},
		{entity.Id("abcdef01234"), "John Doe", "a", true},
		{entity.Id("abcdef01234"), "John Doe", "abcdef01234", true},
		{entity.Id("abcdef01234"), "John Doe", "", false},
	}

	for _, tc := range testcases {
		testCaseName := fmt.Sprintf("Id: %q, Name: %q, Query: %q, Match: %v", tc.id, tc.name, tc.query, tc.match)
		t.Run(testCaseName, func(t *testing.T) {
			id := IdentityExcerpt{Id: tc.id, Name: tc.name}
			result := id.Match(tc.query)
			assert.Equal(t, tc.match, result)
		})
	}
}
