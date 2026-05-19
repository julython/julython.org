package help

import (
	"net/http"

	"july/internal/components"
	"july/internal/handlers"
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

	layout := components.LayoutData{
		Title:       "Help",
		CurrentPath: "/help",
		User:        userInfoFromContext(r),
	}

	HelpPage(layout).Render(ctx, w)
}

func (h *Handler) About(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	layout := components.LayoutData{
		Title:       "About",
		CurrentPath: "/about",
		User:        userInfoFromContext(r),
	}

	AboutPage(layout).Render(ctx, w)
}

func (h *Handler) Privacy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	layout := components.LayoutData{
		Title:       "Privacy",
		CurrentPath: "/privacy",
		User:        userInfoFromContext(r),
	}

	PrivacyPage(layout).Render(ctx, w)
}

func userInfoFromContext(r *http.Request) *components.UserInfo {
	u := handlers.UserFromContext(r.Context())
	if u == nil {
		return nil
	}
	return &components.UserInfo{
		Username:  u.Username,
		AvatarURL: u.AvatarURL,
	}
}
