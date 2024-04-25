package bug

import (
	"fmt"

	"github.com/daedaleanai/git-ticket/identity"
)

// Invoked to validate if a workflow transition can be taken. Returns an error if the transition is invalid.
type ValidationFunc func(snap *Snapshot, next Status) error

// Invoked to update a ticket as a consequence of a workflow status transition.
type ActionFunc func(b Interface, snapshot *Snapshot, next Status, author identity.Interface, unixTime int64) error

type Transition struct {
	start          Status
	end            Status
	validationHook []ValidationFunc
	actionHook     []ActionFunc
}

type Workflow struct {
	label        Label
	initialState Status
	transitions  []Transition
}

var workflowStore []Workflow

// FindWorkflow searches a list of labels and attempts to match them to a workflow, returning the first found
func FindWorkflow(names []Label) *Workflow {
	for _, l := range names {
		if l.IsWorkflow() {
			for wf := range workflowStore {
				if workflowStore[wf].label == l {
					return &workflowStore[wf]
				}
			}
		}
	}
	return nil
}

// GetWorkflowLabels returns a slice of all the available workflow labels
func GetWorkflowLabels() []Label {
	var labels []Label
	for _, wf := range workflowStore {
		labels = append(labels, wf.label)
	}
	return labels
}

// NextStatuses returns a slice of next possible statuses in the workflow
// for the given one
func (w *Workflow) NextStatuses(s Status) []Status {
	var validStatuses []Status
	for _, t := range w.transitions {
		if t.start == s {
			validStatuses = append(validStatuses, t.end)
		}
	}
	return validStatuses
}

// ValidateTransition checks if the transition is valid for a given start and end
func (w *Workflow) ValidateTransition(snap *Snapshot, to Status) error {
	for _, t := range w.transitions {
		if t.start == snap.Status && t.end == to {
			if t.validationHook != nil {
				for _, v := range t.validationHook {
					if err := v(snap, to); err != nil {
						return err
					}
				}
			}
			return nil
		}
	}

	// invalid transition, return error with list of valid transitions
	nextStatuses := w.NextStatuses(snap.Status)
	return fmt.Errorf("invalid transition %s->%s, possible next statuses: %s", snap.Status, to, nextStatuses)
}

// ApplyTransitionActions invokes the actionHooks of the transition that was taken
func (w *Workflow) ApplyTransitionActions(b Interface, snap *Snapshot, to Status, author identity.Interface, unixTime int64) error {
	for _, t := range w.transitions {
		if t.start == snap.Status && t.end == to {
			if t.actionHook != nil {
				for _, a := range t.actionHook {
					if err := a(b, snap, to, author, unixTime); err != nil {
						return err
					}
				}
			}
			return nil
		}
	}

	return nil
}

func init() {
	// Initialise list of workflows
	workflowStore = []Workflow{
		{label: "workflow:eng",
			initialState: ProposedStatus,
			transitions: []Transition{
				{start: ProposedStatus, end: VettedStatus,
					validationHook: []ValidationFunc{ValidateCcb}},
				{start: ProposedStatus, end: RejectedStatus, actionHook: []ActionFunc{
					ClearAllCcbApprovals}},
				{start: VettedStatus, end: InProgressStatus,
					validationHook: []ValidationFunc{ValidateAssigneeSet}},
				{start: VettedStatus, end: RejectedStatus,
					validationHook: []ValidationFunc{ValidateCcb}, actionHook: []ActionFunc{ClearAllCcbApprovals}},
				{start: InProgressStatus, end: VettedStatus},
				{start: InProgressStatus, end: InReviewStatus},
				{start: InProgressStatus, end: RejectedStatus,
					validationHook: []ValidationFunc{ValidateCcb}, actionHook: []ActionFunc{ClearAllCcbApprovals}},
				{start: InReviewStatus, end: InProgressStatus},
				{start: InReviewStatus, end: ReviewedStatus},
				{start: InReviewStatus, end: RejectedStatus,
					validationHook: []ValidationFunc{ValidateCcb}, actionHook: []ActionFunc{ClearAllCcbApprovals}},
				{start: ReviewedStatus, end: InProgressStatus},
				{start: ReviewedStatus, end: AcceptedStatus,
					validationHook: []ValidationFunc{ValidateCcb,
						ValidateChecklistsCompleted}},
				{start: ReviewedStatus, end: RejectedStatus,
					validationHook: []ValidationFunc{ValidateCcb}, actionHook: []ActionFunc{ClearAllCcbApprovals}},
				{start: AcceptedStatus, end: MergedStatus},
				{start: AcceptedStatus, end: RejectedStatus,
					validationHook: []ValidationFunc{ValidateCcb}, actionHook: []ActionFunc{ClearAllCcbApprovals}},
				{start: MergedStatus, end: AcceptedStatus},
				{start: RejectedStatus, end: ProposedStatus},
			},
		},
		{label: "workflow:qa",
			initialState: ProposedStatus,
			transitions: []Transition{
				{start: ProposedStatus, end: InProgressStatus,
					validationHook: []ValidationFunc{ValidateAssigneeSet}},
				{start: ProposedStatus, end: RejectedStatus},
				{start: InProgressStatus, end: DoneStatus},
				{start: InProgressStatus, end: RejectedStatus},
				{start: DoneStatus, end: InProgressStatus},
				{start: RejectedStatus, end: ProposedStatus},
			},
		},
		{label: "workflow:change",
			initialState: ProposedStatus,
			transitions: []Transition{
				{start: ProposedStatus, end: InProgressStatus,
					validationHook: []ValidationFunc{ValidateAssigneeSet}},
				{start: ProposedStatus, end: RejectedStatus},
				{start: InProgressStatus, end: DoneStatus},
				{start: InProgressStatus, end: RejectedStatus},
				{start: DoneStatus, end: InProgressStatus},
				{start: RejectedStatus, end: ProposedStatus},
			},
		},
	}
}
