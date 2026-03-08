package api

import (
	"bytes"
	"july/internal/components"
	"net/http"
	"strings"
)

// errorWriter buffers the response so we can intercept error status codes.
type errorWriter struct {
	http.ResponseWriter
	buf     bytes.Buffer
	status  int
	headers http.Header
}

func (ew *errorWriter) Header() http.Header {
	return ew.headers
}

func (ew *errorWriter) WriteHeader(code int) {
	ew.status = code
}

func (ew *errorWriter) Write(b []byte) (int, error) {
	if ew.status == 0 {
		ew.status = http.StatusOK
	}
	return ew.buf.Write(b)
}

// ErrorMiddleware intercepts 4xx/5xx responses and renders a pretty error page.
func ErrorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ew := &errorWriter{ResponseWriter: w, status: 0, headers: make(http.Header)}
		next.ServeHTTP(ew, r)

		if ew.status == 0 {
			ew.status = http.StatusOK
		}

		// Pass through non-error responses and non-HTML requests unchanged
		if ew.status < 400 || !isHTMLRequest(r) {
			for k, v := range ew.headers {
				for _, vv := range v {
					w.Header().Add(k, vv)
				}
			}
			w.WriteHeader(ew.status)
			w.Write(ew.buf.Bytes())
			return
		}

		data := components.LayoutData{
			Title:       http.StatusText(ew.status),
			CurrentPath: r.URL.Path,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(ew.status)
		components.ErrorPage(data, ew.status).Render(r.Context(), w)
	})
}

func isHTMLRequest(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return accept == "" || strings.Contains(accept, "text/html")
}
