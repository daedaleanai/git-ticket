package webui

import (
	"fmt"
	"log"
	"net/http"
)

type validationError struct {
	msg string
}

func (e *validationError) Error() string { return e.msg }

type notFoundError struct {
	msg string
}

func (e *notFoundError) Error() string { return e.msg }

func errorIntoResponse(e error, w http.ResponseWriter) {
	switch e.(type) {
	default:
		w.WriteHeader(500)
		w.Write([]byte("An unknown error occurred"))
		log.Println(fmt.Sprintf("Internal server error: %s", e.Error()))
	case *validationError:
		w.WriteHeader(400)
		w.Write([]byte("Invalid request: "))
		w.Write([]byte(e.Error()))
	case *notFoundError:
		w.WriteHeader(404)
		w.Write([]byte("Resource not found: "))
		w.Write([]byte(e.Error()))
	}

}
