package bug

import (
	"fmt"
)

type ValidationFunc func(snap *Snapshot, next Status) error

type Transition struct {
	start Status
	end   Status
	hook  []ValidationFunc
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
			if t.hook != nil {
				for _, v := range t.hook {
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

func init() {
	// Initialise list of workflows
	workflowStore = []Workflow{
		{label: "workflow:eng",
			initialState: ProposedStatus,
			transitions: []Transition{
				{start: ProposedStatus, end: VettedStatus,
					hook: []ValidationFunc{ValidateCcb}},
				{start: ProposedStatus, end: RejectedStatus},
				{start: VettedStatus, end: InProgressStatus,
					hook: []ValidationFunc{ValidateAssigneeSet}},
				{start: VettedStatus, end: RejectedStatus},
				{start: InProgressStatus, end: VettedStatus},
				{start: InProgressStatus, end: InReviewStatus},
				{start: InProgressStatus, end: RejectedStatus},
				{start: InReviewStatus, end: InProgressStatus},
				{start: InReviewStatus, end: ReviewedStatus},
				{start: InReviewStatus, end: RejectedStatus},
				{start: ReviewedStatus, end: InProgressStatus},
				{start: ReviewedStatus, end: AcceptedStatus,
					hook: []ValidationFunc{ValidateCcb,
						ValidateChecklistsCompleted}},
				{start: ReviewedStatus, end: RejectedStatus},
				{start: AcceptedStatus, end: MergedStatus},
				{start: AcceptedStatus, end: RejectedStatus},
				{start: MergedStatus, end: AcceptedStatus},
				{start: RejectedStatus, end: ProposedStatus},
			},
		},
		{label: "workflow:qa",
			initialState: ProposedStatus,
			transitions: []Transition{
				{start: ProposedStatus, end: InProgressStatus},
				{start: ProposedStatus, end: RejectedStatus},
				{start: InProgressStatus, end: DoneStatus},
				{start: InProgressStatus, end: RejectedStatus},
				{start: DoneStatus, end: InProgressStatus},
				{start: RejectedStatus, end: ProposedStatus},
			},
		},
	}
}
