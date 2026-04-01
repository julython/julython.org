package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"july/internal/db"
	"july/internal/metrics"
)

type analysisPayload struct {
	SHA        string          `json:"sha"`
	MetricType string          `json:"metricType"`
	Level      int16           `json:"level"`
	Data       json.RawMessage `json:"data"`
}

// POST /api/projects/{projectID}/analysis
// Level 0/1: scores the tile and upserts the metric row.
// Level 2/3: records an AI-graded level up, data contains the AI reasoning.
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

	// L1 scan — parse the typed struct and score it.
	m, err := metrics.Parse(p.MetricType, p.Data)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	score := metrics.Score(m)

	if p.Level >= 2 && score == 10 {
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
		w.WriteHeader(http.StatusNoContent)
		return
	}

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

	w.WriteHeader(http.StatusNoContent)
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
