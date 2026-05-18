package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

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

		respondJSON(w, r, http.StatusOK, analysisResponse{
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

	respondJSON(w, r, http.StatusOK, analysisResponse{
		MetricType: saved.MetricType,
		Score:      saved.Score,
		Level:      saved.Level,
	})
}
