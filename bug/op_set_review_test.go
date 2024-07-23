package bug

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/daedaleanai/git-ticket/bug/review"

	"github.com/daedaleanai/git-ticket/identity"
	"github.com/stretchr/testify/assert"
)

var rene = identity.NewBare("René Descarte", "rene@descartes.fr")

var testUpdates = []review.ReviewUpdate{
	{
		PhabTransaction: review.PhabTransaction{
			TransId:   "10000",
			PhabUser:  "USERID1",
			Timestamp: 0,
			Type:      review.StatusTransaction,
			Status:    "in progress"},
		AuthorId: rene,
	},
	{
		PhabTransaction: review.PhabTransaction{
			TransId:   "10005",
			PhabUser:  "USERID1",
			Timestamp: 5,
			Type:      review.StatusTransaction,
			Status:    "on review"},
		AuthorId: rene,
	},
	{
		PhabTransaction: review.PhabTransaction{
			TransId:   "10010",
			PhabUser:  "USERID1",
			Timestamp: 10,
			Type:      review.StatusTransaction,
			Status:    "complete"},
		AuthorId: rene,
	},
	{
		PhabTransaction: review.PhabTransaction{
			TransId:   "10001",
			PhabUser:  "USERID2",
			Timestamp: 1,
			Type:      review.CommentTransaction,
			Diff:      123,
			Path:      "code/under_test.go",
			Line:      1,
			Text:      "needs work"},
		AuthorId: rene,
	},
	{
		PhabTransaction: review.PhabTransaction{
			TransId:   "10002",
			PhabUser:  "USERID2",
			Timestamp: 2,
			Type:      review.CommentTransaction,
			Diff:      124,
			Path:      "code/under_test.go",
			Line:      1,
			Text:      "LGTM"},
		AuthorId: rene,
	},
}

func TestOpSetReview_SetReview(t *testing.T) {
	unix := time.Now().Unix()
	bug1 := NewBug()

	before, err := SetReview(bug1, rene, unix,
		&review.PhabReviewInfo{RevisionId: "D1234",
			LastTransaction: "12345",
			Updates:         testUpdates,
		})
	assert.NoError(t, err)

	data, err := json.Marshal(before)
	assert.NoError(t, err)

	var after SetReviewOperation
	err = json.Unmarshal(data, &after)
	assert.NoError(t, err)

	// enforce creating the IDs
	before.Id()
	rene.Id()

	assert.Equal(t, before, &after)
}

func TestOpSetReview_Apply(t *testing.T) {

	var rene = identity.NewBare("René Descarte", "rene@descartes.fr")
	unix := time.Now().Unix()
	snapshot := NewBug().Compile()

	// create an operation and apply to the snapshot
	setReviewOp := NewSetReviewOp(rene, unix, &review.PhabReviewInfo{RevisionId: "D1234",
		LastTransaction: "12345",
		Updates:         []review.ReviewUpdate{testUpdates[0]}})
	setReviewOp.Apply(&snapshot)

	// sumation holds a local copy of what should be in the snapshot
	sumation := &review.PhabReviewInfo{RevisionId: "D1234",
		LastTransaction: "12345",
		Updates:         []review.ReviewUpdate{testUpdates[0]},
	}

	assert.Equal(t, sumation, snapshot.Reviews["D1234"])

	// add an extra Update
	setReviewOp2 := NewSetReviewOp(rene, unix, &review.PhabReviewInfo{RevisionId: "D1234",
		LastTransaction: "12346",
		Updates:         []review.ReviewUpdate{testUpdates[1]}})
	setReviewOp2.Apply(&snapshot)

	sumation.Updates = append(sumation.Updates, testUpdates[1])
	sumation.LastTransaction = "12346"

	assert.Equal(t, sumation, snapshot.Reviews["D1234"])

	// and a couple more
	setReviewOp3 := NewSetReviewOp(rene, unix, &review.PhabReviewInfo{RevisionId: "D1234",
		LastTransaction: "12347",
		Updates:         testUpdates[1:2]})
	setReviewOp3.Apply(&snapshot)

	sumation.Updates = append(sumation.Updates, testUpdates[1:2]...)
	sumation.LastTransaction = "12347"

	assert.Equal(t, sumation, snapshot.Reviews["D1234"])

	// remove the review
	setReviewOp4 := NewSetReviewOp(rene, unix, &review.RemoveReview{ReviewId: "D1234"})
	setReviewOp4.Apply(&snapshot)

	assert.NotContains(t, snapshot.Reviews, "D1234")
}
