package http

import (
	"fmt"
	"net/http"
)

type InvalidRequestError struct {
	Msg string
}

type ValidationError InvalidRequestError

func (e *ValidationError) Error() string { return e.Msg }

func (e *InvalidRequestError) Error() string { return e.Msg }

type MalformedRequestError struct {
	Prev error
}

func (e *MalformedRequestError) Error() string {
	return fmt.Errorf("failed to decode body: %w", e.Prev).Error()
}

type notFoundError struct {
	msg string
}

func (e *notFoundError) Error() string { return e.msg }

func TicketNotFound(ticketId string) *notFoundError {
	return &notFoundError{msg: fmt.Sprintf("unable to find ticket with id [%s]", ticketId)}
}

func ErrorIntoResponse(e error, w http.ResponseWriter) {
	switch e.(type) {
	default:
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("An unknown error occurred: "))
	case *InvalidRequestError, *ValidationError, *MalformedRequestError:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request: "))
	case *notFoundError:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Resource not found: "))
	}
	w.Write([]byte(e.Error()))
}
