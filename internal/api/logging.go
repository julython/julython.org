package api

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"july/internal/config"
)

// LoggingMiddleware injects a request-scoped zerolog logger (with request ID)
// into the context and logs each request on completion.
func LoggingMiddleware(logger zerolog.Logger, cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get("X-Request-Id")
			if reqID == "" {
				reqID = generateRequestID()
			}
			w.Header().Set("X-Request-Id", reqID)

			ctx := logger.With().
				Str("request_id", reqID).
				Str("method", r.Method).
				Str("path", r.URL.Path)

			if cfg.IsProduction() {
				traceID, spanID := parseTraceHeader(r.Header.Get("X-Cloud-Trace-Context"))
				trace := fmt.Sprintf("projects/%s/traces/%s", cfg.ProjectID, traceID)
				ctx = ctx.
					Str("logging.googleapis.com/trace", trace).
					Str("logging.googleapis.com/spanId", spanID).
					Dict("logging.googleapis.com/labels", zerolog.Dict().
						Str("request_id", reqID),
					)
			}

			reqLogger := ctx.Logger()
			r = r.WithContext(reqLogger.WithContext(r.Context()))

			start := time.Now()
			wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			reqLogger.Info().
				Int("status", wrapped.status).
				Dur("duration", time.Since(start)).
				Msgf("%s %s", r.Method, r.URL.RequestURI())
		})
	}
}

// parseTraceHeader parses "X-Cloud-Trace-Context: TRACE_ID/SPAN_ID;o=1"
func parseTraceHeader(h string) (traceID, spanID string) {
	if h == "" {
		return "", ""
	}
	parts := strings.SplitN(h, ";", 2)
	ids := strings.SplitN(parts[0], "/", 2)
	traceID = ids[0]
	if len(ids) == 2 {
		spanID = ids[1]
	}
	return traceID, spanID
}

// responseWriter captures the HTTP status code for logging.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return fmt.Sprintf("%x", b)
}
