package bug

import (
	"fmt"

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

// CommentTimelineItem is a TimelineItem that holds a Comment and its edition history
type CommentTimelineItem struct {
	id        entity.Id
	Index     int
	Author    identity.Interface
	Message   string
	Files     []repository.Hash
	CreatedAt timestamp.Timestamp
}

func NewCommentTimelineItem(ID entity.Id, index int, comment Comment) CommentTimelineItem {
	return CommentTimelineItem{
		id:        ID,
		Index:     index,
		Author:    comment.Author,
		Message:   comment.Message,
		Files:     comment.Files,
		CreatedAt: comment.UnixTime,
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
