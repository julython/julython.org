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
