package query

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/daedaleanai/git-ticket/bug"
)

type AstNode interface {
	String() string
	Span() Span

	// Just a marker method. Does not do anything
	astNode()
}

type FilterNode interface {
	AstNode

	// Just a marker method. Does not do anything
	filterNode()
}

// A filter that can additionally be used for coloring and not just matching tickets
type ColorFilterNode interface {
	FilterNode

	// Just a marker method. Does not do anything
	colorFilterNode()
}

type LiteralMatcherNode interface {
	AstNode

	// Just a marker method. Does not do anything
	literalMatcherNode()

	Match(text string) bool
}

// Filters a ticket by status
type StatusFilter struct {
	Statuses []bug.Status
	span     Span
}

func (f *StatusFilter) String() string {
	statuses := []string{}
	for _, s := range f.Statuses {
		statuses = append(statuses, s.String())
	}
	return fmt.Sprintf("status(%s)", strings.Join(statuses, ", "))
}
func (f *StatusFilter) Span() Span {
	return f.span
}
func (*StatusFilter) astNode()    {}
func (*StatusFilter) filterNode() {}

// Filters a ticket by author name
type AuthorFilter struct {
	Author LiteralMatcherNode
	span   Span
}

func (f *AuthorFilter) String() string {
	return fmt.Sprintf("author(%s)", f.Author)
}
func (f *AuthorFilter) Span() Span {
	return f.span
}
func (*AuthorFilter) astNode()         {}
func (*AuthorFilter) filterNode()      {}
func (*AuthorFilter) colorFilterNode() {}

// Filters a ticket by assignee name
type AssigneeFilter struct {
	Assignee LiteralMatcherNode
	span     Span
}

func (f *AssigneeFilter) String() string {
	return fmt.Sprintf("assignee(%s)", f.Assignee)
}
func (f *AssigneeFilter) Span() Span {
	return f.span
}
func (*AssigneeFilter) astNode()         {}
func (*AssigneeFilter) filterNode()      {}
func (*AssigneeFilter) colorFilterNode() {}

// Filters a ticket by assigned CCB
type CcbFilter struct {
	Ccb  LiteralMatcherNode
	span Span
}

func (f *CcbFilter) String() string {
	return fmt.Sprintf("ccb(%s)", f.Ccb)
}
func (f *CcbFilter) Span() Span {
	return f.span
}
func (*CcbFilter) astNode()    {}
func (*CcbFilter) filterNode() {}

// Filters a ticket by assigned CCB that is pending approval
type CcbPendingFilter struct {
	Ccb  LiteralMatcherNode
	span Span
}

func (f *CcbPendingFilter) String() string {
	return fmt.Sprintf("ccb-pending(%s)", f.Ccb)
}
func (f *CcbPendingFilter) Span() Span {
	return f.span
}
func (*CcbPendingFilter) astNode()         {}
func (*CcbPendingFilter) filterNode()      {}
func (*CcbPendingFilter) colorFilterNode() {}

// Filters a ticket by actor
type ActorFilter struct {
	Actor LiteralMatcherNode
	span  Span
}

func (f *ActorFilter) String() string {
	return fmt.Sprintf("actor(%s)", f.Actor)
}
func (f *ActorFilter) Span() Span {
	return f.span
}
func (*ActorFilter) astNode()    {}
func (*ActorFilter) filterNode() {}

// Filters a ticket by participant
type ParticipantFilter struct {
	Participant LiteralMatcherNode
	span        Span
}

func (f *ParticipantFilter) String() string {
	return fmt.Sprintf("participant(%s)", f.Participant)
}
func (f *ParticipantFilter) Span() Span {
	return f.span
}
func (*ParticipantFilter) astNode()    {}
func (*ParticipantFilter) filterNode() {}

// Filters a ticket label by name
type LabelFilter struct {
	Label LiteralMatcherNode
	span  Span
}

func (f *LabelFilter) String() string {
	return fmt.Sprintf("label(%s)", f.Label)
}
func (f *LabelFilter) Span() Span {
	return f.span
}
func (*LabelFilter) astNode()         {}
func (*LabelFilter) filterNode()      {}
func (*LabelFilter) colorFilterNode() {}

// Filters a ticket label by title
type TitleFilter struct {
	Title LiteralMatcherNode
	span  Span
}

func (f *TitleFilter) String() string {
	return fmt.Sprintf("label(%s)", f.Title)
}
func (f *TitleFilter) Span() Span {
	return f.span
}
func (*TitleFilter) astNode()    {}
func (*TitleFilter) filterNode() {}

// Filter that inverts an inner Filter
type NotFilter struct {
	Inner FilterNode
	span  Span
}

func (f *NotFilter) String() string {
	return fmt.Sprintf("not(%s)", f.Inner)
}
func (f *NotFilter) Span() Span {
	return f.span
}
func (*NotFilter) astNode()    {}
func (*NotFilter) filterNode() {}

// Filter tickets by creation date
type CreationDateFilter struct {
	Date   time.Time
	Before bool
	span   Span
}

func (f *CreationDateFilter) String() string {
	if f.Before {
		return fmt.Sprintf("create-before(%s)", f.Date)
	}
	return fmt.Sprintf("create-after(%s)", f.Date)
}
func (f *CreationDateFilter) Span() Span {
	return f.span
}
func (*CreationDateFilter) astNode()    {}
func (*CreationDateFilter) filterNode() {}

// Filter tickets by last edit date
type EditDateFilter struct {
	Date   time.Time
	Before bool
	span   Span
}

func (f *EditDateFilter) String() string {
	if f.Before {
		return fmt.Sprintf("edit-before(%s)", f.Date)
	}
	return fmt.Sprintf("edit-after(%s)", f.Date)
}
func (f *EditDateFilter) Span() Span {
	return f.span
}
func (*EditDateFilter) astNode()    {}
func (*EditDateFilter) filterNode() {}

// Filter that is matched if all inner filters are true
type AllFilter struct {
	Inner []FilterNode
	span  Span
}

func (f *AllFilter) String() string {
	inner := []string{}
	for _, f := range f.Inner {
		inner = append(inner, f.String())
	}
	return fmt.Sprintf("all(%s)", strings.Join(inner, ", "))
}
func (f *AllFilter) Span() Span {
	return f.span
}
func (*AllFilter) astNode()    {}
func (*AllFilter) filterNode() {}

// Filter that is matched if any of the inner filters is true
type AnyFilter struct {
	Inner []FilterNode
	span  Span
}

func (f *AnyFilter) String() string {
	inner := []string{}
	for _, f := range f.Inner {
		inner = append(inner, f.String())
	}
	return fmt.Sprintf("any(%s)", strings.Join(inner, ", "))
}
func (f *AnyFilter) Span() Span {
	return f.span
}
func (*AnyFilter) astNode()    {}
func (*AnyFilter) filterNode() {}

// Selects the ordering of the nodes
type OrderByNode struct {
	OrderBy        OrderBy
	OrderDirection OrderDirection
	span           Span
}

func (f *OrderByNode) String() string {
	return fmt.Sprintf("order-by(%v, %v)", f.OrderBy, f.OrderDirection)
}
func (f *OrderByNode) Span() Span {
	return f.span
}
func (*OrderByNode) astNode() {}

// Selects the ordering of the nodes
type ColorByNode struct {
	ColorFilter ColorFilterNode
	span        Span
}

func (f *ColorByNode) String() string {
	return fmt.Sprintf("color-by(%v)", f.ColorFilter)
}
func (f *ColorByNode) Span() Span {
	return f.span
}
func (*ColorByNode) astNode() {}

type LiteralNode struct {
	Token Token
}

func (n *LiteralNode) String() string {
	return fmt.Sprintf("%q", n.Token.Literal)
}
func (n *LiteralNode) Span() Span {
	return n.Token.Span
}
func (*LiteralNode) astNode()            {}
func (*LiteralNode) literalMatcherNode() {}
func (n *LiteralNode) Match(text string) bool {
	return n.Token.Literal == text
}

type RegexNode struct {
	Token Token
	Regex regexp.Regexp
}

func (n *RegexNode) String() string {
	return fmt.Sprintf("r%q", n.Token.Literal)
}
func (n *RegexNode) Span() Span {
	return n.Token.Span
}
func (*RegexNode) astNode()            {}
func (*RegexNode) literalMatcherNode() {}
func (n *RegexNode) Match(text string) bool {
	return n.Regex.Match([]byte(text))
}
