package query

import (
	"fmt"
	"strings"
	"time"

	"github.com/daedaleanai/git-ticket/bug"
)

type AstNode interface {
	String() string

	// Just a marker method. Does not do anything
	astNode()
}

type FilterNode interface {
	AstNode

	// Just a marker method. Does not do anything
	filterNode()
}

// Filters a ticket by status
type StatusFilter struct {
	Statuses []bug.Status
}

func (f *StatusFilter) String() string {
	statuses := []string{}
	for _, s := range f.Statuses {
		statuses = append(statuses, s.String())
	}
	return fmt.Sprintf("status(%s)", strings.Join(statuses, ", "))
}
func (*StatusFilter) astNode()    {}
func (*StatusFilter) filterNode() {}

// Filters a ticket by author name
type AuthorFilter struct {
	AuthorName string
}

func (f *AuthorFilter) String() string {
	return fmt.Sprintf("author(%s)", f.AuthorName)
}
func (*AuthorFilter) astNode()    {}
func (*AuthorFilter) filterNode() {}

// Filters a ticket by assignee name
type AssigneeFilter struct {
	AssigneeName string
}

func (f *AssigneeFilter) String() string {
	return fmt.Sprintf("assignee(%s)", f.AssigneeName)
}
func (*AssigneeFilter) astNode()    {}
func (*AssigneeFilter) filterNode() {}

// Filters a ticket by assigned CCB
type CcbFilter struct {
	CcbName string
}

func (f *CcbFilter) String() string {
	return fmt.Sprintf("ccb(%s)", f.CcbName)
}
func (*CcbFilter) astNode()    {}
func (*CcbFilter) filterNode() {}

// Filters a ticket by assigned CCB that is pending approval
type CcbPendingFilter struct {
	CcbName string
}

func (f *CcbPendingFilter) String() string {
	return fmt.Sprintf("ccb-pending(%s)", f.CcbName)
}
func (*CcbPendingFilter) astNode()    {}
func (*CcbPendingFilter) filterNode() {}

// Filters a ticket by actor
type ActorFilter struct {
	ActorName string
}

func (f *ActorFilter) String() string {
	return fmt.Sprintf("actor(%s)", f.ActorName)
}
func (*ActorFilter) astNode()    {}
func (*ActorFilter) filterNode() {}

// Filters a ticket by participant
type ParticipantFilter struct {
	ParticipantName string
}

func (f *ParticipantFilter) String() string {
	return fmt.Sprintf("participant(%s)", f.ParticipantName)
}
func (*ParticipantFilter) astNode()    {}
func (*ParticipantFilter) filterNode() {}

// Filters a ticket label by name
type LabelFilter struct {
	LabelName string
}

func (f *LabelFilter) String() string {
	return fmt.Sprintf("label(%s)", f.LabelName)
}
func (*LabelFilter) astNode()    {}
func (*LabelFilter) filterNode() {}

// Filters a ticket label by title
type TitleFilter struct {
	Title string
}

func (f *TitleFilter) String() string {
	return fmt.Sprintf("label(%s)", f.Title)
}
func (*TitleFilter) astNode()    {}
func (*TitleFilter) filterNode() {}

// Filter that inverts an inner Filter
type NotFilter struct {
	Inner FilterNode
}

func (f *NotFilter) String() string {
	return fmt.Sprintf("no(%s)", f.Inner)
}
func (*NotFilter) astNode()    {}
func (*NotFilter) filterNode() {}

// Filter tickets by creation date
type CreationDateFilter struct {
	Date   time.Time
	Before bool
}

func (f *CreationDateFilter) String() string {
	if f.Before {
		return fmt.Sprintf("create-before(%s)", f.Date)
	}
	return fmt.Sprintf("create-after(%s)", f.Date)
}
func (*CreationDateFilter) astNode()    {}
func (*CreationDateFilter) filterNode() {}

// Filter tickets by last edit date
type EditDateFilter struct {
	Date   time.Time
	Before bool
}

func (f *EditDateFilter) String() string {
	if f.Before {
		return fmt.Sprintf("edit-before(%s)", f.Date)
	}
	return fmt.Sprintf("edit-after(%s)", f.Date)
}
func (*EditDateFilter) astNode()    {}
func (*EditDateFilter) filterNode() {}

// Filter that is matched if all inner filters are true
type AndFilter struct {
	Inner []FilterNode
}

func (f *AndFilter) String() string {
	inner := []string{}
	for _, f := range f.Inner {
		inner = append(inner, f.String())
	}
	return fmt.Sprintf("all(%s)", strings.Join(inner, ", "))
}
func (*AndFilter) astNode()    {}
func (*AndFilter) filterNode() {}

// Filter that is matched if any of the inner filters is true
type OrFilter struct {
	Inner []FilterNode
}

func (f *OrFilter) String() string {
	inner := []string{}
	for _, f := range f.Inner {
		inner = append(inner, f.String())
	}
	return fmt.Sprintf("any(%s)", strings.Join(inner, ", "))
}
func (*OrFilter) astNode()    {}
func (*OrFilter) filterNode() {}

// Selects the ordering of the nodes
type OrderByNode struct {
	OrderBy        OrderBy
	OrderDirection OrderDirection
}

func (f *OrderByNode) String() string {
	return fmt.Sprintf("order-by(%v, %v)", f.OrderBy, f.OrderDirection)
}
func (*OrderByNode) astNode() {}

// Selects the ordering of the nodes
type ColorByNode struct {
	ColorBy     ColorBy
	CcbUserName ColorByCcbUserName
	LabelPrefix ColorByLabelPrefix
}

func (f *ColorByNode) String() string {
	return fmt.Sprintf("color-by(%v, %v, %v)", f.ColorBy, f.CcbUserName, f.LabelPrefix)
}
func (*ColorByNode) astNode() {}

type LiteralNode struct {
	token Token
}

func (f *LiteralNode) String() string {
	return f.token.Literal
}
func (*LiteralNode) astNode() {}
