package handlers

import (
	"net/http"

	"july/internal/components"
)

type HelpHandler struct{}

func NewHelpHandler() *HelpHandler {
	return &HelpHandler{}
}

func (h *HelpHandler) Help(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUserFromContext(r)

	layout := components.LayoutData{
		Title:       "Help",
		CurrentPath: "/help",
		User:        user,
	}

	components.HelpPage(layout).Render(ctx, w)
}

func (h *HelpHandler) About(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUserFromContext(r)

	layout := components.LayoutData{
		Title:       "About",
		CurrentPath: "/about",
		User:        user,
	}

	components.AboutPage(layout).Render(ctx, w)
}

func (h *HelpHandler) Privacy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := getUserFromContext(r)

	layout := components.LayoutData{
		Title:       "Privacy",
		CurrentPath: "/privacy",
		User:        user,
	}

	components.PrivacyPage(layout).Render(ctx, w)
}
