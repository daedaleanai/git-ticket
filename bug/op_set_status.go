package bug

import (
	"encoding/json"
	"fmt"

	termtext "github.com/MichaelMure/go-term-text"
	"github.com/pkg/errors"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/util/timestamp"
)

// SetStatusOperation will change the status of a bug
type SetStatusOperation struct {
	OpBase
	Status Status `json:"status"`
}

// Sign-post method for gqlgen
func (op *SetStatusOperation) IsOperation() {}

func (op *SetStatusOperation) base() *OpBase {
	return &op.OpBase
}

func (op *SetStatusOperation) Id() entity.Id {
	return idOperation(op)
}

func (op *SetStatusOperation) Apply(snapshot *Snapshot) {
	snapshot.Status = op.Status
	snapshot.addActor(op.Author)

	item := &SetStatusTimelineItem{
		id:       op.Id(),
		Author:   op.Author,
		UnixTime: timestamp.Timestamp(op.UnixTime),
		Status:   op.Status,
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *SetStatusOperation) Validate() error {
	if err := opBaseValidate(op, SetStatusOp); err != nil {
		return err
	}

	if err := op.Status.Validate(); err != nil {
		return errors.Wrap(err, "status")
	}

	return nil
}

// UnmarshalJSON is a two step JSON unmarshaling
// This workaround is necessary to avoid the inner OpBase.MarshalJSON
// overriding the outer op's MarshalJSON
func (op *SetStatusOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct {
		Status Status `json:"status"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Status = aux.Status

	return nil
}

// Sign post method for gqlgen
func (op *SetStatusOperation) IsAuthored() {}

func NewSetStatusOp(author identity.Interface, unixTime int64, status Status) *SetStatusOperation {
	return &SetStatusOperation{
		OpBase: newOpBase(SetStatusOp, author, unixTime),
		Status: status,
	}
}

type SetStatusTimelineItem struct {
	id       entity.Id
	Author   identity.Interface
	UnixTime timestamp.Timestamp
	Status   Status
}

func (s SetStatusTimelineItem) Id() entity.Id {
	return s.id
}

func (s SetStatusTimelineItem) When() timestamp.Timestamp {
	return s.UnixTime
}

func (s SetStatusTimelineItem) String() string {
	return fmt.Sprintf("(%s) %s: %s",
		s.UnixTime.Time().Format("2006-01-02 15:04:05"),
		termtext.LeftPadMaxLine(s.Author.DisplayName(), 15, 0),
		s.Status.Action())
}

// Sign post method for gqlgen
func (s *SetStatusTimelineItem) IsAuthored() {}

// Convenience function to apply the operation
func Open(b Interface, author identity.Interface, unixTime int64) (*SetStatusOperation, error) {
	// TODO function left in for now to maintain compatibility with graphql and termui
	op := NewSetStatusOp(author, unixTime, ProposedStatus)
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}

// Convenience function to apply the operation
func Close(b Interface, author identity.Interface, unixTime int64) (*SetStatusOperation, error) {
	// TODO function left in for now to maintain compatibility with graphql and termui
	op := NewSetStatusOp(author, unixTime, MergedStatus)
	if err := op.Validate(); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}

// Convenience function to apply the operation
func SetStatus(b Interface, author identity.Interface, unixTime int64, status Status) (*SetStatusOperation, error) {
	op := NewSetStatusOp(author, unixTime, status)
	if err := op.Validate(); err != nil {
		return nil, err
	}

	snap := b.Compile()
	if err := snap.ValidateTransition(status); err != nil {
		return nil, err
	}
	b.Append(op)
	return op, nil
}
