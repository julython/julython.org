package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		// */* does not contain "text/html"
		assert.False(t, isHTMLRequest(req))
	})
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

	t.Run("passes through JSON API responses unchanged", func(t *testing.T) {
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
		mw.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// --- copyResponse ---

func TestCopyResponseHeaders(t *testing.T) {
	ew := &errorWriter{
		status: http.StatusCreated,
		headers: http.Header{
			"X-Custom": []string{"value1", "value2"},
		},
	}
	rec := httptest.NewRecorder()
	copyResponse(rec, ew)
	assert.ElementsMatch(t, []string{"value1", "value2"}, rec.Header()["X-Custom"])
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

// --- errorWriter ---

func TestErrorWriterStatusCode(t *testing.T) {
	ew := &errorWriter{status: 0, headers: make(http.Header)}
	ew.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, ew.status)
}

func TestErrorWriterWriteSetsDefaultStatus(t *testing.T) {
	ew := &errorWriter{status: 0, headers: make(http.Header)}
	_, _ = ew.Write([]byte("body"))
	assert.Equal(t, http.StatusOK, ew.status)
}

func TestErrorWriterBuffer(t *testing.T) {
	ew := &errorWriter{status: 0, headers: make(http.Header)}
	n, _ := ew.Write([]byte("test data"))
	assert.Equal(t, 9, n)
	assert.Equal(t, "test data", ew.buf.String())
}

func TestErrorWriterHeader(t *testing.T) {
	ew := &errorWriter{headers: make(http.Header)}
	ew.Header().Set("X-Test", "value")
	assert.Equal(t, "value", ew.headers.Get("X-Test"))
}

// --- RecoveryMiddleware ---

func TestRecoveryMiddleware(t *testing.T) {
	t.Run("recovers from panic", func(t *testing.T) {
		panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("something went wrong")
		})
		mw := RecoveryMiddleware(panicHandler)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		mw.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Internal Server Error")
	})

	t.Run("passes through normal requests", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
		mw := RecoveryMiddleware(next)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		mw.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})
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

func Test_errorWriter_Header(t *testing.T) {
	type fields struct {
		ResponseWriter http.ResponseWriter
		buf            bytes.Buffer
		status         int
		headers        http.Header
	}
	tests := []struct {
		name   string
		fields fields
		want   http.Header
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ew := &errorWriter{
				ResponseWriter: tt.fields.ResponseWriter,
				buf:            tt.fields.buf,
				status:         tt.fields.status,
				headers:        tt.fields.headers,
			}
			if got := ew.Header(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("errorWriter.Header() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_errorWriter_WriteHeader(t *testing.T) {
	type fields struct {
		ResponseWriter http.ResponseWriter
		buf            bytes.Buffer
		status         int
		headers        http.Header
	}
	type args struct {
		code int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ew := &errorWriter{
				ResponseWriter: tt.fields.ResponseWriter,
				buf:            tt.fields.buf,
				status:         tt.fields.status,
				headers:        tt.fields.headers,
			}
			ew.WriteHeader(tt.args.code)
		})
	}
}

func Test_errorWriter_Write(t *testing.T) {
	type fields struct {
		ResponseWriter http.ResponseWriter
		buf            bytes.Buffer
		status         int
		headers        http.Header
	}
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ew := &errorWriter{
				ResponseWriter: tt.fields.ResponseWriter,
				buf:            tt.fields.buf,
				status:         tt.fields.status,
				headers:        tt.fields.headers,
			}
			got, err := ew.Write(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("errorWriter.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("errorWriter.Write() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_copyResponse(t *testing.T) {
	type args struct {
		w  http.ResponseWriter
		ew *errorWriter
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copyResponse(tt.args.w, tt.args.ew)
		})
	}
}

func Test_isHTMLRequest(t *testing.T) {
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHTMLRequest(tt.args.r); got != tt.want {
				t.Errorf("isHTMLRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isHTMXRequest(t *testing.T) {
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHTMXRequest(tt.args.r); got != tt.want {
				t.Errorf("isHTMXRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
