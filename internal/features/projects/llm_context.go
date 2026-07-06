package projects

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"july/internal/db"
	"july/internal/metrics"
	"july/internal/shared"
)

type metricLLMContextResponse struct {
	MetricType string `json:"metricType"`
	RepoName   string `json:"repoName"`
	L1Score    int16  `json:"l1Score"`
	Level      int16  `json:"level"`
	SHA        string `json:"sha"`
	// SystemPrompt and UserPrompt are consumed by the browser WebLLM (chat.completions).
	SystemPrompt string `json:"systemPrompt"`
	UserPrompt   string `json:"userPrompt"`
}

// metricLLMErrorResponse is returned with 4xx for GET …/llm-context when AI context cannot be built.
type metricLLMErrorResponse struct {
	Error         string `json:"error"`
	Message       string `json:"message"`
	MetricType    string `json:"metricType"`
	MetricName    string `json:"metricName"`
	L1Score       *int16 `json:"l1Score,omitempty"`
	HelpURL       string `json:"helpUrl"`
	MetricHelpURL string `json:"metricHelpUrl"`
}

func writeMetricLLMJSONError(w http.ResponseWriter, r *http.Request, status int, body metricLLMErrorResponse) {
	shared.RespondJSON(w, r, status, body)
}

// GET /api/projects/{projectID}/analysis/metrics/{metricType}/llm-context
// Returns a compact, metric-focused prompt bundle for browser-side WebLLM (after server L1 exists).
// Public (unauthenticated) access is allowed.
func (h *projectHandler) GetProjectMetricLLMContext(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := r.PathValue("projectID")
	metricType := r.PathValue("metricType")
	logger := log.Ctx(ctx)

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		logger.Warn().Str("project", projectID).Msg("bad project uuid")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	project, err := h.queries.GetProjectByID(ctx, projectUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		logger.Warn().Str("project", projectID).Msg("unknown project")
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		logger.Error().Err(err).Str("project_id", projectID).Msg("get project")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if !isKnownMetricType(metricType) {
		logger.Warn().Str("type", metricType).Msg("unknown metric type")
		http.Error(w, "bad request: unknown metric type", http.StatusBadRequest)
		return
	}

	// Attempt to fetch L1 metric data; fall back to README if unavailable.
	var data map[string]any
	var score int16
	var level int16
	var sha string

	row, err := h.queries.GetAnalysisMetric(ctx, db.GetAnalysisMetricParams{
		ProjectID:  projectUUID,
		MetricType: metricType,
	})
	if err == nil && row.Data != nil && row.Score > 0 {
		data = row.Data
		score = row.Score
		level = row.Level
		sha = row.Sha
	} else {
		// No L1 data for this metric: use Parse which returns
		// an all-false zero-value struct for unknown types.
		score = 0
		level = 0
		sha = ""
		m, _ := metrics.Parse(metricType, nil)
		b, _ := json.Marshal(m)
		json.Unmarshal(b, &data)
	}

	userPrompt := metrics.BuildMetricLLMUserContent(metricType, data, project.Url, score, level)
	shared.RespondJSON(w, r, http.StatusOK, metricLLMContextResponse{
		MetricType:   metricType,
		RepoName:     project.Url,
		L1Score:      score,
		Level:        level,
		SHA:          sha,
		SystemPrompt: metrics.MetricLLMSystemPrompt,
		UserPrompt:   userPrompt,
	})
}

func isKnownMetricType(s string) bool {
	for _, m := range metrics.MetricOrder {
		if m == s {
			return true
		}
	}
	return false
}
