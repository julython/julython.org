package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"july/internal/i18n"

	ctxi18n "github.com/invopop/ctxi18n"
)

func init() {
	if err := i18n.Init(); err != nil {
		_, _ = os.Stderr.WriteString("i18n init failed: " + err.Error() + "\n")
		os.Exit(1)
	}
}

// --- isHTMXRequest ---

func TestIsHTMXRequest(t *testing.T) {
	t.Run("true when HX-Request is true", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("HX-Request", "true")
		assert.True(t, isHTMXRequest(req))
	})

	t.Run("false when HX-Request is absent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		assert.False(t, isHTMXRequest(req))
	})

	t.Run("false when HX-Request is false", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("HX-Request", "false")
		assert.False(t, isHTMXRequest(req))
	})
}

// --- isHTMLRequest ---

func TestIsHTMLRequest(t *testing.T) {
	t.Run("true when Accept header contains text/html", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
		assert.True(t, isHTMLRequest(req))
	})

	t.Run("true when Accept header is absent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		assert.True(t, isHTMLRequest(req))
	})

	t.Run("false when Accept is application/json only", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept", "application/json")
		assert.False(t, isHTMLRequest(req))
	})

	t.Run("false when Accept is */* without text/html", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept", "*/*")
		assert.False(t, isHTMLRequest(req))
	})
}

// --- errorWriter ---

func TestErrorWriterStatusCode(t *testing.T) {
	ew := &errorWriter{status: 0, headers: make(http.Header)}
	assert.Equal(t, 0, ew.status)
	ew.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, ew.status)
}

func TestErrorWriterWriteSetsDefaultStatus(t *testing.T) {
	ew := &errorWriter{status: 0, headers: make(http.Header)}
	n, err := ew.Write([]byte("body"))
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, "body", ew.buf.String())
	assert.Equal(t, http.StatusOK, ew.status)
}

func TestErrorWriterWritesBuffer(t *testing.T) {
	ew := &errorWriter{status: 0, headers: make(http.Header)}
	_, err := ew.Write([]byte("hello"))
	require.NoError(t, err)
	_, err = ew.Write([]byte(" world"))
	require.NoError(t, err)
	assert.Equal(t, "hello world", ew.buf.String())
	assert.Equal(t, http.StatusOK, ew.status)
}

func TestErrorWriterHeader(t *testing.T) {
	ew := &errorWriter{headers: make(http.Header)}
	ew.Header().Set("X-Test", "value")
	assert.Equal(t, "value", ew.headers.Get("X-Test"))
	ew.Header().Add("X-Multi", "a")
	ew.Header().Add("X-Multi", "b")
	assert.Equal(t, []string{"a", "b"}, ew.headers["X-Multi"])
}

// --- copyResponse ---

func TestCopyResponseHeaders(t *testing.T) {
	ew := &errorWriter{
		status:  http.StatusCreated,
		headers: http.Header{"X-Custom": []string{"value1", "value2"}},
	}
	rec := httptest.NewRecorder()
	copyResponse(rec, ew)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, []string{"value1", "value2"}, rec.Header()["X-Custom"])
}

func TestCopyResponseBody(t *testing.T) {
	buf := &bytes.Buffer{}
	buf.WriteString("hello world")
	ew := &errorWriter{
		buf:    *buf,
		status: http.StatusCreated,
	}
	rec := httptest.NewRecorder()
	copyResponse(rec, ew)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "hello world", rec.Body.String())
}

func TestCopyResponseToNonRecorder(t *testing.T) {
	// copyResponse should work with any http.ResponseWriter, not just httptest.Recorder.
	ew := &errorWriter{
		status:  418,
		headers: http.Header{"X-I-Am": []string{"a-teapot"}},
		buf:     *bytes.NewBufferString("teapots only"),
	}
	rec := httptest.NewRecorder()
	copyResponse(rec, ew)
	assert.Equal(t, 418, rec.Code)
	assert.Contains(t, rec.Body.String(), "teapots only")
	assert.Equal(t, "a-teapot", rec.Header().Get("X-I-Am"))
}

// --- ErrorMiddleware ---

func TestErrorMiddleware(t *testing.T) {
	t.Run("passes through 2xx unchanged", func(t *testing.T) {
		rec := httptest.NewRecorder()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
		mw := ErrorMiddleware(next)
		mw.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})

	t.Run("replaces 404 with HTML error page for HTML requests", func(t *testing.T) {
		rec := httptest.NewRecorder()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("body should be discarded"))
		})
		mw := ErrorMiddleware(next)
		req := httptest.NewRequest(http.MethodGet, "/page", nil)
		c, _ := ctxi18n.WithLocale(req.Context(), "en")
		req = req.WithContext(c)
		mw.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.NotContains(t, rec.Body.String(), "body should be discarded")
		assert.Contains(t, rec.Body.String(), "404")
	})

	t.Run("replaces 500 with HTML error page", func(t *testing.T) {
		rec := httptest.NewRecorder()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		mw := ErrorMiddleware(next)
		req := httptest.NewRequest(http.MethodGet, "/crash", nil)
		c, _ := ctxi18n.WithLocale(req.Context(), "en")
		req = req.WithContext(c)
		mw.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "500")
	})

	t.Run("passes through JSON API responses", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		})
		mw := ErrorMiddleware(next)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
		mw.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	})

	t.Run("passes through non-HTML requests for non-API paths", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		})
		mw := ErrorMiddleware(next)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/page", nil)
		req.Header.Set("Accept", "application/json")
		c, _ := ctxi18n.WithLocale(req.Context(), "en")
		req = req.WithContext(c)
		mw.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "error")
	})

	t.Run("HTMX 404 returns partial HTML with 200", func(t *testing.T) {
		rec := httptest.NewRecorder()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		mw := ErrorMiddleware(next)
		req := httptest.NewRequest(http.MethodGet, "/page", nil)
		req.Header.Set("HX-Request", "true")
		c, _ := ctxi18n.WithLocale(req.Context(), "en")
		req = req.WithContext(c)
		mw.ServeHTTP(rec, req)
		// HTMX gets a 200 so it can swap the fragment.
		assert.Equal(t, http.StatusOK, rec.Code)
		// Contains the 404 message in an HTMX error fragment (HTML entity-encoded).
		assert.Contains(t, rec.Body.String(), "doesn&#39;t exist")
	})
}
