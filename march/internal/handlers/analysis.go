package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"july/internal/db"
	"july/internal/metrics"
	"july/internal/services"
)

type analysisPayload struct {
	SHA        string          `json:"sha"`
	MetricType string          `json:"metricType"`
	Level      int16           `json:"level"`
	Data       json.RawMessage `json:"data"`
}

type analysisResponse struct {
	MetricType string `json:"metricType"`
	Score      int16  `json:"score"`
	Level      int16  `json:"level"`
}

// POST /api/projects/{projectID}/analysis
// Level 0/1: scores the tile and upserts the metric row (heuristic L1 when score > 0).
// Level 2/3: records an AI-graded level up; requires existing heuristic row with score > 0.
func (h *ProjectHandler) PostProjectAnalysis(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := r.PathValue("projectID")

	sessionUser := UserFromContext(ctx)
	if sessionUser == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := h.userService.FindByID(ctx, sessionUser.ID)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	project, err := h.queries.GetProjectByID(ctx, projectUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("project_id", projectID).Msg("get project")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if !canEditProject(&user, project) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var p analysisPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	logger := log.Ctx(ctx).
		With().
		Str("SHA", p.SHA).
		Str("Metric", p.MetricType).
		Str("projectID", projectID).
		Int16("level", p.Level).
		Logger()

	if p.SHA == "" || p.MetricType == "" || len(p.Data) == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if p.Level < 0 || p.Level > 3 {
		http.Error(w, "bad request: level must be 0 to 3", http.StatusBadRequest)
		return
	}

	// L2/L3 path: AI grading — gate on stored heuristic score > 0 (any partial L1), not 10/10.
	if p.Level >= 2 {
		existing, err := h.queries.GetAnalysisMetric(ctx, db.GetAnalysisMetricParams{
			ProjectID:  projectUUID,
			MetricType: p.MetricType,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "bad request: metric must have heuristic data (L1) before AI grading", http.StatusBadRequest)
			return
		}
		if err != nil {
			logger.Error().Err(err).Msg("get analysis metric for level guard")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if existing.Score <= 0 || existing.Level < 1 {
			http.Error(w, "bad request: metric must have heuristic data (L1) before AI grading", http.StatusBadRequest)
			return
		}

		if err := h.queries.UpdateAnalysisMetricLevel(ctx, db.UpdateAnalysisMetricLevelParams{
			ProjectID:  projectUUID,
			MetricType: p.MetricType,
			Level:      p.Level,
			UpdatedBy:  user.ID,
		}); err != nil {
			logger.Error().Err(err).Msg("update analysis metric level")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, analysisResponse{
			MetricType: p.MetricType,
			Score:      existing.Score,
			Level:      p.Level,
		})
		return
	}

	m, err := metrics.Parse(p.MetricType, p.Data)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	score := metrics.Score(m)

	var data db.JSONMap
	json.Unmarshal(p.Data, &data)

	if err := h.queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
		ID:         db.NewID(),
		ProjectID:  projectUUID,
		MetricType: p.MetricType,
		Score:      score,
		Data:       data,
		Sha:        p.SHA,
		UpdatedBy:  user.ID,
	}); err != nil {
		logger.Error().Err(err).Msg("upsert analysis metric")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Fetch the persisted row so the response reflects the true level
	// (the SQL upsert owns the L0/L1 transition logic).
	saved, err := h.queries.GetAnalysisMetric(ctx, db.GetAnalysisMetricParams{
		ProjectID:  projectUUID,
		MetricType: p.MetricType,
	})
	if err != nil {
		logger.Error().Err(err).Msg("get analysis metric after upsert")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, analysisResponse{
		MetricType: saved.MetricType,
		Score:      saved.Score,
		Level:      saved.Level,
	})
}

type l1ScanMetricRow struct {
	MetricType string `json:"metricType"`
	Score      int16  `json:"score"`
	Level      int16  `json:"level"`
}

type l1ScanResponse struct {
	SHA     string            `json:"sha"`
	Metrics []l1ScanMetricRow `json:"metrics"`
}

// POST /api/projects/{projectID}/analysis/l1
// Runs server-side L1 analysis and upserts all eight metric rows (public GitHub repos only).
func (h *ProjectHandler) PostProjectL1Scan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := r.PathValue("projectID")

	sessionUser := UserFromContext(ctx)
	if sessionUser == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := h.userService.FindByID(ctx, sessionUser.ID)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	project, err := h.queries.GetProjectByID(ctx, projectUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("project_id", projectID).Msg("get project")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if !canEditProject(&user, project) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if project.IsPrivate {
		http.Error(w, "server L1 is not available for private repositories", http.StatusBadRequest)
		return
	}

	if h.githubToken == "" {
		http.Error(w, "GITHUB_TOKEN is not configured", http.StatusServiceUnavailable)
		return
	}

	l1 := services.NewL1Scanner(h.pool, h.githubToken)
	scanCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if err := l1.RunL1Scan(scanCtx, project, user.ID); err != nil {
		log.Ctx(ctx).Error().Err(err).Str("project_id", projectID).Msg("L1 scan")
		http.Error(w, "L1 scan failed", http.StatusBadGateway)
		return
	}

	rows, err := h.queries.GetAnalysisMetricsByProject(ctx, project.ID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("get analysis metrics after L1")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var sha string
	metricsOut := make([]l1ScanMetricRow, 0, len(rows))
	for _, row := range rows {
		if sha == "" && row.Sha != "" {
			sha = row.Sha
		}
		metricsOut = append(metricsOut, l1ScanMetricRow{
			MetricType: row.MetricType,
			Score:      row.Score,
			Level:      row.Level,
		})
	}

	writeJSON(w, l1ScanResponse{SHA: sha, Metrics: metricsOut})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// canEditProject returns true if the user is an admin or the slug owner.
// Slug format: {service_prefix}-{username}-{repo...}
// e.g. "gh-rmyers-my-cool-repo" → owner is "rmyers"
func canEditProject(user *db.User, project db.Project) bool {
	if user.Role == "admin" {
		return true
	}
	owner, err := slugOwner(project.Slug)
	if err != nil {
		return false
	}
	return strings.EqualFold(user.Username, owner)
}

// slugOwner parses the owner username out of a project slug.
// Returns an error if the slug doesn't contain at least a prefix and owner.
func slugOwner(slug string) (string, error) {
	parts := strings.SplitN(slug, "-", 3)
	if len(parts) < 3 {
		return "", fmt.Errorf("slug %q has no owner segment", slug)
	}
	return parts[1], nil
}
