package bug

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Transition struct {
	start Status
	end   Status
	hook  string
}

type Workflow struct {
	label        Label
	initialState Status
	transitions  []Transition
}

var workflowStore []Workflow

// FindWorkflow searches the workflow store by name and returns a pointer to it
func FindWorkflow(name Label) *Workflow {
	for wf := range workflowStore {
		if workflowStore[wf].label == name {
			return &workflowStore[wf]
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
func (w *Workflow) NextStatuses(s Status) ([]Status, error) {
	var validStatuses []Status
	for _, t := range w.transitions {
		if t.start == s {
			validStatuses = append(validStatuses, t.end)
		}
	}
	return validStatuses, nil
}

// ValidateTransition checks if the transition is valid for a given start and end
func (w *Workflow) ValidateTransition(from, to Status) error {
	for _, t := range w.transitions {
		if t.start == from && t.end == to {
			if t.hook != "" {
				hookArgs := strings.Split(t.hook, " ")
				cmd := exec.Command(hookArgs[0], hookArgs[1:]...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				return cmd.Run()
			}
			return nil
		}
	}

	// invalid transition, return error with list of valid transitions
	nextStatuses, _ := w.NextStatuses(from)
	return fmt.Errorf("invalid transition %s->%s, possible next statuses: %s", from, to, nextStatuses)
}

func init() {
	// Initialise list of workflows
	workflowStore = []Workflow{
		{label: "workflow:eng",
			initialState: ProposedStatus,
			transitions: []Transition{
				{start: ProposedStatus, end: VettedStatus},
				{start: ProposedStatus, end: RejectedStatus},
				{start: VettedStatus, end: ProposedStatus},
				{start: VettedStatus, end: InProgressStatus},
				{start: VettedStatus, end: RejectedStatus},
				{start: InProgressStatus, end: InReviewStatus},
				{start: InProgressStatus, end: RejectedStatus},
				{start: InReviewStatus, end: InProgressStatus},
				{start: InReviewStatus, end: ReviewedStatus},
				{start: InReviewStatus, end: RejectedStatus},
				{start: ReviewedStatus, end: AcceptedStatus},
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
			},
		},
	}
}
