package api

import (
	"bytes"
	"july/internal/components"
	"net/http"
	"strings"
)

type errorWriter struct {
	http.ResponseWriter
	buf     bytes.Buffer
	status  int
	headers http.Header
}

func (ew *errorWriter) Header() http.Header  { return ew.headers }
func (ew *errorWriter) WriteHeader(code int) { ew.status = code }
func (ew *errorWriter) Write(b []byte) (int, error) {
	if ew.status == 0 {
		ew.status = http.StatusOK
	}
	return ew.buf.Write(b)
}

func ErrorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ew := &errorWriter{ResponseWriter: w, status: 0, headers: make(http.Header)}
		next.ServeHTTP(ew, r)

		if ew.status == 0 {
			ew.status = http.StatusOK
		}

		// Pass through non-errors unchanged.
		if ew.status < 400 {
			copyResponse(w, ew)
			return
		}

		// JSON and plain-text APIs under /api/ must not be replaced with HTML error pages
		// (e.g. clients with no Accept header, or */* without text/html).
		if strings.HasPrefix(r.URL.Path, "/api/") {
			copyResponse(w, ew)
			return
		}

		// Handle HTMX first to return partial html
		if isHTMXRequest(r) {
			data := components.LayoutData{
				Title:       http.StatusText(ew.status),
				CurrentPath: r.URL.Path,
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("HX-Retarget", "#htmx-error")
			w.Header().Set("HX-Reswap", "innerHTML")
			w.WriteHeader(http.StatusOK)
			components.ErrorFragment(data, ew.status).Render(r.Context(), w)
			return
		}

		// Pass through non-HTML requests (JSON APIs etc.) unchanged.
		if !isHTMLRequest(r) {
			copyResponse(w, ew)
			return
		}

		data := components.LayoutData{
			Title:       http.StatusText(ew.status),
			CurrentPath: r.URL.Path,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// HTMX needs a 200 to actually swap the response body.
		// Use HX-Retarget + HX-Reswap to redirect the fragment to a known error container.
		if isHTMXRequest(r) {
			w.Header().Set("HX-Retarget", "#htmx-error")
			w.Header().Set("HX-Reswap", "innerHTML")
			w.WriteHeader(http.StatusOK)
			components.ErrorFragment(data, ew.status).Render(r.Context(), w)
			return
		}

		w.WriteHeader(ew.status)
		components.ErrorPage(data, ew.status).Render(r.Context(), w)
	})
}

func copyResponse(w http.ResponseWriter, ew *errorWriter) {
	for k, v := range ew.headers {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(ew.status)
	w.Write(ew.buf.Bytes())
}

func isHTMLRequest(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return accept == "" || strings.Contains(accept, "text/html")
}

func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
