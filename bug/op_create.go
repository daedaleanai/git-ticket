package bug

import (
	"encoding/json"
	"fmt"
	"strings"

	termtext "github.com/MichaelMure/go-term-text"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/daedaleanai/git-ticket/util/text"
	"github.com/daedaleanai/git-ticket/util/timestamp"
)

var _ Operation = &CreateOperation{}

// CreateOperation define the initial creation of a bug
type CreateOperation struct {
	OpBase
	Title   string            `json:"title"`
	Message string            `json:"message"`
	Files   []repository.Hash `json:"files"`
}

// Sign-post method for gqlgen
func (op *CreateOperation) IsOperation() {}

func (op *CreateOperation) base() *OpBase {
	return &op.OpBase
}

func (op *CreateOperation) Id() entity.Id {
	return idOperation(op)
}

func (op *CreateOperation) Apply(snapshot *Snapshot) {
	snapshot.addActor(op.Author)
	snapshot.addParticipant(op.Author)

	snapshot.Title = op.Title

	comment := Comment{
		id:       op.Id(),
		Message:  op.Message,
		Author:   op.Author,
		UnixTime: timestamp.Timestamp(op.UnixTime),
	}

	snapshot.Comments = []Comment{comment}
	snapshot.Author = op.Author
	snapshot.CreateTime = op.Time()

	snapshot.Timeline = []TimelineItem{
		&CreateTimelineItem{
			CommentTimelineItem: NewCommentTimelineItem(op.Id(), 0, comment),
		},
	}
}

func (op *CreateOperation) GetFiles() []repository.Hash {
	return op.Files
}

func (op *CreateOperation) Validate() error {
	if err := opBaseValidate(op, CreateOp); err != nil {
		return err
	}

	if text.Empty(op.Title) {
		return fmt.Errorf("title is empty")
	}

	if strings.Contains(op.Title, "\n") {
		return fmt.Errorf("title should be a single line")
	}

	if !text.Safe(op.Title) {
		return fmt.Errorf("title is not fully printable")
	}

	if !text.Safe(op.Message) {
		return fmt.Errorf("message is not fully printable")
	}

	return nil
}

// UnmarshalJSON is a two step JSON unmarshaling
// This workaround is necessary to avoid the inner OpBase.MarshalJSON
// overriding the outer op's MarshalJSON
func (op *CreateOperation) UnmarshalJSON(data []byte) error {
	// Unmarshal OpBase and the op separately

	base := OpBase{}
	err := json.Unmarshal(data, &base)
	if err != nil {
		return err
	}

	aux := struct {
		Title   string            `json:"title"`
		Message string            `json:"message"`
		Files   []repository.Hash `json:"files"`
	}{}

	err = json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	op.OpBase = base
	op.Title = aux.Title
	op.Message = aux.Message
	op.Files = aux.Files

	return nil
}

// Sign post method for gqlgen
func (op *CreateOperation) IsAuthored() {}

func NewCreateOp(author identity.Interface, unixTime int64, title, message string, files []repository.Hash) *CreateOperation {
	return &CreateOperation{
		OpBase:  newOpBase(CreateOp, author, unixTime),
		Title:   title,
		Message: message,
		Files:   files,
	}
}

// CreateTimelineItem replace a Create operation in the Timeline and hold its edition history
type CreateTimelineItem struct {
	CommentTimelineItem
}

func (c CreateTimelineItem) String() string {
	message := c.Message
	if c.Message == "" {
		message = "No description provided."
	}
	termWidth, _, err := text.GetTermDim()
	if err != nil {
		termWidth = 200
	}
	message, _ = termtext.WrapLeftPadded(message, termWidth, timelineCommentOffset)
	return fmt.Sprintf("(%s) %s: created ticket\n%s",
		c.CreatedAt.Time().Format("2006-01-02 15:04:05"),
		termtext.LeftPadMaxLine(c.Author.DisplayName(), timelineDisplayNameWidth, 0),
		message)
}

// Sign post method for gqlgen
func (c *CreateTimelineItem) IsAuthored() {}

// Convenience function to apply the operation
func Create(author identity.Interface, unixTime int64, title, message string) (*Bug, *CreateOperation, error) {
	return CreateWithFiles(author, unixTime, title, message, nil)
}

func CreateWithFiles(author identity.Interface, unixTime int64, title, message string, files []repository.Hash) (*Bug, *CreateOperation, error) {
	newBug := NewBug()
	createOp := NewCreateOp(author, unixTime, title, message, files)

	if err := createOp.Validate(); err != nil {
		return nil, createOp, err
	}

	newBug.Append(createOp)

	return newBug, createOp, nil
}
