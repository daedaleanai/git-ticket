package bug

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/util/timestamp"
)

var _ Operation = &LabelChangeOperation{}

// LabelChangeOperation define a Bug operation to add or remove labels
type LabelChangeOperation struct {
	OpBase
	Added   []Label `json:"added"`
	Removed []Label `json:"removed"`
}

// Sign-post method for gqlgen
func (op *LabelChangeOperation) IsOperation() {}

func (op *LabelChangeOperation) base() *OpBase {
	return &op.OpBase
}

func (op *LabelChangeOperation) Id() entity.Id {
	return idOperation(op)
}

// Apply apply the operation
func (op *LabelChangeOperation) Apply(snapshot *Snapshot) {
	snapshot.addActor(op.Author)

	// Add in the set
AddLoop:
	for _, added := range op.Added {
		for _, label := range snapshot.Labels {
			if label == added {
				// Already exist
				continue AddLoop
			}
		}

		snapshot.Labels = append(snapshot.Labels, added)
	}

	// Remove in the set
	for _, removed := range op.Removed {
		for i, label := range snapshot.Labels {
			if label == removed {
				snapshot.Labels[i] = snapshot.Labels[len(snapshot.Labels)-1]
				snapshot.Labels = snapshot.Labels[:len(snapshot.Labels)-1]
			}
		}
	}

	// Sort
	sort.Slice(snapshot.Labels, func(i, j int) bool {
		return string(snapshot.Labels[i]) < string(snapshot.Labels[j])
	})

	item := &LabelChangeTimelineItem{
		id:       op.Id(),
		Author:   op.Author,
		UnixTime: timestamp.Timestamp(op.UnixTime),
		Added:    op.Added,
		Removed:  op.Removed,
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *LabelChangeOperation) Validate() error {
	if err := opBaseValidate(op, LabelChangeOp); err != nil {
		return err
	}

	for _, l := range op.Added {
		if err := l.Validate(); err != nil {
			return errors.Wrap(err, "added label")
		}
	}

	for _, l := range op.Removed {
		if err := l.Validate(); err != nil {
			return errors.Wrap(err, "removed label")
		}
	}

	if len(op.Added)+len(op.Removed) <= 0 {
		return fmt.Errorf("no label change")
	}

	return nil
}

// UnmarshalJSON is a two step JSON unmarshaling
// This workaround is necessary to avoid the inner OpBase.MarshalJSON
// overriding the outer op's MarshalJSON
func (op *LabelChangeOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct {
		Added   []Label `json:"added"`
		Removed []Label `json:"removed"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Added = aux.Added
	op.Removed = aux.Removed

	return nil
}

// Sign post method for gqlgen
func (op *LabelChangeOperation) IsAuthored() {}

func NewLabelChangeOperation(author identity.Interface, unixTime int64, added, removed []Label) *LabelChangeOperation {
	return &LabelChangeOperation{
		OpBase:  newOpBase(LabelChangeOp, author, unixTime),
		Added:   added,
		Removed: removed,
	}
}

type LabelChangeTimelineItem struct {
	id       entity.Id
	Author   identity.Interface
	UnixTime timestamp.Timestamp
	Added    []Label
	Removed  []Label
}

func (l LabelChangeTimelineItem) Id() entity.Id {
	return l.id
}

func (l LabelChangeTimelineItem) When() timestamp.Timestamp {
	return l.UnixTime
}

func (l LabelChangeTimelineItem) String() string {
	var output strings.Builder
	if len(l.Added) > 0 {
		output.WriteString("added labels ")
		for _, label := range l.Added {
			output.WriteString("\"" + string(label) + "\" ")
		}
	}
	if len(l.Removed) > 0 {
		output.WriteString("removed labels ")
		for _, label := range l.Removed {
			output.WriteString("\"" + string(label) + "\" ")
		}
	}
	return fmt.Sprintf("(%s) %-20s: %s",
		l.UnixTime.Time().Format(time.RFC822),
		l.Author.DisplayName(),
		output.String())
}

// Sign post method for gqlgen
func (l *LabelChangeTimelineItem) IsAuthored() {}

// ChangeLabels is a convenience function to apply the operation
func ChangeLabels(b Interface, author identity.Interface, unixTime int64, add, remove []string) ([]LabelChangeResult, *LabelChangeOperation, error) {
	var added, removed []Label
	var results []LabelChangeResult
	var newWorkflow *Workflow

	snap := b.Compile()

	for _, str := range add {
		label := Label(str)

		// check for duplicate
		if labelExist(added, label) {
			results = append(results, LabelChangeResult{Label: label, Status: LabelChangeDuplicateInOp})
			continue
		}

		// check that the label doesn't already exist
		if labelExist(snap.Labels, label) {
			results = append(results, LabelChangeResult{Label: label, Status: LabelChangeAlreadySet})
			continue
		}

		// if it's a workflow, check it exists
		if label.IsWorkflow() {
			if newWorkflow = FindWorkflow(Label(str)); newWorkflow == nil {
				results = append(results, LabelChangeResult{Label: label, Status: LabelChangeInvalidWorkflow})
				continue
			}
			// if so, remove any existing workflows
			for _, lbl := range snap.Labels {
				if lbl.IsWorkflow() {
					remove = append(remove, string(lbl))
				}
			}
		}

		added = append(added, label)
		results = append(results, LabelChangeResult{Label: label, Status: LabelChangeAdded})
	}

	for _, str := range remove {
		label := Label(str)

		// check for duplicate
		if labelExist(removed, label) {
			results = append(results, LabelChangeResult{Label: label, Status: LabelChangeDuplicateInOp})
			continue
		}

		// check that the label actually exist
		if !labelExist(snap.Labels, label) {
			results = append(results, LabelChangeResult{Label: label, Status: LabelChangeDoesntExist})
			continue
		}

		// unless this is a workflow change, don't allow workflow labels to be removed
		if newWorkflow == nil && label.IsWorkflow() {
			results = append(results, LabelChangeResult{Label: label, Status: LabelChangeInvalidWorkflow})
			continue
		}

		removed = append(removed, label)
		results = append(results, LabelChangeResult{Label: label, Status: LabelChangeRemoved})
	}

	if len(added) == 0 && len(removed) == 0 {
		return results, nil, fmt.Errorf("no label added or removed")
	}

	labelOp := NewLabelChangeOperation(author, unixTime, added, removed)

	if err := labelOp.Validate(); err != nil {
		return nil, nil, err
	}

	b.Append(labelOp)

	// If we're changing workflows then force the status to the initial state
	if newWorkflow != nil {
		op := NewSetStatusOp(author, unixTime, newWorkflow.initialState)
		if err := op.Validate(); err != nil {
			return nil, nil, err
		}
		b.Append(op)
	}

	return results, labelOp, nil
}

// ForceChangeLabels is a convenience function to apply the operation
// The difference with ChangeLabels is that no checks of deduplications are done. You are entirely
// responsible of what you are doing. In the general case, you want to use ChangeLabels instead.
// The intended use of this function is to allow importers to create legal but unexpected label changes,
// like removing a label with no information of when it was added before.
func ForceChangeLabels(b Interface, author identity.Interface, unixTime int64, add, remove []string) (*LabelChangeOperation, error) {
	added := make([]Label, len(add))
	for i, str := range add {
		added[i] = Label(str)
	}

	removed := make([]Label, len(remove))
	for i, str := range remove {
		removed[i] = Label(str)
	}

	labelOp := NewLabelChangeOperation(author, unixTime, added, removed)

	if err := labelOp.Validate(); err != nil {
		return nil, err
	}

	b.Append(labelOp)

	return labelOp, nil
}

func labelExist(labels []Label, label Label) bool {
	for _, l := range labels {
		if l == label {
			return true
		}
	}

	return false
}

type LabelChangeStatus int

const (
	_ LabelChangeStatus = iota
	LabelChangeAdded
	LabelChangeRemoved
	LabelChangeDuplicateInOp
	LabelChangeAlreadySet
	LabelChangeDoesntExist
	LabelChangeInvalidWorkflow
)

type LabelChangeResult struct {
	Label  Label
	Status LabelChangeStatus
}

func (l LabelChangeResult) String() string {
	switch l.Status {
	case LabelChangeAdded:
		return fmt.Sprintf("label %s added", l.Label)
	case LabelChangeRemoved:
		return fmt.Sprintf("label %s removed", l.Label)
	case LabelChangeDuplicateInOp:
		return fmt.Sprintf("label %s is a duplicate", l.Label)
	case LabelChangeAlreadySet:
		return fmt.Sprintf("label %s was already set", l.Label)
	case LabelChangeDoesntExist:
		return fmt.Sprintf("label %s doesn't exist on this bug", l.Label)
	case LabelChangeInvalidWorkflow:
		return fmt.Sprintf("invalid workflow operation %s", l.Label)
	default:
		panic(fmt.Sprintf("unknown label change status %v", l.Status))
	}
}
