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
	if loadable, ok := ctx.Value(l.ContextKey()).(ContextLoadable); !ok || loadable == nil {
		panic("loadable not found in request context")
	} else {
		return loadable
	}
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
