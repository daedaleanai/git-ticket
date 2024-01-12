package bug

import (
	"encoding/json"
	"fmt"

	termtext "github.com/MichaelMure/go-term-text"
	"github.com/pkg/errors"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/daedaleanai/git-ticket/util/text"
	"github.com/daedaleanai/git-ticket/util/timestamp"
)

var _ Operation = &EditCommentOperation{}

// EditCommentOperation will change a comment in the bug
type EditCommentOperation struct {
	OpBase
	Target  entity.Id         `json:"target"`
	Message string            `json:"message"`
	Files   []repository.Hash `json:"files"`
}

// Sign-post method for gqlgen
func (op *EditCommentOperation) IsOperation() {}

func (op *EditCommentOperation) base() *OpBase {
	return &op.OpBase
}

func (op *EditCommentOperation) Id() entity.Id {
	return idOperation(op)
}

func (op *EditCommentOperation) Apply(snapshot *Snapshot) {
	// Todo: currently any message can be edited, even by a different author
	// crypto signature are needed.

	snapshot.addActor(op.Author)

	comment := Comment{
		id:       op.Target,
		Author:   op.Author,
		Message:  op.Message,
		Files:    op.Files,
		Edited:   true,
		UnixTime: timestamp.Timestamp(op.UnixTime),
	}

	// Updating the corresponding comment
	var index int
	for i := range snapshot.Comments {
		if snapshot.Comments[i].Id() == op.Target {
			snapshot.Comments[i].Message = op.Message
			snapshot.Comments[i].Files = op.Files
			snapshot.Comments[i].Edited = true
			index = i
			break
		}
	}

	item := &EditCommentTimelineItem{
		CommentTimelineItem: NewCommentTimelineItem(op.Id(), index, comment),
	}

	snapshot.Timeline = append(snapshot.Timeline, item)
}

func (op *EditCommentOperation) GetFiles() []repository.Hash {
	return op.Files
}

func (op *EditCommentOperation) Validate() error {
	if err := opBaseValidate(op, EditCommentOp); err != nil {
		return err
	}

	if err := op.Target.Validate(); err != nil {
		return errors.Wrap(err, "target hash is invalid")
	}

	if !text.Safe(op.Message) {
		return fmt.Errorf("message is not fully printable")
	}

	return nil
}

// UnmarshalJSON is a two step JSON unmarshaling
// This workaround is necessary to avoid the inner OpBase.MarshalJSON
// overriding the outer op's MarshalJSON
func (op *EditCommentOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct {
		Target  entity.Id         `json:"target"`
		Message string            `json:"message"`
		Files   []repository.Hash `json:"files"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Target = aux.Target
	op.Message = aux.Message
	op.Files = aux.Files

	return nil
}

// Sign post method for gqlgen
func (op *EditCommentOperation) IsAuthored() {}

func NewEditCommentOp(author identity.Interface, unixTime int64, target entity.Id, message string, files []repository.Hash) *EditCommentOperation {
	return &EditCommentOperation{
		OpBase:  newOpBase(EditCommentOp, author, unixTime),
		Target:  target,
		Message: message,
		Files:   files,
	}
}

// CreateTimelineItem replace a AddComment operation in the Timeline and hold its edition history
type EditCommentTimelineItem struct {
	CommentTimelineItem
}

func (a EditCommentTimelineItem) String() string {
	termWidth, _, err := text.GetTermDim()
	if err != nil {
		termWidth = 200
	}
	comment, _ := termtext.WrapLeftPadded(a.Message, termWidth, timelineCommentOffset)
	return fmt.Sprintf("(%s) %s: edited comment #%d\n%s",
		a.CreatedAt.Time().Format("2006-01-02 15:04:05"),
		termtext.LeftPadMaxLine(a.Author.DisplayName(), timelineDisplayNameWidth, 0),
		a.Index,
		comment)
}

// Sign post method for gqlgen
func (a *EditCommentTimelineItem) IsAuthored() {}

// Convenience function to apply the operation
func EditComment(b Interface, author identity.Interface, unixTime int64, target entity.Id, message string) (*EditCommentOperation, error) {
	return EditCommentWithFiles(b, author, unixTime, target, message, nil)
}

func EditCommentWithFiles(b Interface, author identity.Interface, unixTime int64, target entity.Id, message string, files []repository.Hash) (*EditCommentOperation, error) {
	editCommentOp := NewEditCommentOp(author, unixTime, target, message, files)
	if err := editCommentOp.Validate(); err != nil {
		return nil, err
	}
	b.Append(editCommentOp)
	return editCommentOp, nil
}

// Convenience function to edit the body of a bug (the first comment)
func EditCreateComment(b Interface, author identity.Interface, unixTime int64, message string) (*EditCommentOperation, error) {
	createOp := b.FirstOp().(*CreateOperation)
	return EditComment(b, author, unixTime, createOp.Id(), message)
}

// Convenience function to edit the body of a bug (the first comment)
func EditCreateCommentWithFiles(b Interface, author identity.Interface, unixTime int64, message string, files []repository.Hash) (*EditCommentOperation, error) {
	createOp := b.FirstOp().(*CreateOperation)
	return EditCommentWithFiles(b, author, unixTime, createOp.Id(), message, files)
}
