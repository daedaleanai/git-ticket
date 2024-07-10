package http

import (
	"context"
	"net/http"
)

func LoadIntoContext(r *http.Request, l ContextLoadable) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), l.ContextKey(), l))
}

func LoadFromContext(ctx context.Context, l ContextLoadable) ContextLoadable {
	return ctx.Value(l.ContextKey()).(ContextLoadable)
}

type ContextLoadable interface {
	ContextKey() string
}
