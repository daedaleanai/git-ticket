package cache

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/query"
)

// resolver has the resolving functions needed by filters.
// This exist mainly to go through the functions of the cache with proper locking.
type resolver interface {
	ResolveIdentityExcerpt(id entity.Id) (*IdentityExcerpt, error)
}

func executeFilter(filter query.FilterNode, resolver resolver, b *BugExcerpt) bool {
	switch filter := filter.(type) {
	case *query.StatusFilter:
		return executeStatusFilter(filter, resolver, b)
	case *query.AuthorFilter:
		return executeAuthorFilter(filter, resolver, b)
	case *query.AssigneeFilter:
		return executeAssigneeFilter(filter, resolver, b)
	case *query.CcbFilter:
		return executeCcbFilter(filter, resolver, b)
	case *query.CcbPendingFilter:
		return executeCcbPendingFilter(filter, resolver, b)
	case *query.ActorFilter:
		return executeActorFilter(filter, resolver, b)
	case *query.ParticipantFilter:
		return executeParticipantFilter(filter, resolver, b)
	case *query.LabelFilter:
		return executeLabelFilter(filter, resolver, b)
	case *query.TitleFilter:
		return executeTitleFilter(filter, resolver, b)
	case *query.NotFilter:
		return executeNotFilter(filter, resolver, b)
	case *query.CreationDateFilter:
		return executeCreationDateFilter(filter, resolver, b)
	case *query.EditDateFilter:
		return executeEditDateFilter(filter, resolver, b)
	case *query.AllFilter:
		return executeAllFilter(filter, resolver, b)
	case *query.AnyFilter:
		return executeAnyFilter(filter, resolver, b)
	default:
		log.Fatal("Unhandled type when executing filter: ", reflect.TypeOf(filter))
		return false
	}
}

func executeMatcherOnIdentity(matcher query.LiteralMatcherNode, resolver resolver, userId entity.Id) bool {
	ident, err := resolver.ResolveIdentityExcerpt(userId)
	if err != nil {
		panic(fmt.Sprintf("Error resolving identity %q for filtering: %v", userId, err))
	}

	return ExecuteMatcherOnIdentity(matcher, ident)
}

func ExecuteMatcherOnIdentity(matcher query.LiteralMatcherNode, ident *IdentityExcerpt) bool {
	switch matcher := matcher.(type) {
	case *query.LiteralNode:
		lit := strings.ToLower(matcher.Token.Literal)
		return ident.Match(lit)
	case *query.RegexNode:
		return matcher.Match(ident.Name)
	default:
		log.Fatal("Unhandled LiteralMatcherNode type: ", reflect.TypeOf(matcher))
		return false
	}
}

func executeStatusFilter(filter *query.StatusFilter, resolver resolver, b *BugExcerpt) bool {
	for _, s := range filter.Statuses {
		if b.Status == s {
			return true
		}
	}
	return false
}

func executeAuthorFilter(filter *query.AuthorFilter, resolver resolver, b *BugExcerpt) bool {
	if b.AuthorId == "" {
		log.Fatal("Ticket does not have an author: ", b.Id.Human())
	}
	return executeMatcherOnIdentity(filter.Author, resolver, b.AuthorId)
}

func executeAssigneeFilter(filter *query.AssigneeFilter, resolver resolver, b *BugExcerpt) bool {
	if b.AssigneeId == "" {
		// Never matches unassigned tickets. To match unassigned tickets: not(assignee(r".*"))
		return false
	}

	return executeMatcherOnIdentity(filter.Assignee, resolver, b.AssigneeId)
}

func executeCcbFilter(filter *query.CcbFilter, resolver resolver, b *BugExcerpt) bool {
	for _, id := range b.Ccb {
		if executeMatcherOnIdentity(filter.Ccb, resolver, id.User) {
			return true
		}
	}

	return false
}

func executeCcbPendingFilter(filter *query.CcbPendingFilter, resolver resolver, b *BugExcerpt) bool {
	workflow := bug.FindWorkflow(b.Labels)
	if workflow == nil {
		// No workflow assigned
		return false
	}

	nextStatuses := workflow.NextStatuses(b.Status)

	// For each of the next possible statuses of the ticket check if there is a ccb assigned,
	// who is the queried user and the associated state is not approved
	for _, id := range b.Ccb {
		if executeMatcherOnIdentity(filter.Ccb, resolver, id.User) {
			for _, nextStatus := range nextStatuses {
				if nextStatus == id.Status && id.State != bug.ApprovedCcbState {
					return true
				}
			}
		}
	}
	return false
}

func executeActorFilter(filter *query.ActorFilter, resolver resolver, b *BugExcerpt) bool {
	for _, id := range b.Actors {
		if executeMatcherOnIdentity(filter.Actor, resolver, id) {
			return true
		}
	}
	return false
}

func executeParticipantFilter(filter *query.ParticipantFilter, resolver resolver, b *BugExcerpt) bool {
	for _, id := range b.Participants {
		if executeMatcherOnIdentity(filter.Participant, resolver, id) {
			return true
		}
	}
	return false
}

func executeLabelFilter(filter *query.LabelFilter, resolver resolver, b *BugExcerpt) bool {
	runMatcher := func(label bug.Label) bool {
		switch matcher := filter.Label.(type) {
		case *query.LiteralNode:
			expected := matcher.Token.Literal
			return expected == string(label)
		case *query.RegexNode:
			return matcher.Match(string(label))
		default:
			log.Fatal("Unhandled LiteralMatcherNode type: ", reflect.TypeOf(filter.Label))
			return false
		}
	}

	for _, l := range b.Labels {
		if runMatcher(l) {
			return true
		}
	}
	return false
}

func executeTitleFilter(filter *query.TitleFilter, resolver resolver, b *BugExcerpt) bool {
	switch matcher := filter.Title.(type) {
	case *query.LiteralNode:
		expected := strings.ToLower(matcher.Token.Literal)
		return expected == strings.ToLower(b.Title)
	case *query.RegexNode:
		return matcher.Match(b.Title)
	default:
		log.Fatal("Unhandled LiteralMatcherNode type: ", reflect.TypeOf(filter.Title))
		return false
	}
}

func executeNotFilter(filter *query.NotFilter, resolver resolver, b *BugExcerpt) bool {
	return !executeFilter(filter.Inner, resolver, b)
}

func executeCreationDateFilter(filter *query.CreationDateFilter, resolver resolver, b *BugExcerpt) bool {
	return filter.Before && b.CreateTime().Before(filter.Date)
}

func executeEditDateFilter(filter *query.EditDateFilter, resolver resolver, b *BugExcerpt) bool {
	return filter.Before && b.EditTime().Before(filter.Date)
}

func executeAllFilter(filter *query.AllFilter, resolver resolver, b *BugExcerpt) bool {
	for _, f := range filter.Inner {
		if !executeFilter(f, resolver, b) {
			return false
		}
	}
	return true
}

func executeAnyFilter(filter *query.AnyFilter, resolver resolver, b *BugExcerpt) bool {
	for _, f := range filter.Inner {
		if executeFilter(f, resolver, b) {
			return true
		}
	}
	return false
}
