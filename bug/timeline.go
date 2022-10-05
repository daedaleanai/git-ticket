package bug

import (
	"fmt"
	"strings"

	termtext "github.com/MichaelMure/go-term-text"

	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/repository"
	"github.com/daedaleanai/git-ticket/util/timestamp"
)

type TimelineItem interface {
	// ID return the identifier of the item
	Id() entity.Id
	// When returns the time of the item
	When() timestamp.Timestamp
	// Timeline specific print message
	String() string
}

// CommentHistoryStep hold one version of a message in the history
type CommentHistoryStep struct {
	// The author of the edition, not necessarily the same as the author of the
	// original comment
	Author identity.Interface
	// The new message
	Message  string
	UnixTime timestamp.Timestamp
}

// CommentTimelineItem is a TimelineItem that holds a Comment and its edition history
type CommentTimelineItem struct {
	id        entity.Id
	Author    identity.Interface
	Message   string
	Files     []repository.Hash
	CreatedAt timestamp.Timestamp
	LastEdit  timestamp.Timestamp
	History   []CommentHistoryStep
}

func NewCommentTimelineItem(ID entity.Id, comment Comment) CommentTimelineItem {
	return CommentTimelineItem{
		id:        ID,
		Author:    comment.Author,
		Message:   comment.Message,
		Files:     comment.Files,
		CreatedAt: comment.UnixTime,
		LastEdit:  comment.UnixTime,
		History: []CommentHistoryStep{
			{
				Message:  comment.Message,
				UnixTime: comment.UnixTime,
			},
		},
	}
}

func (c *CommentTimelineItem) Id() entity.Id {
	return c.id
}

func (c CommentTimelineItem) When() timestamp.Timestamp {
	return c.CreatedAt
}

func (c CommentTimelineItem) String() string {
	return fmt.Sprintf("(%s) %s: %s",
		c.CreatedAt.Time().Format("2006-01-02 15:04:05"),
		termtext.LeftPadMaxLine(c.Author.DisplayName(), 15, 0),
		c.Message)
}

// Append will append a new comment in the history and update the other values
func (c *CommentTimelineItem) Append(comment Comment) {
	c.Message = comment.Message
	c.Files = comment.Files
	c.LastEdit = comment.UnixTime
	c.History = append(c.History, CommentHistoryStep{
		Author:   comment.Author,
		Message:  comment.Message,
		UnixTime: comment.UnixTime,
	})
}

// Edited say if the comment was edited
func (c *CommentTimelineItem) Edited() bool {
	return len(c.History) > 1
}

// MessageIsEmpty return true is the message is empty or only made of spaces
func (c *CommentTimelineItem) MessageIsEmpty() bool {
	return len(strings.TrimSpace(c.Message)) == 0
}
