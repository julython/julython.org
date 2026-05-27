package layout

import (
	"july/internal/auth"
	"net/http"
)

type NavItem struct {
	Label string
	URL   string
}

type LayoutData struct {
	Title       string
	CurrentPath string
	User        *UserInfo
}

type UserInfo struct {
	Username  string
	AvatarURL string
}

func UserInfoFromContext(r *http.Request) *UserInfo {
	u := auth.UserFromContext(r.Context())
	if u == nil {
		return nil
	}
	return &UserInfo{
		Username:  u.Username,
		AvatarURL: u.AvatarURL,
	}
}
