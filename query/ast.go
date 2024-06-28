package query

import (
	"time"

	"github.com/daedaleanai/git-ticket/bug"
)

// A node that implements a filter
type FilterNode interface {
	// Just a marker method. Does not do anything
	filterNode()
}

// Filters a ticket by status
type StatusFilter struct {
	Status bug.Status
}

func (*StatusFilter) filterNode() {}

// Filters a ticket by author name
type AuthorFilter struct {
	AuthorName string
}

func (*AuthorFilter) filterNode() {}

// Filters a ticket by assignee name
type AssigneeFilter struct {
	AssigneeName string
}

func (*AssigneeFilter) filterNode() {}

// Filters a ticket by assigned CCB
type CcbFilter struct {
	CcbName string
}

func (*CcbFilter) filterNode() {}

// Filters a ticket by assigned CCB that is pending approval
type CcbPendingFilter struct {
	CcbName string
}

func (*CcbPendingFilter) filterNode() {}

// Filters a ticket by actor
type ActorFilter struct {
	ActorName string
}

func (*ActorFilter) filterNode() {}

// Filters a ticket by participant
type ParticipantFilter struct {
	ParticipantName string
}

func (*ParticipantFilter) filterNode() {}

// Filters a ticket label by name
type LabelFilter struct {
	LabelName string
}

func (*LabelFilter) filterNode() {}

// Filters a ticket label by title
type TitleFilter struct {
	Title string
}

func (*TitleFilter) filterNode() {}

// Filter that inverts an inner Filter
type NoFilter struct {
	Inner FilterNode
}

func (*NoFilter) filterNode() {}

// Filter tickets by creation date
type CreationDateFilter struct {
	Date   time.Time
	before bool
}

func (*CreationDateFilter) filterNode() {}

// Filter tickets by last edit date
type EditDateFilter struct {
	Date   time.Time
	before bool
}

func (*EditDateFilter) filterNode() {}

// Filter that is matched if all inner filters are true
type AllFilter struct {
	Inner []FilterNode
}

func (*AllFilter) filterNode() {}

// Filter that is matched if any of the inner filters is true
type AnyFilter struct {
	Inner []FilterNode
}

func (*AnyFilter) filterNode() {}

// Selects the ordering of the nodes
type OrderByNode struct {
	OrderBy        OrderBy
	OrderDirection OrderDirection
}

// Selects the ordering of the nodes
type ColorByNode struct {
	ColorBy     ColorBy
	CcbUserName ColorByCcbUserName
	LabelPrefix ColorByLabelPrefix
}
