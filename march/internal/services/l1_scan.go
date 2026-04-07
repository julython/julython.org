package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"july/internal/db"
	"july/internal/metrics"
)

// L1Scanner runs server-side L1 analysis and upserts analysis_metrics rows.
type L1Scanner struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	token   string
}

// NewL1Scanner wires the app-wide queries handle and pool from the composition root (same as api.buildMux).
// Pool is used only for transactions; queries is used for pool-scoped SQL (e.g. SetProjectIsPrivate).
func NewL1Scanner(queries *db.Queries, pool *pgxpool.Pool, token string) *L1Scanner {
	return &L1Scanner{queries: queries, pool: pool, token: token}
}

// IsConfigured returns true when a non-empty GitHub token was provided (public-repo reads).
func (s *L1Scanner) IsConfigured() bool {
	return strings.TrimSpace(s.token) != ""
}

// MetricOrder matches analysisBoardSpec and EvaluateL1 keys.
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

// RunL1Scan fetches the repo tree, scores metrics, and upserts all eight rows in one transaction.
// No-op when token is empty, project is private, or service is not github.
func (s *L1Scanner) RunL1Scan(ctx context.Context, project db.Project, updatedBy uuid.UUID) error {
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

	client := metrics.NewClient(s.token)
	res, err := client.FetchL1Scan(ctx, owner, repoName)
	if err != nil {
		if errors.Is(err, metrics.ErrGitHubForbidden) {
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

	results := metrics.EvaluateL1(res)

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
	return nil
}
