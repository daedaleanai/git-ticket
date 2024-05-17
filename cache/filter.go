package cache

import (
	"strings"
	"time"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/query"
)

// resolver has the resolving functions needed by filters.
// This exist mainly to go through the functions of the cache with proper locking.
type resolver interface {
	ResolveIdentityExcerpt(id entity.Id) (*IdentityExcerpt, error)
}

// Filter is a predicate that match a subset of bugs
type Filter func(excerpt *BugExcerpt, resolver resolver) bool

// StatusFilter return a Filter that match a bug status
func StatusFilter(status bug.Status) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		return excerpt.Status == status
	}
}

// AuthorFilter return a Filter that match a bug author
func AuthorFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		query = strings.ToLower(query)

		// Normal identity
		if excerpt.AuthorId != "" {
			author, err := resolver.ResolveIdentityExcerpt(excerpt.AuthorId)
			if err != nil {
				panic(err)
			}

			return author.Match(query)
		}

		// Legacy identity support
		return strings.Contains(strings.ToLower(excerpt.LegacyAuthor.Name), query) ||
			strings.Contains(strings.ToLower(excerpt.LegacyAuthor.Login), query)
	}
}

// AssigneeFilter return a Filter that match a bug assignee
func AssigneeFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		query = strings.ToLower(query)

		// Normal identity
		if excerpt.AssigneeId != "" {
			assignee, err := resolver.ResolveIdentityExcerpt(excerpt.AssigneeId)
			if err != nil {
				panic(err)
			}

			return assignee.Match(query)
		} else if query == "unassigned" {
			return true
		}

		return false
	}
}

// CcbFilter return a Filter that match a bug ccb
func CcbFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		query = strings.ToLower(query)

		if query == "unassigned" && len(excerpt.Ccb) == 0 {
			return true
		}

		for _, id := range excerpt.Ccb {
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
}

// CcbPendingFilter return a Filter that matches a ticket with pending ccb approval
func CcbPendingFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		query = strings.ToLower(query)
		workflow := bug.FindWorkflow(excerpt.Labels)
		if workflow == nil {
			// No workflow assigned
			return false
		}
		nextStatuses := workflow.NextStatuses(excerpt.Status)

		// For each of the next possible statuses of the ticket check if there is a ccb assigned,
		// who is the queried user and the associated state is not approved
		for _, id := range excerpt.Ccb {
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
}

// LabelFilter return a Filter that match a label
func LabelFilter(label string) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		for _, l := range excerpt.Labels {
			if string(l) == label {
				return true
			}
		}
		return false
	}
}

// ActorFilter return a Filter that match a bug actor
func ActorFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		query = strings.ToLower(query)

		for _, id := range excerpt.Actors {
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
}

// ParticipantFilter return a Filter that match a bug participant
func ParticipantFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		query = strings.ToLower(query)

		for _, id := range excerpt.Participants {
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
}

// TitleFilter return a Filter that match if the title contains the given query
func TitleFilter(query string) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		return strings.Contains(
			strings.ToLower(excerpt.Title),
			strings.ToLower(query),
		)
	}
}

// NoLabelFilter return a Filter that match the absence of labels
func NoLabelFilter() Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		return len(excerpt.Labels) == 0
	}
}

// CreateBeforeFilter returns a Filter that match create time being before the provided time
func CreateBeforeFilter(filterTime time.Time) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		if filterTime.IsZero() {
			return true
		}
		return excerpt.CreateTime().Before(filterTime)
	}
}

// CreateAfterFilter returns a Filter that match create time being after the provided time
func CreateAfterFilter(filterTime time.Time) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		if filterTime.IsZero() {
			return true
		}
		return excerpt.CreateTime().After(filterTime)
	}
}

// EditBeforeFilter returns a Filter that match edit time being before the provided time
func EditBeforeFilter(filterTime time.Time) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		if filterTime.IsZero() {
			return true
		}
		return excerpt.EditTime().Before(filterTime)
	}
}

// EditAfterFilter returns a Filter that match edit time being after the provided time
func EditAfterFilter(filterTime time.Time) Filter {
	return func(excerpt *BugExcerpt, resolver resolver) bool {
		if filterTime.IsZero() {
			return true
		}
		return excerpt.EditTime().After(filterTime)
	}
}

// Matcher is a collection of Filter that implement a complex filter
type Matcher struct {
	Status      []Filter
	Author      []Filter
	Assignee    []Filter
	Ccb         []Filter
	CcbPending  []Filter
	Actor       []Filter
	Participant []Filter
	Label       []Filter
	Title       []Filter
	NoFilters   []Filter
	TimeFilters []Filter
}

// compileMatcher transform a query.Filters into a specialized matcher
// for the cache.
func compileMatcher(filters query.Filters) *Matcher {
	result := &Matcher{}

	for _, value := range filters.Status {
		result.Status = append(result.Status, StatusFilter(value))
	}
	for _, value := range filters.Author {
		result.Author = append(result.Author, AuthorFilter(value))
	}
	for _, value := range filters.Assignee {
		result.Assignee = append(result.Assignee, AssigneeFilter(value))
	}
	for _, value := range filters.Ccb {
		result.Ccb = append(result.Ccb, CcbFilter(value))
	}
	for _, value := range filters.CcbPending {
		result.CcbPending = append(result.CcbPending, CcbPendingFilter(value))
	}
	for _, value := range filters.Actor {
		result.Actor = append(result.Actor, ActorFilter(value))
	}
	for _, value := range filters.Participant {
		result.Participant = append(result.Participant, ParticipantFilter(value))
	}
	for _, value := range filters.Label {
		result.Label = append(result.Label, LabelFilter(value))
	}
	for _, value := range filters.Title {
		result.Title = append(result.Title, TitleFilter(value))
	}
	result.TimeFilters = append(result.TimeFilters, CreateBeforeFilter(filters.CreateBefore))
	result.TimeFilters = append(result.TimeFilters, CreateAfterFilter(filters.CreateAfter))
	result.TimeFilters = append(result.TimeFilters, EditBeforeFilter(filters.EditBefore))
	result.TimeFilters = append(result.TimeFilters, EditAfterFilter(filters.EditAfter))

	return result
}

// Match check if a bug match the set of filters
func (f *Matcher) Match(excerpt *BugExcerpt, resolver resolver) bool {
	if match := f.orMatch(f.Status, excerpt, resolver); !match {
		return false
	}

	if match := f.orMatch(f.Author, excerpt, resolver); !match {
		return false
	}

	if match := f.orMatch(f.Assignee, excerpt, resolver); !match {
		return false
	}

	if match := f.orMatch(f.Ccb, excerpt, resolver); !match {
		return false
	}

	if match := f.orMatch(f.CcbPending, excerpt, resolver); !match {
		return false
	}

	if match := f.orMatch(f.Participant, excerpt, resolver); !match {
		return false
	}

	if match := f.orMatch(f.Actor, excerpt, resolver); !match {
		return false
	}

	if match := f.orMatch(f.Label, excerpt, resolver); !match {
		return false
	}

	if match := f.andMatch(f.NoFilters, excerpt, resolver); !match {
		return false
	}

	if match := f.andMatch(f.Title, excerpt, resolver); !match {
		return false
	}

	if match := f.andMatch(f.TimeFilters, excerpt, resolver); !match {
		return false
	}

	return true
}

// Check if any of the filters provided match the bug
func (*Matcher) orMatch(filters []Filter, excerpt *BugExcerpt, resolver resolver) bool {
	if len(filters) == 0 {
		return true
	}

	match := false
	for _, f := range filters {
		match = match || f(excerpt, resolver)
	}

	return match
}

// Check if all of the filters provided match the bug
func (*Matcher) andMatch(filters []Filter, excerpt *BugExcerpt, resolver resolver) bool {
	if len(filters) == 0 {
		return true
	}

	match := true
	for _, f := range filters {
		match = match && f(excerpt, resolver)
	}

	return match
}
