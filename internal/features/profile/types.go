package profile

type ProfileSection string

const (
	SectionOverview ProfileSection = "overview"
	SectionWebhooks ProfileSection = "webhooks"
	SectionSettings ProfileSection = "settings"
)

type OverviewData struct {
	Username  string
	Name      string
	AvatarURL string
	Emails    []string
}

type SettingsData struct {
	Name    string
	Success bool
	Error   string
}
