package bug

import (
	"encoding/json"
	"fmt"
	"sort"

	review2 "github.com/daedaleanai/git-ticket/bug/review"

	termtext "github.com/MichaelMure/go-term-text"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/util/timestamp"
)

var _ Operation = &SetReviewOperation{}

// SetReviewOperation will update the review associated with a ticket
type SetReviewOperation struct {
	OpBase
	Review review2.PullRequest `json:"review"`
}

//Sign-post method for gqlgen
func (op *SetReviewOperation) IsOperation() {}

func (op *SetReviewOperation) base() *OpBase {
	return &op.OpBase
}

func (op *SetReviewOperation) Id() entity.Id {
	return idOperation(op)
}

// addToTimeline takes the current operation and splits it into timeline entries
// which represent actual changes made in the review process
func (op *SetReviewOperation) addToTimeline(snapshot *Snapshot) {
	// Add all the timeline items to the snapshot, finally sort them
	for _, tl := range op.Review.History() {
		snapshot.Timeline = append(snapshot.Timeline, &SetReviewTimelineItem{
			id:       op.Id(),
			Author:   tl.Author(),
			UnixTime: tl.Timestamp(),
			Review:   op.Review,
			Event:    tl,
		})
	}

	sort.Slice(snapshot.Timeline, func(i, j int) bool {
		return snapshot.Timeline[i].When() < snapshot.Timeline[j].When()
	})
}

// removeFromTimeline prunes entries from the timeline have the same revision id as this operation
func (op *SetReviewOperation) removeFromTimeline(snapshot *Snapshot) {
	var newTimeline []TimelineItem

	for _, tl := range snapshot.Timeline {
		if rtl, isRtl := tl.(*SetReviewTimelineItem); !isRtl || rtl.Review.Id() != op.Review.Id() {
			newTimeline = append(newTimeline, tl)
		}
	}

	snapshot.Timeline = newTimeline
}

func (op *SetReviewOperation) Apply(snapshot *Snapshot) {

	if _, ok := op.Review.(*review2.RemoveReview); ok {
		// This review has been removed from the ticket
		delete(snapshot.Reviews, op.Review.Id())

		op.removeFromTimeline(snapshot)
	} else {
		// Update the review data, if it's not already there an empty ReviewInfo
		// struct will be returned
		r, _ := snapshot.Reviews[op.Review.Id()]
		if r == nil {
			r = op.Review
		} else {
			r.Merge(op.Review)
		}
		snapshot.Reviews[op.Review.Id()] = r

		op.addToTimeline(snapshot)
	}

	snapshot.addActor(op.Author)
}

func (op *SetReviewOperation) Validate() error {
	if err := opBaseValidate(op, SetReviewOp); err != nil {
		return err
	}

	return nil
}

// MarshalJSON serializes to json preserving type information
func (op *SetReviewOperation) MarshalJSON() ([]byte, error) {
	wrapper := struct {
		OpBase
		Phabricator *review2.PhabReviewInfo `json:"review"`
		Gitea       *review2.GiteaInfo      `json:"reviewGitea"`
		Remove      *review2.RemoveReview   `json:"removeReview"`
	}{}
	wrapper.OpBase = op.OpBase
	if phab, ok := op.Review.(*review2.PhabReviewInfo); ok {
		wrapper.Phabricator = phab
	} else if gitea, ok := op.Review.(*review2.GiteaInfo); ok {
		wrapper.Gitea = gitea
	} else if remove, ok := op.Review.(*review2.RemoveReview); ok {
		wrapper.Remove = remove
	} else {
		panic("Unknown review info")
	}
	return json.Marshal(wrapper)
}

// UnmarshalJSON is a two step JSON unmarshaling
// This workaround is necessary to avoid the inner OpBase.MarshalJSON
// overriding the outer op's MarshalJSON
func (op *SetReviewOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	wrapper := struct {
		Phabricator *review2.PhabReviewInfo `json:"review"`
		Gitea       *review2.GiteaInfo      `json:"reviewGitea"`
		Remove      *review2.RemoveReview   `json:"removeReview"`
	}{}

	err = json.Unmarshal(data, &wrapper)
	if err != nil {
		return err
	}
	op.OpBase = base
	if wrapper.Phabricator != nil {
		op.Review = wrapper.Phabricator
	} else if wrapper.Gitea != nil {
		op.Review = wrapper.Gitea
	} else if wrapper.Remove != nil {
		op.Review = wrapper.Remove
	} else {
		return fmt.Errorf("Unable to parse review info")
	}
	return nil
}

// Sign post method for gqlgen
func (op *SetReviewOperation) IsAuthored() {}

func NewSetReviewOp(author identity.Interface, unixTime int64, review review2.PullRequest) *SetReviewOperation {
	return &SetReviewOperation{
		OpBase: newOpBase(SetReviewOp, author, unixTime),
		Review: review,
	}
}

type SetReviewTimelineItem struct {
	id       entity.Id
	Author   identity.Interface
	UnixTime timestamp.Timestamp
	Review   review2.PullRequest
	Event    review2.TimelineEvent
}

func (s SetReviewTimelineItem) Id() entity.Id {
	return s.id
}

func (s SetReviewTimelineItem) When() timestamp.Timestamp {
	return s.UnixTime
}

func (s SetReviewTimelineItem) String() string {
	return fmt.Sprintf("(%s) %s: updated %s %s",
		s.UnixTime.Time().Format("2006-01-02 15:04:05"),
		termtext.LeftPadMaxLine(s.Author.DisplayName(), timelineDisplayNameWidth, 0),
		s.Review.Id(),
		s.Event.Summary())
}

// Sign post method for gqlgen
func (s *SetReviewTimelineItem) IsAuthored() {}

// Convenience function to apply the operation
func SetReview(b Interface, author identity.Interface, unixTime int64, review review2.PullRequest) (*SetReviewOperation, error) {
	setReviewOp := NewSetReviewOp(author, unixTime, review)

	if err := setReviewOp.Validate(); err != nil {
		return nil, err
	}

	b.Append(setReviewOp)
	return setReviewOp, nil
}
