package projects

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/invopop/ctxi18n/i18n"
	"github.com/jackc/pgx/v5"

	"july/internal/components/analysis"
	"july/internal/db"
	"july/internal/shared"
)

// ProjectService holds dependencies for project-detail queries.
type ProjectService struct {
	Queries *db.Queries
}

// NewProjectService creates a new ProjectService.
func NewProjectService(q *db.Queries) *ProjectService {
	return &ProjectService{Queries: q}
}

// BuildAnalysisBoard fetches analysis metrics for a project and builds the
// tiles, scoring totals, and last-analyzed info used by the detail page and
// the player board page.
func (s *ProjectService) BuildAnalysisBoard(ctx context.Context, projectID uuid.UUID) (ProjectAnalysisBoard, error) {
	analysisRows, err := s.Queries.GetAnalysisMetricsByProject(ctx, projectID)
	if err != nil {
		return ProjectAnalysisBoard{}, err
	}

	// analysisSpec defines the order and i18n keys for analysis tiles.
	var analysisBoardSpec = []struct {
		key     string
		i18nKey string
	}{
		{"readme", i18n.T(ctx, "projects.MetricReadme")},
		{"tests", i18n.T(ctx, "projects.MetricTests")},
		{"ci", i18n.T(ctx, "projects.MetricCI")},
		{"structure", i18n.T(ctx, "projects.MetricStructure")},
		{"linting", i18n.T(ctx, "projects.MetricLinting")},
		{"deps", i18n.T(ctx, "projects.MetricDeps")},
		{"docs", i18n.T(ctx, "projects.MetricDocs")},
		{"ai_ready", i18n.T(ctx, "projects.MetricAIReady")},
	}

	levelByType := make(map[string]int16, len(analysisRows))
	scoreByType := make(map[string]int16, len(analysisRows))
	for _, row := range analysisRows {
		levelByType[row.MetricType] = row.Level
		scoreByType[row.MetricType] = row.Score
	}

	tiles := make([]analysis.AnalysisTile, 0, len(analysisBoardSpec))
	earned := 0
	for _, spec := range analysisBoardSpec {
		level := levelByType[spec.key]
		if level < 0 {
			level = 0
		}
		if level > 3 {
			level = 3
		}
		score := scoreByType[spec.key]
		// Points align score (0–10) with level (0–3): max 10*3*2 = 60 per metric.
		earned += int(score) * int(level) * 2
		tiles = append(tiles, analysis.AnalysisTile{
			MetricKey: spec.key,
			Level:     level,
			Score:     score,
			I18nKey:   spec.i18nKey,
		})
	}

	shaDistinct := make(map[string]struct{})
	var lastMetricAt time.Time
	var haveMetricAt bool
	for _, row := range analysisRows {
		if row.Sha != "" {
			shaDistinct[row.Sha] = struct{}{}
		}
		if !haveMetricAt || row.UpdatedAt.After(lastMetricAt) {
			lastMetricAt = row.UpdatedAt
			haveMetricAt = true
		}
	}

	board := ProjectAnalysisBoard{
		Tiles:            tiles,
		EarnedPts:        earned,
		MaxPts:           analysis.AnalysisBoardMaxPts,
		AnalysisRunCount: len(shaDistinct),
	}
	if haveMetricAt {
		board.LastAnalyzedAgo = shared.TimeAgo(lastMetricAt)
	}
	return board, nil
}

// GameActivitySummary fetches game-scoped activity aggregates and board stats
// for a given project and game.
func (s *ProjectService) GameActivitySummary(ctx context.Context, projectID, gameID uuid.UUID) (ProjectGameActivitySummary, error) {
	result := ProjectGameActivitySummary{HasGame: false}
	if gameID == uuid.Nil {
		return result, nil
	}

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	weekStart := now.Add(-7 * 24 * time.Hour)

	agg, err := s.Queries.GetProjectGameActivityAggregates(ctx, db.GetProjectGameActivityAggregatesParams{
		ProjectID:  projectID,
		GameID:     db.UUID(gameID),
		WeekStart:  weekStart,
		MonthStart: monthStart,
	})
	if err != nil {
		return result, nil
	}

	result.HasGame = true
	result.CommitsThisMonth = int(agg.CommitsThisMonth)
	result.CommitsThisWeek = int(agg.CommitsThisWeek)
	result.FileTouchCount = int(agg.FileTouchCount)
	result.UniqueDirs = int(agg.UniqueDirs)

	board, bErr := s.Queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
		ProjectID: projectID,
		GameID:    gameID,
	})
	if bErr == nil {
		result.Board = &analysis.BoardStats{
			CommitCount:      int(board.CommitCount),
			ContributorCount: int(board.ContributorCount),
		}
	} else if !errors.Is(bErr, pgx.ErrNoRows) {
		return result, bErr
	}
	return result, nil
}

// BuildProjectBoardInfo fetches the info needed to render a single board
// card on the player page: analysis tiles and game activity.
func (s *ProjectService) BuildProjectBoardInfo(ctx context.Context, projectID, gameID uuid.UUID) (ProjectDetailData, error) {
	analysisBoard, err := s.BuildAnalysisBoard(ctx, projectID)
	if err != nil {
		return ProjectDetailData{}, err
	}

	gameActivity, err := s.GameActivitySummary(ctx, projectID, gameID)
	if err != nil {
		return ProjectDetailData{}, err
	}

	return ProjectDetailData{
		AnalysisBoard: analysisBoard,
		GameActivity:  gameActivity,
	}, nil
}
