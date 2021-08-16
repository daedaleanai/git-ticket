package bug

import (
	"fmt"
	"time"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
)

// Snapshot is a compiled form of the Bug data structure used for storage and merge
type Snapshot struct {
	id entity.Id

	Status       Status
	Title        string
	Comments     []Comment
	Labels       []Label
	Checklists   map[Label]map[entity.Id]ChecklistSnapshot // label and reviewer id
	Reviews      map[string]ReviewInfo                     // Phabricator Differential ID
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

// SearchTimelineItem will search in the timeline for an item matching the given hash
func (snap *Snapshot) SearchTimelineItem(id entity.Id) (TimelineItem, error) {
	for i := range snap.Timeline {
		if snap.Timeline[i].Id() == id {
			return snap.Timeline[i], nil
		}
	}

	return nil, fmt.Errorf("timeline item not found")
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
func (snap *Snapshot) GetCcbState(id entity.Id) CcbState {
	for _, c := range snap.Ccb {
		if c.User.Id() == id {
			return c.State
		}
	}
	return RemovedCcbState
}

// CheckCcbApproved returns an error if the CCB group is not large enough or not all have approved the ticket
func (snap *Snapshot) CheckCcbApproved() error {
	if len(snap.Ccb) < 2 {
		return fmt.Errorf("ticket has insufficient CCB group (min: 2)")
	}
	for _, c := range snap.Ccb {
		if c.State != ApprovedCcbState {
			return fmt.Errorf("not all CCB group have approved ticket")
		}
	}
	return nil
}

// Sign post method for gqlgen
func (snap *Snapshot) IsAuthored() {}

// GetUserChecklists returns a map of checklists associated with this snapshot for the given reviewer id
func (snap *Snapshot) GetUserChecklists(reviewer entity.Id) (map[Label]Checklist, error) {
	checklists := make(map[Label]Checklist)

	// Only checklists named in the labels list are currently valid
	for _, l := range snap.Labels {
		if l.IsChecklist() {
			if snapshotChecklist, present := snap.Checklists[l][reviewer]; present {
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

// NextStates returns a slice of next possible states for the assigned workflow
func (snap *Snapshot) NextStates() ([]Status, error) {
	for _, l := range snap.Labels {
		if l.IsWorkflow() {
			w := FindWorkflow(l)
			if w == nil {
				return nil, fmt.Errorf("invalid workflow %s", l)
			}
			return w.NextStates(snap.Status)
		}
	}
	return nil, fmt.Errorf("ticket has no associated workflow")
}

// ValidateTransition returns an error if the supplied state is an invalid
// destination from the current state for the assigned workflow
func (snap *Snapshot) ValidateTransition(newStatus Status) error {
	for _, l := range snap.Labels {
		if l.IsWorkflow() {
			w := FindWorkflow(l)
			if w == nil {
				return fmt.Errorf("invalid workflow %s", l)
			}
			return w.ValidateTransition(snap.Status, newStatus)
		}
	}
	return fmt.Errorf("ticket has no associated workflow")
}
