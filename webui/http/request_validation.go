package http

import (
	"fmt"
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/webui/session"
	"net/http"
	"net/url"
)

type ValidatedPayload interface {
	Validate(c *cache.RepoCache) map[string]ValidationError
}

func IsValid(p ValidatedPayload, c *cache.RepoCache) bool {
	return len(p.Validate(c)) == 0
}

func WithValidatedPayload(f func(url.Values) ValidatedPayload, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			repo := LoadFromContext(r.Context(), &ContextualRepoCache{}).(*ContextualRepoCache).Repo
			if err := r.ParseForm(); err != nil {
				panic(fmt.Errorf("failed to parse payload: %w", err))
			}

			payload := f(r.Form)
			errors := payload.Validate(repo)

			if len(errors) > 0 {
				bag := LoadFromContext(r.Context(), &session.FlashMessageBag{}).(*session.FlashMessageBag)
				var messages []session.FlashValidationError

				for key, err := range errors {
					messages = append(messages, session.NewValidationError(key, &err))
				}

				bag.AddValidationErrors(messages...)
			}
		}
		handler(w, r)
	}
}
