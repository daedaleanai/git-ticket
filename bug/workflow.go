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

// NextStates returns a slice of next possible states in the workflow
// for the given one
func (w *Workflow) NextStates(s Status) ([]Status, error) {
	var validStates []Status
	for _, t := range w.transitions {
		if t.start == s {
			validStates = append(validStates, t.end)
		}
	}
	return validStates, nil
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
	nextStates, _ := w.NextStates(from)
	return fmt.Errorf("invalid transition %s->%s, possible next states: %s", from, to, nextStates)
}

func init() {
	// Initialise list of workflows
	workflowStore = []Workflow{
		Workflow{label: "workflow:eng",
			initialState: ProposedStatus,
			transitions: []Transition{
				Transition{start: ProposedStatus, end: VettedStatus},
				Transition{start: VettedStatus, end: ProposedStatus},
				Transition{start: VettedStatus, end: InProgressStatus},
				Transition{start: InProgressStatus, end: InReviewStatus},
				Transition{start: InReviewStatus, end: InProgressStatus},
				Transition{start: InReviewStatus, end: ReviewedStatus},
				Transition{start: ReviewedStatus, end: AcceptedStatus},
				Transition{start: AcceptedStatus, end: MergedStatus},
			},
		},
		Workflow{label: "workflow:qa",
			initialState: ProposedStatus,
			transitions: []Transition{
				Transition{start: ProposedStatus, end: InProgressStatus},
				Transition{start: InProgressStatus, end: DoneStatus},
			},
		},
	}
}
