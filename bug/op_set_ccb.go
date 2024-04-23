package bug

import (
	"encoding/json"
	"fmt"
	"strings"

	termtext "github.com/MichaelMure/go-term-text"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/util/timestamp"
)

// SetCcbOperation will change the CCB status of a bug
type SetCcbOperation struct {
	OpBase
	Ccb CcbInfo `json:"ccb"`
}

// Sign-post method for gqlgen
func (op *SetCcbOperation) IsOperation() {}

func (op *SetCcbOperation) base() *OpBase {
	return &op.OpBase
}

func (op *SetCcbOperation) Id() entity.Id {
	return idOperation(op)
}

func (op *SetCcbOperation) Apply(snapshot *Snapshot) {
	// first determine if the user in this operation is already in the ticket CCB group
	inCcb := false
	inCcbIndex := 0
	for i, c := range snapshot.Ccb {
		if c.User.Id() == op.Ccb.User.Id() && c.Status == op.Ccb.Status {
			inCcb = true
			inCcbIndex = i
		}
	}

	// update the snapshot based on this operation
	switch op.Ccb.State {
	case AddedCcbState:
		if inCcb {
			// already in the group, null op
			return
		}
		snapshot.Ccb = append(snapshot.Ccb, op.Ccb)

	case RemovedCcbState:
		if !inCcb {
			// not in the group, null op
			return
		}
		snapshot.Ccb = append(snapshot.Ccb[:inCcbIndex], snapshot.Ccb[inCcbIndex+1:]...)

	case ApprovedCcbState:
		if !inCcb {
			// not in the group, null op
			return
		}
		snapshot.Ccb[inCcbIndex].State = ApprovedCcbState

	case BlockedCcbState:
		if !inCcb {
			// not in the group, null op
			return
		}
		snapshot.Ccb[inCcbIndex].State = BlockedCcbState

	}

	snapshot.addActor(op.Author)

	item := &SetCcbTimelineItem{
		id:       op.Id(),
		Author:   op.Author,
		UnixTime: timestamp.Timestamp(op.UnixTime),
		Ccb:      op.Ccb,
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *SetCcbOperation) Validate() error {
	if err := opBaseValidate(op, SetCcbOp); err != nil {
		return err
	}

	return nil
}

// UnmarshalJSON is a two step JSON unmarshaling
// This workaround is necessary to avoid the inner OpBase.MarshalJSON
// overriding the outer op's MarshalJSON
func (op *SetCcbOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	type CcbInfoJson struct {
		User   json.RawMessage `json:"user"`
		Status Status          `json:"status"`
		State  CcbState        `json:"state"`
	}
	aux := struct {
		Ccb CcbInfoJson `json:"ccb"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	// delegate the decoding of the identity
	user, err := identity.UnmarshalJSON(aux.Ccb.User)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Ccb.User = user
	op.Ccb.Status = aux.Ccb.Status
	op.Ccb.State = aux.Ccb.State

	return nil
}

// Sign post method for gqlgen
func (op *SetCcbOperation) IsAuthored() {}

func NewSetCcbOp(author identity.Interface, unixTime int64, user identity.Interface, status Status, state CcbState) *SetCcbOperation {
	return &SetCcbOperation{
		OpBase: newOpBase(SetCcbOp, author, unixTime),
		Ccb:    CcbInfo{User: user, Status: status, State: state},
	}
}

type SetCcbTimelineItem struct {
	id       entity.Id
	Author   identity.Interface
	UnixTime timestamp.Timestamp
	Ccb      CcbInfo
}

func (s SetCcbTimelineItem) Id() entity.Id {
	return s.id
}

func (s SetCcbTimelineItem) When() timestamp.Timestamp {
	return s.UnixTime
}

func (s SetCcbTimelineItem) String() string {
	var output strings.Builder
	switch s.Ccb.State {
	case AddedCcbState:
		output.WriteString("added \"" + s.Ccb.User.DisplayName() + "\" as CCB approver for status " + s.Ccb.Status.String())
	case RemovedCcbState:
		output.WriteString("removed \"" + s.Ccb.User.DisplayName() + "\" as CCB approver for status " + s.Ccb.Status.String())
	case ApprovedCcbState:
		output.WriteString("approved ticket status " + s.Ccb.Status.String())
	case BlockedCcbState:
		output.WriteString("blocked ticket status " + s.Ccb.Status.String())
	}
	return fmt.Sprintf("(%s) %s: %s",
		s.UnixTime.Time().Format("2006-01-02 15:04:05"),
		termtext.LeftPadMaxLine(s.Author.DisplayName(), timelineDisplayNameWidth, 0),
		output.String())
}

// Sign post method for gqlgen
func (s *SetCcbTimelineItem) IsAuthored() {}

// Convenience function to apply the operation
func SetCcb(b Interface, author identity.Interface, unixTime int64, user identity.Interface, status Status, state CcbState) (*SetCcbOperation, error) {
	op := NewSetCcbOp(author, unixTime, user, status, state)
	if err := op.Validate(); err != nil {
		return nil, err
	}

	b.Append(op)
	return op, nil
}

// Clear CCB approvals of the given user and status
func ClearCcbApprovals(b Interface, author identity.Interface, unixTime int64, user identity.Interface, status Status) (*SetCcbOperation, error) {
	op := NewSetCcbOp(author, unixTime, user, status, RemovedCcbState)
	if err := op.Validate(); err != nil {
		return nil, err
	}

	b.Append(op)
	return op, nil
}
