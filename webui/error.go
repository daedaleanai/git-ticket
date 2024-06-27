package webui

import (
	"fmt"
	"log"
	"net/http"
)

type InvalidRequestError struct {
	msg string
}

func (e *InvalidRequestError) Error() string { return e.msg }

type MalformedRequestError struct {
	prev error
}

func (e *MalformedRequestError) Error() string {
	return fmt.Errorf("failed to decode body: %w", e.prev).Error()
}

type NotFoundError struct {
	msg string
}

func (e *NotFoundError) Error() string { return e.msg }

func TicketNotFound(ticketId string) *NotFoundError {
	return &NotFoundError{msg: fmt.Sprintf("unable to find ticket with id [%s]", ticketId)}
}

func ErrorIntoResponse(e error, w http.ResponseWriter) {
	switch e.(type) {
	default:
		w.WriteHeader(500)
		w.Write([]byte("An unknown error occurred"))
		log.Println(fmt.Sprintf("Internal server error: %s", e.Error()))
	case *InvalidRequestError:
		w.WriteHeader(400)
		w.Write([]byte("Invalid request: "))
		w.Write([]byte(e.Error()))
	case *NotFoundError:
		w.WriteHeader(404)
		w.Write([]byte("Resource not found: "))
		w.Write([]byte(e.Error()))
	}

}
