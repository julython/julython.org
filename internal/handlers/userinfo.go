package handlers

import (
	"july/internal/components/layout"
	"net/http"
)

// getUserFromContext wraps handlers.UserFromContext and converts to *layout.UserInfo.
// This is used by handlers that haven't yet been migrated to the features/ layout.
func getUserFromContext(r *http.Request) *layout.UserInfo {
	u := UserFromContext(r.Context())
	if u == nil {
		return nil
	}
	return &layout.UserInfo{
		Username:  u.Username,
		AvatarURL: u.AvatarURL,
	}
}
