package bug

import (
	"encoding/json"
	"fmt"

	termtext "github.com/MichaelMure/go-term-text"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/util/timestamp"
	"github.com/pkg/errors"
)

var _ Operation = &SetChecklistOperation{}

// SetChecklistOperation will update the checklist associated with a ticket
type SetChecklistOperation struct {
	OpBase
	Checklist Checklist `json:"checklist"`
}

//Sign-post method for gqlgen
func (op *SetChecklistOperation) IsOperation() {}

func (op *SetChecklistOperation) base() *OpBase {
	return &op.OpBase
}

func (op *SetChecklistOperation) Id() entity.Id {
	return idOperation(op)
}

func (op *SetChecklistOperation) Apply(snapshot *Snapshot) {
	if snapshot.Checklists[op.Checklist.Label] == nil {
		snapshot.Checklists[op.Checklist.Label] = make(map[entity.Id]ChecklistSnapshot)
	}
	snapshot.Checklists[op.Checklist.Label][op.Author.Id()] = ChecklistSnapshot{Checklist: op.Checklist, LastEdit: op.Time()}
	snapshot.addActor(op.Author)

	item := &SetChecklistTimelineItem{
		id:        op.Id(),
		Author:    op.Author,
		UnixTime:  timestamp.Timestamp(op.UnixTime),
		Checklist: op.Checklist,
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *SetChecklistOperation) Validate() error {
	if err := opBaseValidate(op, SetChecklistOp); err != nil {
		return err
	}

	for _, s := range op.Checklist.Sections {
		for _, q := range s.Questions {
			if err := q.State.Validate(); err != nil {
				return errors.Wrap(err, "state")
			}
		}
	}

	return nil
}

// UnmarshalJSON is a two step JSON unmarshaling
// This workaround is necessary to avoid the inner OpBase.MarshalJSON
// overriding the outer op's MarshalJSON
func (op *SetChecklistOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct {
		Checklist Checklist `json:"checklist"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Checklist = aux.Checklist

	return nil
}

// Sign post method for gqlgen
func (op *SetChecklistOperation) IsAuthored() {}

func NewSetChecklistOp(author identity.Interface, unixTime int64, cl Checklist) *SetChecklistOperation {
	return &SetChecklistOperation{
		OpBase:    newOpBase(SetChecklistOp, author, unixTime),
		Checklist: cl,
	}
}

type SetChecklistTimelineItem struct {
	id        entity.Id
	Author    identity.Interface
	UnixTime  timestamp.Timestamp
	Checklist Checklist
}

func (s SetChecklistTimelineItem) Id() entity.Id {
	return s.id
}

func (s SetChecklistTimelineItem) When() timestamp.Timestamp {
	return s.UnixTime
}

func (s SetChecklistTimelineItem) String() string {
	return fmt.Sprintf("(%s) %s: edited \"%s\"",
		s.UnixTime.Time().Format("2006-01-02 15:04:05"),
		termtext.LeftPadMaxLine(s.Author.DisplayName(), 15, 0),
		s.Checklist.Title)
}

// Sign post method for gqlgen
func (s *SetChecklistTimelineItem) IsAuthored() {}

// Convenience function to apply the operation
func SetChecklist(b Interface, author identity.Interface, unixTime int64, cl Checklist) (*SetChecklistOperation, error) {
	setChecklistOp := NewSetChecklistOp(author, unixTime, cl)

	if err := setChecklistOp.Validate(); err != nil {
		return nil, err
	}

	b.Append(setChecklistOp)
	return setChecklistOp, nil
}
