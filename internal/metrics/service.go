package metrics

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"july/internal/db"
	"july/internal/services"
)

// Scanner runs server-side L1 analysis and upserts analysis_metrics rows.
type Scanner struct {
	queries     *db.Queries
	pool        *pgxpool.Pool
	gameService *services.GameService
	token       string
}

// NewScanner wires the app-wide queries handle and pool from the composition root (same as api.buildMux).
// Pool is used only for transactions; queries is used for pool-scoped SQL (e.g. SetProjectIsPrivate).
func NewScanner(queries *db.Queries, pool *pgxpool.Pool, gs *services.GameService, token string) *Scanner {
	return &Scanner{queries: queries, pool: pool, gameService: gs, token: token}
}

// IsConfigured returns true when a non-empty GitHub token was provided (public-repo reads).
func (s *Scanner) IsConfigured() bool {
	return strings.TrimSpace(s.token) != ""
}

// MetricOrder matches analysisBoardSpec and Evaluate keys.
var MetricOrder = []string{
	"readme", "tests", "ci", "structure", "linting", "deps", "docs", "ai_ready",
}

// ParseGitHubOwnerRepo extracts owner/repo from a github.com repository URL.
func ParseGitHubOwnerRepo(repoURL string) (owner, repo string, err error) {
	u := strings.TrimSpace(repoURL)
	u = strings.TrimPrefix(u, "https://github.com/")
	u = strings.TrimPrefix(u, "http://github.com/")
	u = strings.TrimSuffix(strings.TrimSuffix(u, ".git"), "/")
	parts := strings.SplitN(u, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("cannot parse GitHub owner/repo from URL")
	}
	return parts[0], parts[1], nil
}

// RunScan fetches the repo tree, scores metrics, and upserts all eight rows in one transaction.
// No-op when token is empty, project is private, or service is not github.
func (s *Scanner) RunScan(ctx context.Context, project db.Project, updatedBy uuid.UUID) error {
	if project.Service != "github" {
		return nil
	}
	if project.IsPrivate {
		return nil
	}
	if strings.TrimSpace(s.token) == "" {
		return nil
	}

	owner, repoName, err := ParseGitHubOwnerRepo(project.Url)
	if err != nil {
		return err
	}

	client := NewClient(s.token)
	res, err := client.FetchScanData(ctx, owner, repoName)
	if err != nil {
		if err.Error() == "github: forbidden" {
			if err := s.queries.SetProjectIsPrivate(ctx, db.SetProjectIsPrivateParams{
				ID:        project.ID,
				IsPrivate: true,
			}); err != nil {
				return fmt.Errorf("set project private after GitHub 403: %w", err)
			}
			log.Info().Str("slug", project.Slug).Str("id", project.ID.String()).Msg("marked project private after GitHub 403 during L1 scan")
			return nil
		}
		return fmt.Errorf("fetch L1 scan: %w", err)
	}

	results := Evaluate(res)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := db.New(tx)
	for _, metricType := range MetricOrder {
		mr, ok := results[metricType]
		if !ok {
			continue
		}
		data := db.JSONMap(mr.Data)
		if err := qtx.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID:         db.NewID(),
			ProjectID:  project.ID,
			MetricType: metricType,
			Score:      mr.Score,
			Level:      CalculateLevel(mr.Score),
			Data:       data,
			Sha:        res.SHA,
			UpdatedBy:  updatedBy,
		}); err != nil {
			return fmt.Errorf("upsert metric %s: %w", metricType, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// Update verified_points for all boards of this project.
	totalScore := 0
	for _, mr := range results {
		// TODO(rmyers): we need to add level to the scoring
		totalScore += int(mr.Score * 1 * 2)
	}
	if totalScore > 0 {
		if err := s.queries.UpdateBoardVerifiedPointsByProjectID(ctx, db.UpdateBoardVerifiedPointsByProjectIDParams{
			ProjectID:      project.ID,
			VerifiedPoints: int32(totalScore),
		}); err != nil {
			log.Warn().Err(err).Str("project", project.Slug).Msg("failed to update board verified_points")
		}

		// Refresh the player who owns this project's board.
		game, err := s.gameService.GetActiveGame(ctx)
		if err == nil {
			if err := s.gameService.RefreshPlayerAfterScan(ctx, game.ID, project.ID); err != nil {
				log.Warn().Err(err).Str("project", project.Slug).Msg("failed to refresh player after scan")
			}
		}
	}

	return nil
}
