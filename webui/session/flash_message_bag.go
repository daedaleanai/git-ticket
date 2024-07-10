package session

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

const flashMessageBagContextKey = "flash_message_bag_context"

func (b *FlashMessageBag) ContextKey() string {
	return flashMessageBagContextKey
}

// **Note:** we're only using sessions to show flash messages.
// If we ever use it for auth stuff (which is probably never), this should be an env var.
const ddlnSessionKey = "ddln_gt_session"

const validationErrorFlashKey = "validation_errors"
const miscellaneousFlashKey = "misc"

func (b *FlashMessageBag) AddValidationErrors(errors ...FlashValidationError) {
	defer b.save()

	for _, e := range errors {
		m, err := json.Marshal(e)
		if err != nil {
			panic(fmt.Sprintf("failed to create new flash message '%s': %s", e.Message, err))
		}

		b.session.AddFlash(m, validationErrorFlashKey)
	}
}

func (b *FlashMessageBag) AddMessage(message FlashMessage) {
	m, err := json.Marshal(message)
	if err != nil {
		panic(fmt.Sprintf("failed to create new flash message '%s': %s", message.Message, err))
	}
	b.session.AddFlash(m, miscellaneousFlashKey)
}

func (b *FlashMessageBag) save() {
	if err := b.session.Save(b.request, b.writer); err != nil {
		panic(fmt.Sprintf("failed to save flash messages to session: %s", err))
	}
}

func NewFlashMessageBag(r *http.Request, w http.ResponseWriter) (*FlashMessageBag, error) {
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

func NewValidationError(field string, err error) FlashValidationError {
	return FlashValidationError{
		Field:   field,
		Message: err.Error(),
	}
}

func (b *FlashMessageBag) Messages() []FlashMessage {
	var flashes []FlashMessage

	for _, v := range b.session.Flashes(miscellaneousFlashKey) {
		var m FlashMessage
		unmarshalMessage(v, &m)
		flashes = append(flashes, m)
	}

	return flashes
}

func unmarshalMessage[M *FlashMessage | *FlashValidationError](b interface{}, m M) {
	s := fmt.Sprintf("%s", b)

	if err := json.Unmarshal([]byte(s), m); err != nil {
		panic(fmt.Sprintf("failed to read flash messages: %s", s))
	}
}

func (b *FlashMessageBag) ValidationErrors() map[string]FlashValidationError {
	var flashes []FlashValidationError

	for _, v := range b.session.Flashes(validationErrorFlashKey) {
		var m FlashValidationError
		unmarshalMessage(v, &m)
		flashes = append(flashes, m)
	}

	errors := make(map[string]FlashValidationError)
	for _, f := range flashes {
		errors[f.Field] = f
	}
	return errors
}

type FlashMessage struct {
	MessageType flashMessageType
	Message     interface{}
}

type FlashValidationError struct {
	Field   string
	Message string
}

func (f *FlashMessage) CssClass() string {
	var s string
	switch f.MessageType {
	case successMsg:
		s = "success"
	case errorMsg:
		s = "danger"
	}
	return s
}

type flashMessageType int

const (
	errorMsg flashMessageType = iota
	successMsg
)
