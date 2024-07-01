package cache

import (
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
	case *query.AndFilter:
		return executeAndFilter(filter, resolver, b)
	case *query.OrFilter:
		return executeOrFilter(filter, resolver, b)
	default:
		log.Fatal("Unhandled type when executing filter: ", reflect.TypeOf(filter))
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
	query := filter.AuthorName
	query = strings.ToLower(query)

	// Normal identity
	if b.AuthorId != "" {
		author, err := resolver.ResolveIdentityExcerpt(b.AuthorId)
		if err != nil {
			panic(err)
		}

		return author.Match(query)
	}

	// Legacy identity support
	return strings.Contains(strings.ToLower(b.LegacyAuthor.Name), query) ||
		strings.Contains(strings.ToLower(b.LegacyAuthor.Login), query)
}

func executeAssigneeFilter(filter *query.AssigneeFilter, resolver resolver, b *BugExcerpt) bool {
	query := filter.AssigneeName
	query = strings.ToLower(query)

	if query == "" {
		return b.AssigneeId == ""
	}

	if b.AssigneeId != "" {
		assignee, err := resolver.ResolveIdentityExcerpt(b.AssigneeId)
		if err != nil {
			panic(err)
		}

		return assignee.Match(query)
	}

	return false
}

func executeCcbFilter(filter *query.CcbFilter, resolver resolver, b *BugExcerpt) bool {
	query := filter.CcbName
	query = strings.ToLower(query)

	if query == "" {
		return len(b.Ccb) == 0
	}

	for _, id := range b.Ccb {
		identityExcerpt, err := resolver.ResolveIdentityExcerpt(id.User)
		if err != nil {
			panic(err)
		}

		if identityExcerpt.Match(query) {
			return true
		}
	}

	return false
}

func executeCcbPendingFilter(filter *query.CcbPendingFilter, resolver resolver, b *BugExcerpt) bool {
	query := filter.CcbName
	query = strings.ToLower(query)

	workflow := bug.FindWorkflow(b.Labels)
	if workflow == nil {
		// No workflow assigned
		return false
	}

	nextStatuses := workflow.NextStatuses(b.Status)

	// For each of the next possible statuses of the ticket check if there is a ccb assigned,
	// who is the queried user and the associated state is not approved
	for _, id := range b.Ccb {
		identityExcerpt, err := resolver.ResolveIdentityExcerpt(id.User)
		if err != nil {
			panic(err)
		}

		if identityExcerpt.Match(query) {
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
	query := filter.ActorName
	query = strings.ToLower(query)

	for _, id := range b.Actors {
		identityExcerpt, err := resolver.ResolveIdentityExcerpt(id)
		if err != nil {
			panic(err)
		}

		if identityExcerpt.Match(query) {
			return true
		}
	}
	return false
}

func executeParticipantFilter(filter *query.ParticipantFilter, resolver resolver, b *BugExcerpt) bool {
	query := filter.ParticipantName
	query = strings.ToLower(query)

	for _, id := range b.Participants {
		identityExcerpt, err := resolver.ResolveIdentityExcerpt(id)
		if err != nil {
			panic(err)
		}

		if identityExcerpt.Match(query) {
			return true
		}
	}
	return false
}

func executeLabelFilter(filter *query.LabelFilter, resolver resolver, b *BugExcerpt) bool {
	for _, l := range b.Labels {
		if string(l) == filter.LabelName {
			return true
		}
	}
	return false
}

func executeTitleFilter(filter *query.TitleFilter, resolver resolver, b *BugExcerpt) bool {
	return strings.Contains(
		strings.ToLower(b.Title),
		strings.ToLower(filter.Title),
	)
}

func executeNotFilter(filter *query.NotFilter, resolver resolver, b *BugExcerpt) bool {
	return !executeFilter(filter.Inner, resolver, b)
}

func executeCreationDateFilter(filter *query.CreationDateFilter, resolver resolver, b *BugExcerpt) bool {
	if filter.Before {
		return b.CreateTime().Before(filter.Date)
	}
	return b.CreateTime().After(filter.Date)
}

func executeEditDateFilter(filter *query.EditDateFilter, resolver resolver, b *BugExcerpt) bool {
	if filter.Before {
		return b.EditTime().Before(filter.Date)
	}
	return b.EditTime().After(filter.Date)
}

func executeAndFilter(filter *query.AndFilter, resolver resolver, b *BugExcerpt) bool {
	for _, f := range filter.Inner {
		if !executeFilter(f, resolver, b) {
			return false
		}
	}
	return true
}

func executeOrFilter(filter *query.OrFilter, resolver resolver, b *BugExcerpt) bool {
	for _, f := range filter.Inner {
		if executeFilter(f, resolver, b) {
			return true
		}
	}
	return false
}
