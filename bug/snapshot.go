package bug

import (
	"fmt"
	"time"

	"github.com/daedaleanai/git-ticket/bug/review"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/pkg/errors"
)

// Snapshot is a compiled form of the Bug data structure used for storage and merge
type Snapshot struct {
	id entity.Id

	Status       Status
	Title        string
	Comments     []Comment
	Labels       []Label
	Checklists   map[Label]map[entity.Id]ChecklistSnapshot // label and reviewer id
	Reviews      map[string]review.PullRequest             // Pull request ID
	Author       identity.Interface
	Assignee     identity.Interface
	Actors       []identity.Interface
	Participants []identity.Interface
	Ccb          []CcbInfo
	CreateTime   time.Time

	Timeline []TimelineItem

	Operations []Operation
}

// Return the Bug identifier
func (snap *Snapshot) Id() entity.Id {
	return snap.id
}

// Return the last time a bug was modified
func (snap *Snapshot) EditTime() time.Time {
	if len(snap.Operations) == 0 {
		return time.Unix(0, 0)
	}

	return snap.Operations[len(snap.Operations)-1].Time()
}

// GetCreateMetadata return the creation metadata
func (snap *Snapshot) GetCreateMetadata(key string) (string, bool) {
	return snap.Operations[0].GetMetadata(key)
}

// SearchComment will search for a comment matching the given hash
func (snap *Snapshot) SearchComment(id entity.Id) (*Comment, error) {
	for _, c := range snap.Comments {
		if c.id == id {
			return &c, nil
		}
	}

	return nil, fmt.Errorf("comment item not found")
}

// GetComment will return the comment for a given index
func (snap *Snapshot) GetComment(index int) (*Comment, error) {
	if index < len(snap.Comments) {
		return &snap.Comments[index], nil
	}

	return nil, fmt.Errorf("comment item not found")
}

// append the operation author to the actors list
func (snap *Snapshot) addActor(actor identity.Interface) {
	for _, a := range snap.Actors {
		if actor.Id() == a.Id() {
			return
		}
	}

	snap.Actors = append(snap.Actors, actor)
}

// append the operation author to the participants list
func (snap *Snapshot) addParticipant(participant identity.Interface) {
	for _, p := range snap.Participants {
		if participant.Id() == p.Id() {
			return
		}
	}

	snap.Participants = append(snap.Participants, participant)
}

// HasParticipant return true if the id is a participant
func (snap *Snapshot) HasParticipant(id entity.Id) bool {
	for _, p := range snap.Participants {
		if p.Id() == id {
			return true
		}
	}
	return false
}

// HasAnyParticipant return true if one of the ids is a participant
func (snap *Snapshot) HasAnyParticipant(ids ...entity.Id) bool {
	for _, id := range ids {
		if snap.HasParticipant(id) {
			return true
		}
	}
	return false
}

// HasActor return true if the id is a actor
func (snap *Snapshot) HasActor(id entity.Id) bool {
	for _, p := range snap.Actors {
		if p.Id() == id {
			return true
		}
	}
	return false
}

// HasAnyActor return true if one of the ids is a actor
func (snap *Snapshot) HasAnyActor(ids ...entity.Id) bool {
	for _, id := range ids {
		if snap.HasActor(id) {
			return true
		}
	}
	return false
}

// GetCcbState returns the state assocated with the id in the ticket CCB group
func (snap *Snapshot) GetCcbState(id entity.Id, status Status) CcbState {
	for _, c := range snap.Ccb {
		if c.User.Id() == id && c.Status == status {
			return c.State
		}
	}
	return RemovedCcbState
}

// Sign post method for gqlgen
func (snap *Snapshot) IsAuthored() {}

// GetUserChecklists returns a map of checklists associated with this snapshot for the given reviewer id,
// if the blank flag is set then always return a clean set of checklists
func (snap *Snapshot) GetUserChecklists(reviewer entity.Id, blank bool) (map[Label]Checklist, error) {
	checklists := make(map[Label]Checklist)

	// Only checklists named in the labels list are currently valid
	for _, l := range snap.Labels {
		if l.IsChecklist() {
			if snapshotChecklist, present := snap.Checklists[l][reviewer]; !blank && present {
				checklists[l] = snapshotChecklist.Checklist
			} else {
				var err error
				checklists[l], err = GetChecklist(l)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return checklists, nil
}

// GetChecklistCompoundStates returns a map of checklist states mapped to label, associated with this snapshot
func (snap *Snapshot) GetChecklistCompoundStates() map[Label]ChecklistState {
	states := make(map[Label]ChecklistState)

	// Only checklists named in the labels list are currently valid
	for _, l := range snap.Labels {
		if l.IsChecklist() {
			// default state is TBD
			states[l] = TBD

			clMap, present := snap.Checklists[l]
			if present {
				// at least one user has edited this checklist
			ReviewsLoop:
				for _, cl := range clMap {
					clState := cl.CompoundState()
					switch clState {
					case Failed:
						// someone failed it, it's failed
						states[l] = Failed
						break ReviewsLoop
					case Passed:
						// someone passed it, and no-one failed it yet
						states[l] = Passed
					}
				}
			}
		}
	}
	return states
}

// NextStatuses returns a slice of next possible statuses for the assigned workflow
func (snap *Snapshot) NextStatuses() ([]Status, error) {
	w := FindWorkflow(snap.Labels)
	if w == nil {
		return nil, fmt.Errorf("ticket has no associated workflow")
	}
	return w.NextStatuses(snap.Status), nil
}

// ValidateTransition returns an error if the supplied state is an invalid
// destination from the current state for the assigned workflow
func (snap *Snapshot) ValidateTransition(newStatus Status) error {
	w := FindWorkflow(snap.Labels)
	if w == nil {
		return fmt.Errorf("ticket has no associated workflow")
	}
	return w.ValidateTransition(snap, newStatus)
}

// ValidateAssigneeSet returns an error if the snapshot assignee is not set
func ValidateAssigneeSet(snap *Snapshot, next Status) error {
	if snap.Assignee == nil {
		return errors.New("assignee not set")
	}
	return nil
}

// ValidateCcb returns an error if the snapshot does not have CCB set and approved for the next status
func ValidateCcb(snap *Snapshot, next Status) error {
	var ccbAssigned int
	// Loop through the entire CCB list, each entry represents an approval: a ticket status plus
	// a CCB member who should approve it
	for _, approval := range snap.Ccb {
		if approval.Status == next {
			// This approval is needed for the requested 'next' status
			ccbAssigned++
			if approval.State != ApprovedCcbState {
				return fmt.Errorf("not all CCB have approved ticket status %s", next)
			}
		}
	}
	// Check at least one approval is associated with the requested status
	if ccbAssigned == 0 {
		return fmt.Errorf("no CCB assigned to ticket status %s", next)
	}
	return nil
}

// ValidateChecklistsCompleted returns an error if at least one of the checklists attached to the snapshot
// has not been completed
func ValidateChecklistsCompleted(snap *Snapshot, next Status) error {
	for _, st := range snap.GetChecklistCompoundStates() {
		if st == TBD {
			return errors.New("at least one checklist still TBD")
		}
	}
	return nil
}
