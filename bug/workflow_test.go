package bug

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testWorkflow = Workflow{label: "workflow:test",
	initialState: ProposedStatus,
	transitions: []Transition{
		{start: ProposedStatus, end: VettedStatus,
			validationHook: []ValidationFunc{func(snap *Snapshot, next Status) error { return nil }}},
		{start: VettedStatus, end: ProposedStatus},
		{start: VettedStatus, end: InProgressStatus},
		{start: InProgressStatus, end: InReviewStatus},
		{start: InReviewStatus, end: InProgressStatus,
			validationHook: []ValidationFunc{func(snap *Snapshot, next Status) error { return nil }}},
		{start: InReviewStatus, end: ReviewedStatus},
		{start: ReviewedStatus, end: AcceptedStatus},
		{start: AcceptedStatus, end: MergedStatus},
		{start: MergedStatus, end: AcceptedStatus,
			validationHook: []ValidationFunc{func(snap *Snapshot, next Status) error { return errors.New("epic fail") }}},
		{start: MergedStatus, end: DoneStatus},
	},
}

func TestWorkflow_FindWorkflow(t *testing.T) {
	if wf := FindWorkflow([]Label{"workflow:eng"}); wf == nil || wf.label != "workflow:eng" {
		t.Fatal("Finding workflow:eng failed")
	}

	if wf := FindWorkflow([]Label{"workflow:qa"}); wf == nil || wf.label != "workflow:qa" {
		t.Fatal("Finding workflow:qa failed")
	}

	if wf := FindWorkflow([]Label{"workflow:change"}); wf == nil || wf.label != "workflow:change" {
		t.Fatal("Finding workflow:change failed")
	}

	if wf := FindWorkflow([]Label{"workflow:exp"}); wf == nil || wf.label != "workflow:exp" {
		t.Fatal("Finding workflow:exp failed")
	}

	if FindWorkflow([]Label{"workflow:XYZGASH"}) != nil {
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
		nil,                                // from RejectedStatus
	}

	for currentStatus := FirstStatus; currentStatus <= LastStatus; currentStatus++ {
		assert.Equal(t, nextStatuses[currentStatus], testWorkflow.NextStatuses(currentStatus))
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
		nil,                                // from RejectedStatus
	}

	var snap Snapshot

	// Test validation of state transition
	for from := FirstStatus; from <= LastStatus; from++ {
		snap.Status = from
		for _, to := range validTransitions[from] {
			if err := testWorkflow.ValidateTransition(&snap, to); err != nil {
				t.Fatal("State transition " + from.String() + " > " + to.String() + " flagged invalid when it isn't")
			}
		}
	}

	snap.Status = ProposedStatus
	if err := testWorkflow.ValidateTransition(&snap, MergedStatus); err == nil {
		t.Fatal("State transition proposed > merged flagged valid when it isn't")
	}
}
