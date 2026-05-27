package projects

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"july/internal/auth"
	"july/internal/db"
	"july/internal/metrics"
	"july/internal/services"
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
func (h *ProjectHandler) GetProjectMetricLLMContext(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := r.PathValue("projectID")
	metricType := r.PathValue("metricType")

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

	if !isKnownMetricType(metricType) {
		http.Error(w, "bad request: unknown metric type", http.StatusBadRequest)
		return
	}

	if project.Service != "github" {
		http.Error(w, "bad request: metric LLM context is only available for GitHub projects", http.StatusBadRequest)
		return
	}

	owner, repoName, err := services.ParseGitHubOwnerRepo(project.Url)
	if err != nil {
		http.Error(w, "bad request: project URL is not a GitHub repo", http.StatusBadRequest)
		return
	}
	repoFull := owner + "/" + repoName

	row, err := h.queries.GetAnalysisMetric(ctx, db.GetAnalysisMetricParams{
		ProjectID:  projectUUID,
		MetricType: metricType,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeMetricLLMJSONError(w, r, http.StatusBadRequest, metricLLMErrorResponse{
			Error:         "metric_no_analysis",
			Message:       "There is no L1 analysis row for this metric yet. Run **Rescan analysis (L1)** on the project page first, then try again.",
			MetricType:    metricType,
			MetricName:    metrics.MetricDisplayName(metricType),
			HelpURL:       "/help#analysis-metrics",
			MetricHelpURL: "/help#" + metrics.MetricHelpAnchor(metricType),
		})
		return
	}
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("get analysis metric for llm context")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if row.Score <= 0 {
		score := row.Score
		name := metrics.MetricDisplayName(metricType)
		writeMetricLLMJSONError(w, r, http.StatusBadRequest, metricLLMErrorResponse{
			Error:         "metric_l1_zero",
			Message:       fmt.Sprintf("**%s** has an L1 score of %d/10. The browser AI review needs at least minimal evidence from the server scan (a non-zero score). Improve this area of the repository, then run **Rescan analysis (L1)** on the project page and try again.", name, row.Score),
			MetricType:    metricType,
			MetricName:    name,
			L1Score:       &score,
			HelpURL:       "/help#analysis-metrics",
			MetricHelpURL: "/help#" + metrics.MetricHelpAnchor(metricType),
		})
		return
	}

	var data map[string]any
	if row.Data != nil {
		data = row.Data
	} else {
		data = map[string]any{}
	}

	userPrompt := metrics.BuildMetricLLMUserContent(metricType, data, repoFull, row.Score, row.Level)
	shared.RespondJSON(w, r, http.StatusOK, metricLLMContextResponse{
		MetricType:   metricType,
		RepoName:     repoFull,
		L1Score:      row.Score,
		Level:        row.Level,
		SHA:          row.Sha,
		SystemPrompt: metrics.MetricLLMSystemPrompt,
		UserPrompt:   userPrompt,
	})
}

func isKnownMetricType(s string) bool {
	for _, m := range services.MetricOrder {
		if m == s {
			return true
		}
	}
	return false
}
