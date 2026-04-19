package handlers

import (
	"net/http"

	"july/internal/components/icons"
)

func FaviconHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	icons.Rocket(32, "#c026d3").Render(r.Context(), w)
}
