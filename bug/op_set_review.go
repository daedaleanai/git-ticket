package bug

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	termtext "github.com/MichaelMure/go-term-text"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/util/timestamp"
)

var _ Operation = &SetReviewOperation{}

// SetReviewOperation will update the review associated with a ticket
type SetReviewOperation struct {
	OpBase
	Review ReviewInfo `json:"review"`
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

	// Create a map of timeline items to changes, we'll assume that all changes that
	// happened at the same time were done by the same person
	timelineMap := make(map[int64]*SetReviewTimelineItem)

	for _, u := range op.Review.Updates {
		if tl, exists := timelineMap[u.Timestamp]; exists {
			// Not the first change at this timestamp, update the one in the map
			tl.Review.Updates = append(tl.Review.Updates, u)
			timelineMap[u.Timestamp] = tl
		} else {
			// First one, create a new timeline item using the update author and timestamp
			item := &SetReviewTimelineItem{
				id:       op.Id(),
				Author:   u.Author,
				UnixTime: timestamp.Timestamp(u.Timestamp),
				Review: ReviewInfo{
					RevisionId: op.Review.RevisionId,
					Updates:    []ReviewUpdate{u}},
			}
			timelineMap[u.Timestamp] = item
		}
	}

	// Add all the timeline items to the snapshot, finally sort them
	for _, tl := range timelineMap {
		snapshot.Timeline = append(snapshot.Timeline, tl)
	}
	sort.Slice(snapshot.Timeline, func(i, j int) bool {
		return snapshot.Timeline[i].When() < snapshot.Timeline[j].When()
	})
}

// removeFromTimeline prunes entries from the timeline have the same revision id as this operation
func (op *SetReviewOperation) removeFromTimeline(snapshot *Snapshot) {
	var newTimeline []TimelineItem

	for _, tl := range snapshot.Timeline {
		if rtl, isRtl := tl.(*SetReviewTimelineItem); !isRtl || rtl.Review.RevisionId != op.Review.RevisionId {
			newTimeline = append(newTimeline, tl)
		}
	}

	snapshot.Timeline = newTimeline
}

func (op *SetReviewOperation) Apply(snapshot *Snapshot) {

	if op.Review.LastTransaction == RemoveReviewInfo {
		// This review has been removed from the ticket
		delete(snapshot.Reviews, op.Review.RevisionId)

		op.removeFromTimeline(snapshot)
	} else {
		// Update the review data, if it's not already there an empty ReviewInfo
		// struct will be returned
		r, _ := snapshot.Reviews[op.Review.RevisionId]
		r.RevisionId = op.Review.RevisionId
		r.Title = op.Review.Title
		r.LastTransaction = op.Review.LastTransaction
		r.Updates = append(r.Updates, op.Review.Updates...)
		snapshot.Reviews[op.Review.RevisionId] = r

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

	aux := struct {
		Review ReviewInfo `json:"review"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Review = aux.Review

	return nil
}

// Sign post method for gqlgen
func (op *SetReviewOperation) IsAuthored() {}

func NewSetReviewOp(author identity.Interface, unixTime int64, review *ReviewInfo) *SetReviewOperation {
	return &SetReviewOperation{
		OpBase: newOpBase(SetReviewOp, author, unixTime),
		Review: *review,
	}
}

type SetReviewTimelineItem struct {
	id       entity.Id
	Author   identity.Interface
	UnixTime timestamp.Timestamp
	Review   ReviewInfo
}

func (s SetReviewTimelineItem) Id() entity.Id {
	return s.id
}

func (s SetReviewTimelineItem) When() timestamp.Timestamp {
	return s.UnixTime
}

func (s SetReviewTimelineItem) String() string {
	var output strings.Builder
	var comments int

	for _, u := range s.Review.Updates {
		switch u.Type {
		case UserStatusTransaction:
			output.WriteString("[" + u.Status + "] ")
		case DiffTransaction:
			output.WriteString("[diff>" + strconv.Itoa(u.DiffId) + "] ")
		case CommentTransaction:
			comments = comments + 1
		}
	}

	if comments > 1 {
		output.WriteString("[" + strconv.Itoa(comments) + " comments] ")
	} else if comments > 0 {
		output.WriteString("[1 comment] ")
	}

	return fmt.Sprintf("(%s) %s: updated revision %s %s",
		s.UnixTime.Time().Format("2006-01-02 15:04:05"),
		termtext.LeftPadMaxLine(s.Author.DisplayName(), 15, 0),
		s.Review.RevisionId,
		output.String())
}

// Sign post method for gqlgen
func (s *SetReviewTimelineItem) IsAuthored() {}

// Convenience function to apply the operation
func SetReview(b Interface, author identity.Interface, unixTime int64, review *ReviewInfo) (*SetReviewOperation, error) {
	setReviewOp := NewSetReviewOp(author, unixTime, review)

	if err := setReviewOp.Validate(); err != nil {
		return nil, err
	}

	b.Append(setReviewOp)
	return setReviewOp, nil
}
