package webui

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFeatureFlag_IsEnabled(t *testing.T) {
	enabled := FeatureList([]FeatureFlag{TicketCreate})
	ctx := context.WithValue(context.Background(), enabled.ContextKey(), enabled)

	assert.True(t, TicketCreate.IsEnabled(ctx))
}

func TestFeatureFlag_IsNotEnabled(t *testing.T) {
	ctx := context.Background()

	assert.False(t, TicketCreate.IsEnabled(ctx))
}

func TestMiddleware_ReturnsNotFoundForDisabledFlag(t *testing.T) {
	testMiddleware(t, []string{}, TicketCreate.name(), []FeatureFlag{TicketCreate}, http.StatusNotFound)
}

func TestMiddleware_StoresEnabledFlagInRequestContext(t *testing.T) {
	testMiddleware(t, []string{TicketCreate.name()}, TicketCreate.name(), []FeatureFlag{TicketCreate}, http.StatusOK)
}

func TestMiddleware_DoesNotStoreInvalidFlagInContext(t *testing.T) {
	testMiddleware(t, []string{"foo"}, "bar", FeatureList(nil), http.StatusOK)
}

func testMiddleware(t *testing.T, flags []string, mockRouteName string, expectEnabled FeatureList, expectStatus int) {
	router := mux.NewRouter()
	router.Use(featureFlagMiddleware(flags))
	w := newLoggingResponseWriter(httptest.NewRecorder())

	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	handler := assertEnabledFeatures(t, expectEnabled)
	router.HandleFunc("/", handler).Name(mockRouteName)

	router.ServeHTTP(w, r)

	assert.Equal(t, expectStatus, w.statusCode)
}

func assertEnabledFeatures(t *testing.T, expected FeatureList) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expected, r.Context().Value(expected.ContextKey()))
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
