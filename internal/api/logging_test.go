package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"july/internal/config"
)

// --- LoggingMiddleware ---

func TestLoggingMiddleware(t *testing.T) {
	t.Run("generates request ID when absent", func(t *testing.T) {
		rec := httptest.NewRecorder()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
		logger := zerolog.Nop()
		cfg := &config.Config{Env: "development", ProjectID: "test-project"}
		mw := LoggingMiddleware(logger, cfg)(next)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		mw.ServeHTTP(rec, req)

		// Check that the response header was set.
		reqID := rec.Header().Get("X-Request-Id")
		assert.NotEmpty(t, reqID)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})

	t.Run("passes through request ID from client", func(t *testing.T) {
		rec := httptest.NewRecorder()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
		logger := zerolog.Nop()
		cfg := &config.Config{Env: "development", ProjectID: "test-project"}
		mw := LoggingMiddleware(logger, cfg)(next)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-Id", "custom-id-123")
		mw.ServeHTTP(rec, req)

		assert.Equal(t, "custom-id-123", rec.Header().Get("X-Request-Id"))
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("production mode adds GCP trace info", func(t *testing.T) {
		rec := httptest.NewRecorder()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		logger := zerolog.Nop()
		cfg := &config.Config{Env: "production", ProjectID: "my-project"}
		mw := LoggingMiddleware(logger, cfg)(next)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Cloud-Trace-Context", "abc123/def456;o=1")
		mw.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotEmpty(t, rec.Header().Get("X-Request-Id"))
	})

	t.Run("production mode without trace header", func(t *testing.T) {
		rec := httptest.NewRecorder()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("created"))
		})
		logger := zerolog.Nop()
		cfg := &config.Config{Env: "production", ProjectID: "my-project"}
		mw := LoggingMiddleware(logger, cfg)(next)
		req := httptest.NewRequest(http.MethodPost, "/api/data", nil)
		mw.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "created", rec.Body.String())
	})
}

// --- responseWriter ---

func TestResponseWriterCapturesStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, status: http.StatusOK}
	rw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, rw.status)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestResponseWriterWithDefaultStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, status: http.StatusOK}
	// WriteHeader is never called, so status should remain the default.
	assert.Equal(t, http.StatusOK, rw.status)
}

// --- parseTraceHeader ---

func TestParseTraceHeader(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		traceID, spanID := parseTraceHeader("")
		assert.Equal(t, "", traceID)
		assert.Equal(t, "", spanID)
	})

	t.Run("trace only", func(t *testing.T) {
		traceID, spanID := parseTraceHeader("abc123")
		assert.Equal(t, "abc123", traceID)
		assert.Equal(t, "", spanID)
	})

	t.Run("trace and span", func(t *testing.T) {
		traceID, spanID := parseTraceHeader("abc123/def456;o=1")
		assert.Equal(t, "abc123", traceID)
		assert.Equal(t, "def456", spanID)
	})

	t.Run("trace and span without options", func(t *testing.T) {
		traceID, spanID := parseTraceHeader("abc123/def456")
		assert.Equal(t, "abc123", traceID)
		assert.Equal(t, "def456", spanID)
	})

	t.Run("trace with options but no span", func(t *testing.T) {
		traceID, spanID := parseTraceHeader("abc123;o=1")
		assert.Equal(t, "abc123", traceID)
		assert.Equal(t, "", spanID)
	})
}

// --- generateRequestID ---

func TestGenerateRequestID(t *testing.T) {
	t.Run("produces non-empty hex string", func(t *testing.T) {
		id := generateRequestID()
		assert.NotEmpty(t, id)
		for _, ch := range id {
			assert.True(t, (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f'), "unexpected char %q", ch)
		}
	})

	t.Run("produces unique IDs", func(t *testing.T) {
		ids := make(map[string]bool, 10)
		for i := 0; i < 10; i++ {
			id := generateRequestID()
			assert.False(t, ids[id], "duplicate ID at iteration %d", i)
			ids[id] = true
		}
	})
}
