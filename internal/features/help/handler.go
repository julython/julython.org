package help

import (
	"net/http"

	"july/internal/components/layout"
)

// Handler handles help, about, and privacy page HTTP requests.
type Handler struct{}

// NewHandler creates a new help handler.
func NewHandler() *Handler {
	return &Handler{}
}

// Register mounts help, about, and privacy routes on the given mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /help", h.Help)
	mux.HandleFunc("GET /about", h.About)
	mux.HandleFunc("GET /privacy", h.Privacy)
}

func (h *Handler) Help(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	layout := layout.LayoutData{
		Title:       "Help",
		CurrentPath: "/help",
		User:        layout.UserInfoFromContext(r),
	}

	HelpPage(layout).Render(ctx, w)
}

func (h *Handler) About(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	layout := layout.LayoutData{
		Title:       "About",
		CurrentPath: "/about",
		User:        layout.UserInfoFromContext(r),
	}

	AboutPage(layout).Render(ctx, w)
}

func (h *Handler) Privacy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	layout := layout.LayoutData{
		Title:       "Privacy",
		CurrentPath: "/privacy",
		User:        layout.UserInfoFromContext(r),
	}

	PrivacyPage(layout).Render(ctx, w)
}
