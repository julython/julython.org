package assets

import (
	"net/http"

	"july/internal/components/icons"
)

// Handler handles asset-related HTTP requests (favicon, etc.).
type Handler struct{}

// NewHandler creates a new assets handler.
func NewHandler() *Handler {
	return &Handler{}
}

// Register mounts asset routes on the given mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /favicon.svg", h.Favicon)
}

func (h *Handler) Favicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	icons.Rocket(32, "#c026d3").Render(r.Context(), w)
}
