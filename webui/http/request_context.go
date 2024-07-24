package http

import (
	"context"
	"github.com/daedaleanai/git-ticket/cache"
	"net/http"
)

func LoadIntoContext(r *http.Request, l ContextLoadable) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), l.ContextKey(), l))
}

func LoadFromContext(ctx context.Context, l ContextLoadable) ContextLoadable {
	loadable, ok := FindInContext(l, ctx)
	if !ok {
		panic("loadable not found in request context")
	}
	return loadable.(ContextLoadable)
}

func FindInContext(l ContextLoadable, ctx context.Context) (interface{}, bool) {
	val := ctx.Value(l.ContextKey())
	if val == nil {
		return nil, false
	}
	return val, true
}

type ContextLoadable interface {
	ContextKey() string
}

type ContextualRepoCache struct {
	Repo *cache.RepoCache
}

func (c *ContextualRepoCache) ContextKey() string {
	return "repo_cache_context"
}
