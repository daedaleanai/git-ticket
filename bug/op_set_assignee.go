package bug

import (
	"encoding/json"
	"fmt"

	termtext "github.com/MichaelMure/go-term-text"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/util/timestamp"
)

// SetAssigneeOperation will change the Assignee of a bug
type SetAssigneeOperation struct {
	OpBase
	Assignee identity.Interface `json:"assignee"`
}

// Sign-post method for gqlgen
func (op *SetAssigneeOperation) IsOperation() {}

func (op *SetAssigneeOperation) base() *OpBase {
	return &op.OpBase
}

func (op *SetAssigneeOperation) Id() entity.Id {
	return idOperation(op)
}

func (op *SetAssigneeOperation) Apply(snapshot *Snapshot) {
	snapshot.Assignee = op.Assignee
	snapshot.addActor(op.Author)

	item := &SetAssigneeTimelineItem{
		id:       op.Id(),
		Author:   op.Author,
		UnixTime: timestamp.Timestamp(op.UnixTime),
		Assignee: op.Assignee,
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *SetAssigneeOperation) Validate() error {
	if err := opBaseValidate(op, SetAssigneeOp); err != nil {
		return err
	}

	return nil
}

// UnmarshalJSON is a two step JSON unmarshaling
// This workaround is necessary to avoid the inner OpBase.MarshalJSON
// overriding the outer op's MarshalJSON
func (op *SetAssigneeOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct {
		Assignee json.RawMessage `json:"assignee"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	// delegate the decoding of the identity
	assignee, err := identity.UnmarshalJSON(aux.Assignee)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Assignee = assignee

	return nil
}

// Sign post method for gqlgen
func (op *SetAssigneeOperation) IsAuthored() {}

func NewSetAssigneeOp(author identity.Interface, unixTime int64, assignee identity.Interface) *SetAssigneeOperation {
	return &SetAssigneeOperation{
		OpBase:   newOpBase(SetAssigneeOp, author, unixTime),
		Assignee: assignee,
	}
}

type SetAssigneeTimelineItem struct {
	id       entity.Id
	Author   identity.Interface
	UnixTime timestamp.Timestamp
	Assignee identity.Interface
}

func (s SetAssigneeTimelineItem) Id() entity.Id {
	return s.id
}

func (s SetAssigneeTimelineItem) When() timestamp.Timestamp {
	return s.UnixTime
}

func (s SetAssigneeTimelineItem) String() string {
	return fmt.Sprintf("(%s) %s: set assignee \"%s\"",
		s.UnixTime.Time().Format("2006-01-02 15:04:05"),
		termtext.LeftPadMaxLine(s.Author.DisplayName(), 15, 0),
		s.Assignee.DisplayName())
}

// Sign post method for gqlgen
func (s *SetAssigneeTimelineItem) IsAuthored() {}

// Convenience function to apply the operation
func SetAssignee(b Interface, author identity.Interface, unixTime int64, assignee identity.Interface) (*SetAssigneeOperation, error) {
	op := NewSetAssigneeOp(author, unixTime, assignee)
	if err := op.Validate(); err != nil {
		return nil, err
	}

	b.Append(op)
	return op, nil
}
