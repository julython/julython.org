package game

import "july/internal/components/piechart"

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

// HomeData holds data for the home page.
type HomeData struct {
	Game              GameStats
	DailyCommits      []DayCommits
	LanguageBreakdown []piechart.DataPoint
	RecentCommits     []RecentCommit
	MaxDayCommits     int
}
