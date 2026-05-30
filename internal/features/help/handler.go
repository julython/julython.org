package help

import (
	"net/http"

	"july/internal/components/layout"
)

// Register mounts help, about, and privacy routes on the given mux.
func Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /help", Help)
	mux.HandleFunc("GET /about", About)
	mux.HandleFunc("GET /privacy", Privacy)
}

func Help(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	layout := layout.LayoutData{
		Title:       "Help",
		CurrentPath: "/help",
		User:        layout.UserInfoFromContext(r),
	}

	HelpPage(layout).Render(ctx, w)
}

func About(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	layout := layout.LayoutData{
		Title:       "About",
		CurrentPath: "/about",
		User:        layout.UserInfoFromContext(r),
	}

	AboutPage(layout).Render(ctx, w)
}

func Privacy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	layout := layout.LayoutData{
		Title:       "Privacy",
		CurrentPath: "/privacy",
		User:        layout.UserInfoFromContext(r),
	}

	PrivacyPage(layout).Render(ctx, w)
}
