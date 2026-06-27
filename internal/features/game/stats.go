package game

// GameStats holds aggregated game statistics.
type GameStats struct {
	Name          string
	TotalCommits  int
	TotalUsers    int
	TotalProjects int
}

// DayCommits holds commit count for a single day.
type DayCommits struct {
	Day   int
	Count int
}

// RecentCommit is a shared type used by multiple features (home, activity).
type RecentCommit struct {
	Username    string
	Name        string // Display name for avatar initials
	Author      string
	AvatarURL   string
	Message     string
	Project     string
	ProjectName string
	TimeAgo     string
}
