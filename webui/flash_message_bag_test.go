package webui

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFlashMessageBag_NewError(t *testing.T) {
	bag := createBag(t)

	msg := "help! I'm on fire!"
	bag.Add(NewError(msg))

	assertSingleMessage(t, FlashMessage{
		MessageType: errorMsg,
		Message:     msg,
	}, bag)
}

func TestFlashMessageBag_NewSuccess(t *testing.T) {
	bag := createBag(t)

	msg := "phew! The fire brigade showed up"
	bag.Add(NewSuccess(msg))

	assertSingleMessage(t, FlashMessage{
		MessageType: successMsg,
		Message:     msg,
	}, bag)
}

func TestFlashMessageBag_NewValidationError(t *testing.T) {
	bag := createBag(t)

	key := "fire"
	msg := "Should be water"
	bag.Add(NewValidationError(key, msg))

	assertSingleMessage(t, FlashMessage{
		MessageType: validationErrorMsg,
		Key:         &key,
		Message:     msg,
	}, bag)
}

func TestFlashMessageBag_MessagesClearsAfterRead(t *testing.T) {
	bag := createBag(t)

	bag.Add(NewError("foo"))
	bag.Add(NewError("bar"))

	flashes := bag.Messages()
	require.Len(t, flashes, 2)

	flashes = bag.Messages()
	require.Len(t, flashes, 0)
}

func assertSingleMessage(t *testing.T, expected FlashMessage, bag *FlashMessageBag) {
	flashes := bag.Messages()

	require.Len(t, flashes, 1)
	require.EqualValues(t, expected, flashes[0])
}

func createBag(t *testing.T) *FlashMessageBag {
	r, err := http.NewRequest(http.MethodGet, "example.com", bytes.NewReader([]byte("")))
	require.NoError(t, err)

	bag, err := newFlashMessageBag(r, httptest.NewRecorder())
	require.NoError(t, err)

	return bag
}