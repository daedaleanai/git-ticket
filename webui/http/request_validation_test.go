package http

import (
	"bytes"
	"context"
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/webui/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

const failureKey = "dancing"
const failureMessage = "dancing is required"

func TestValidatePayloadForFailing(t *testing.T) {
	r, err := http.NewRequest(http.MethodPost, "example.com?valid=false", bytes.NewReader([]byte("")))
	require.NoError(t, err)

	r = validatePayloadForRequest(t, r)
	errs := make(map[string]session.FlashValidationError)
	errs[failureKey] = session.NewValidationError(failureKey, &ValidationError{Msg: failureMessage})
	assertValidationErrors(t, r.Context(), errs)
}

func TestValidatePayloadForPassing(t *testing.T) {
	r, err := http.NewRequest(http.MethodPost, "example.com?valid=true", bytes.NewReader([]byte("")))
	require.NoError(t, err)

	r = validatePayloadForRequest(t, r)
	assertValidationErrors(t, r.Context(), make(map[string]session.FlashValidationError))
}

func TestShouldNotValidateGetRequest(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "example.com?valid=false", bytes.NewReader([]byte("")))
	require.NoError(t, err)

	r = validatePayloadForRequest(t, r)
	assertValidationErrors(t, r.Context(), make(map[string]session.FlashValidationError))
}

func validatePayloadForRequest(t *testing.T, r *http.Request) *http.Request {
	w := httptest.NewRecorder()

	bag, err := session.NewFlashMessageBag(r, w)
	require.NoError(t, err)

	r = LoadIntoContext(r, &ContextualRepoCache{Repo: &cache.RepoCache{}})
	r = LoadIntoContext(r, bag)
	WithValidatedPayload(SomePayloadFromValues, handle)(w, r)

	return r
}

func handle(w http.ResponseWriter, r *http.Request) {
	// Do nothing
	return
}

type SomePayload struct {
	Valid bool
}

func (p *SomePayload) Validate(c *cache.RepoCache) map[string]ValidationError {
	errors := make(map[string]ValidationError)

	if !p.Valid {
		errors[failureKey] = ValidationError{Msg: failureMessage}
	}

	return errors
}

func SomePayloadFromValues(values url.Values) ValidatedPayload {
	valid, _ := strconv.ParseBool(values.Get("valid"))

	return &SomePayload{
		Valid: valid,
	}
}

func assertValidationErrors(t *testing.T, c context.Context, expected map[string]session.FlashValidationError) {
	bag := LoadFromContext(c, &session.FlashMessageBag{}).(*session.FlashMessageBag)
	assert.Equal(t, expected, bag.ValidationErrors())
}
