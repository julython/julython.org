package projects

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"july/internal/auth"
	"july/internal/db"
	"july/internal/metrics"
	"july/internal/shared"
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
// Scores the metric, computes the level from the score (0→0, 1–5→1, 6–8→2, 9–10→3),
// and persists the result in one upsert.
func (h *projectHandler) PostProjectAnalysis(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := r.PathValue("projectID")

	sessionUser := auth.UserFromContext(ctx)
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
		Logger()

	if p.SHA == "" || p.MetricType == "" || len(p.Data) == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	m, err := metrics.Parse(p.MetricType, p.Data)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	score := metrics.Score(m)
	level := metrics.CalculateLevel(score)

	var data db.JSONMap
	json.Unmarshal(p.Data, &data)

	if err := h.queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
		ID:         db.NewID(),
		ProjectID:  projectUUID,
		MetricType: p.MetricType,
		Level:      level,
		Score:      score,
		Data:       data,
		Sha:        p.SHA,
		UpdatedBy:  user.ID,
	}); err != nil {
		logger.Error().Err(err).Msg("upsert analysis metric")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Fetch the persisted row so the response reflects the true level.
	saved, err := h.queries.GetAnalysisMetric(ctx, db.GetAnalysisMetricParams{
		ProjectID:  projectUUID,
		MetricType: p.MetricType,
	})
	if err != nil {
		logger.Error().Err(err).Msg("get analysis metric after upsert")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	shared.RespondJSON(w, r, http.StatusOK, analysisResponse{
		MetricType: saved.MetricType,
		Score:      saved.Score,
		Level:      saved.Level,
	})
}
