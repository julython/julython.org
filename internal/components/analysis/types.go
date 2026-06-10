package analysis

// AnalysisTile represents a single analysis metric tile.
type AnalysisTile struct {
	MetricKey string
	Level     int16
	Score     int16
	I18nKey   string
}

// BoardStats represents game board activity stats for a project.
type BoardStats struct {
	CommitCount      int
	ContributorCount int
}

// Max points for the analysis board (8 metrics × 60 pts each).
const AnalysisBoardMaxPts = 480
