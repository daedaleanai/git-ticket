package http

import (
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/webui/session"
	"net/http"
	"net/url"
)

type ValidatedPayload interface {
	FromValues(values url.Values) ValidatedPayload
	Validate(c *cache.RepoCache) map[string]ValidationError
}

func IsValid(p ValidatedPayload, c *cache.RepoCache) bool {
	return len(p.Validate(c)) == 0
}

func ValidatePayload(payload ValidatedPayload, handler http.HandlerFunc) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			repo := LoadFromContext(r.Context(), &ContextualRepoCache{}).(*ContextualRepoCache).Repo
			if err := r.ParseForm(); err != nil {
				panic("failed to parse payload")
			}

			payload = payload.FromValues(r.Form)
			errors := payload.Validate(repo)

			if len(errors) > 0 {
				bag := LoadFromContext(r.Context(), &session.FlashMessageBag{}).(*session.FlashMessageBag)
				var messages []session.FlashValidationError

				for key, err := range errors {
					messages = append(messages, session.NewValidationError(key, err.Error()))
				}

				bag.AddValidationErrors(messages...)
			}
		}
		handler(w, r)
	}
}
