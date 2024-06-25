package webui

import (
	"fmt"
	"log"
	"net/http"
)

type ValidationError struct {
	msg string
}

func (e *ValidationError) Error() string { return e.msg }

type NotFoundError struct {
	msg string
}

func (e *NotFoundError) Error() string { return e.msg }

func ErrorIntoResponse(e error, w http.ResponseWriter) {
	switch e.(type) {
	default:
		w.WriteHeader(500)
		w.Write([]byte("An unknown error occurred"))
		log.Println(fmt.Sprintf("Internal server error: %s", e.Error()))
	case *ValidationError:
		w.WriteHeader(400)
		w.Write([]byte("Invalid request: "))
		w.Write([]byte(e.Error()))
	case *NotFoundError:
		w.WriteHeader(404)
		w.Write([]byte("Resource not found: "))
		w.Write([]byte(e.Error()))
	}

}
