package webui

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/sessions"
	"net/http"
)

type FlashMessageBag struct {
	session *sessions.Session
	request *http.Request
	writer  http.ResponseWriter
}

// **Note:** we're only using sessions to show flash messages.
// If we ever use it for auth stuff (which is probably never), this should be an env var.
const ddlnSessionKey = "DDLN_GT_SESSION"

func (b FlashMessageBag) Add(messages ...FlashMessage) {
	defer func() {
		if err := b.session.Save(b.request, b.writer); err != nil {
			panic(fmt.Sprintf("failed to save flash messages to session: %s", err))
		}
	}()

	for _, message := range messages {
		m, err := json.Marshal(message)
		if err != nil {
			panic(fmt.Sprintf("failed to create new flash message '%s': %s", message.Message, err))
		}

		b.session.AddFlash(m)
	}
}

func newFlashMessageBag(r *http.Request, w http.ResponseWriter) (*FlashMessageBag, error) {
	store := sessions.NewCookieStore([]byte(ddlnSessionKey))
	session, err := store.Get(r, ddlnSessionKey)
	if err != nil {
		return nil, err
	}

	return &FlashMessageBag{
		session: session,
		writer:  w,
		request: r,
	}, nil
}

func NewError(msg string) FlashMessage {
	return FlashMessage{
		MessageType: errorMsg,
		Message:     msg,
	}
}

func NewSuccess(msg string) FlashMessage {
	return FlashMessage{
		MessageType: successMsg,
		Message:     msg,
	}
}

func NewValidationError(key string, msg string) FlashMessage {
	return FlashMessage{
		MessageType: validationErrorMsg,
		Key:         &key,
		Message:     msg,
	}
}

func (b FlashMessageBag) Messages() []FlashMessage {
	var flashes []FlashMessage
	for _, v := range b.session.Flashes() {
		s := fmt.Sprintf("%s", v)
		var m FlashMessage

		if err := json.Unmarshal([]byte(s), &m); err != nil {
			panic(fmt.Sprintf("failed to clear flash messages: %s", s))
		} else {
			flashes = append(flashes, m)
		}
	}

	if err := b.session.Save(b.request, b.writer); err != nil {
		panic(fmt.Sprintf("failed to clear flash messages: %s", err))
	}
	return flashes
}

type FlashMessage struct {
	MessageType flashMessageType
	Message     interface{}
	Key         *string
}

func (f FlashMessage) CssClass() string {
	var s string
	switch f.MessageType {
	case successMsg:
		s = "success"
	case errorMsg, validationErrorMsg:
		s = "danger"
	}
	return s
}

func (f FlashMessage) IsValidationError() bool {
	return f.MessageType == validationErrorMsg
}

type flashMessageType int

const (
	errorMsg flashMessageType = iota
	successMsg
	validationErrorMsg
)
