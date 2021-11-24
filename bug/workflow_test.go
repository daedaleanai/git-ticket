package bug

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testWorkflow = Workflow{label: "workflow:test",
	initialState: ProposedStatus,
	transitions: []Transition{
		Transition{start: ProposedStatus, end: VettedStatus, hook: "echo TEST transitioning from proposed to vetted"},
		Transition{start: VettedStatus, end: ProposedStatus},
		Transition{start: VettedStatus, end: InProgressStatus},
		Transition{start: InProgressStatus, end: InReviewStatus},
		Transition{start: InReviewStatus, end: InProgressStatus, hook: "true"},
		Transition{start: InReviewStatus, end: ReviewedStatus},
		Transition{start: ReviewedStatus, end: AcceptedStatus},
		Transition{start: AcceptedStatus, end: MergedStatus},
		Transition{start: MergedStatus, end: AcceptedStatus, hook: "false"},
		Transition{start: MergedStatus, end: DoneStatus},
	},
}

func TestWorkflow_FindWorkflow(t *testing.T) {
	if wf := FindWorkflow("workflow:eng"); wf == nil || wf.label != "workflow:eng" {
		t.Fatal("Finding workflow:eng failed")
	}

	if wf := FindWorkflow("workflow:qa"); wf == nil || wf.label != "workflow:qa" {
		t.Fatal("Finding workflow:qa failed")
	}

	if FindWorkflow("workflow:XYZGASH") != nil {
		t.Fatal("FindWorkflow returned reference to non-existant workflow")
	}
}

func TestWorkflow_NextStatuses(t *testing.T) {
	// The valid next statuses for each status in the testWorkflow
	var nextStatuses = [][]Status{
		nil,                                // first status is 1
		{VettedStatus},                     // from ProposedStatus
		{ProposedStatus, InProgressStatus}, // from VettedStatus
		{InReviewStatus},                   // from InProgressStatus
		{InProgressStatus, ReviewedStatus}, // from InReviewStatus
		{AcceptedStatus},                   // from ReviewedStatus
		{MergedStatus},                     // from AcceptedStatus
		{AcceptedStatus, DoneStatus},       // from MergedStatus
		nil,                                // from DoneStatus
	}

	for currentStatus := FirstStatus; currentStatus <= LastStatus; currentStatus++ {
		next, err := testWorkflow.NextStatuses(currentStatus)
		if err != nil {
			t.Fatal("Invalid next statuses", currentStatus, ">", next, "(error", err, ")")
		}
		assert.Equal(t, nextStatuses[currentStatus], next)
	}
}

func TestWorkflow_ValidateTransition(t *testing.T) {
	// The valid transitions for each status in the testWorkflow
	var validTransitions = [][]Status{
		nil,                                // first status is 1
		{VettedStatus},                     // from ProposedStatus
		{ProposedStatus, InProgressStatus}, // from VettedStatus
		{InReviewStatus},                   // from InProgressStatus
		{InProgressStatus, ReviewedStatus}, // from InReviewStatus
		{AcceptedStatus},                   // from ReviewedStatus
		{MergedStatus},                     // from AcceptedStatus
		{DoneStatus},                       // from MergedStatus
		nil,                                // from DoneStatus
	}

	// Test validation of state transition
	for from := FirstStatus; from <= LastStatus; from++ {
		for _, to := range validTransitions[from] {
			if err := testWorkflow.ValidateTransition(from, to); err != nil {
				t.Fatal("State transition " + from.String() + " > " + to.String() + " flagged invalid when it isn't")
			}
		}
	}

	if err := testWorkflow.ValidateTransition(ProposedStatus, MergedStatus); err == nil {
		t.Fatal("State transition proposed > merged flagged valid when it isn't")
	}
}
