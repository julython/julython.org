package layout

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
