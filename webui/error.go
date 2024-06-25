package webui

import (
	"fmt"
	"net/http"
)

type invalidRequestError struct {
	msg string
}

func (e *invalidRequestError) Error() string { return e.msg }

type malformedRequestError struct {
	prev error
}

func (e *malformedRequestError) Error() string {
	return fmt.Errorf("failed to decode body: %w", e.prev).Error()
}

type notFoundError struct {
	msg string
}

func (e *notFoundError) Error() string { return e.msg }

func ticketNotFound(ticketId string) *notFoundError {
	return &notFoundError{msg: fmt.Sprintf("unable to find ticket with id [%s]", ticketId)}
}

func errorIntoResponse(e error, w http.ResponseWriter) {
	switch e.(type) {
	default:
		w.WriteHeader(500)
		w.Write([]byte("An unknown error occurred: "))
	case *invalidRequestError:
		w.WriteHeader(400)
		w.Write([]byte("Invalid request: "))
	case *notFoundError:
		w.WriteHeader(404)
		w.Write([]byte("Resource not found: "))
	}
	w.Write([]byte(e.Error()))
}
