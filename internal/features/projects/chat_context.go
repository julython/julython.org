package projects

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"july/internal/auth"
	"july/internal/db"
	"july/internal/metrics"
	"july/internal/services"
	"july/internal/shared"
)

type chatContextRequest struct {
	Message string `json:"message"`
}

type chatContextResponse struct {
	SystemPrompt      string `json:"systemPrompt"`
	UserPrompt        string `json:"userPrompt"`
	MatchedMetric     string `json:"matchedMetric,omitempty"`
	ContextMetric     string `json:"contextMetric"`
	UsedDefaultReadme bool   `json:"usedDefaultReadme"`
	FallbackToReadme  bool   `json:"fallbackToReadme"`
}

// POST /api/projects/{projectID}/analysis/chat-context
// Returns system + user prompts for the browser assistant: keyword-matched metric scan data, or README by default.
func (h *ProjectHandler) PostProjectChatContext(w http.ResponseWriter, r *http.Request) {
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

	if project.Service != "github" {
		http.Error(w, "bad request: chat context is only available for GitHub projects", http.StatusBadRequest)
		return
	}

	owner, repoName, err := services.ParseGitHubOwnerRepo(project.Url)
	if err != nil {
		http.Error(w, "bad request: project URL is not a GitHub repo", http.StatusBadRequest)
		return
	}
	repoFull := owner + "/" + repoName

	var reqBody chatContextRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	msg := strings.TrimSpace(reqBody.Message)
	if msg == "" {
		http.Error(w, "bad request: message required", http.StatusBadRequest)
		return
	}

	matchedMetric, keywordOK := metrics.MatchMetricFromMessage(msg)
	info := metrics.ChatContextInfo{
		TopicMetric:      "readme",
		UsedDefaultTopic: !keywordOK,
	}
	if keywordOK {
		info.KeywordMatched = true
		info.MatchedKeyword = matchedMetric
		info.TopicMetric = matchedMetric
	}

	var data map[string]any

	if keywordOK {
		row, err1 := h.queries.GetAnalysisMetric(ctx, db.GetAnalysisMetricParams{
			ProjectID:  projectUUID,
			MetricType: matchedMetric,
		})
		if err1 == nil && row.Data != nil {
			data = row.Data
			info.NoScanEvidence = false
		} else {
			// No L1 row for this topic: fall back to readme and do not attach
			// README file content as if it were CI/tests/etc.
			info.TopicMetric = "readme"
			data = map[string]any{}
			info.NoScanEvidence = true
			rowRm, errRm := h.queries.GetAnalysisMetric(ctx, db.GetAnalysisMetricParams{
				ProjectID:  projectUUID,
				MetricType: "readme",
			})
			if errRm == nil && rowRm.Data != nil {
				info.PrimaryLanguage = metrics.LanguageFromData(rowRm.Data)
			}
		}
	} else {
		row, errR := h.queries.GetAnalysisMetric(ctx, db.GetAnalysisMetricParams{
			ProjectID:  projectUUID,
			MetricType: "readme",
		})
		if errR == nil && row.Data != nil {
			data = row.Data
			info.NoScanEvidence = false
		} else {
			data = map[string]any{}
			info.NoScanEvidence = true
		}
	}

	if info.PrimaryLanguage == "" {
		info.PrimaryLanguage = metrics.LanguageFromData(data)
	}
	if info.PrimaryLanguage == "" {
		rowRm, errRm := h.queries.GetAnalysisMetric(ctx, db.GetAnalysisMetricParams{
			ProjectID:  projectUUID,
			MetricType: "readme",
		})
		if errRm == nil && rowRm.Data != nil {
			info.PrimaryLanguage = metrics.LanguageFromData(rowRm.Data)
		}
	}

	info.GeneralChat = metrics.IsGenericChatMessage(msg)
	userPrompt := metrics.BuildChatLLMUserContent(repoFull, info, data, msg)
	shared.RespondJSON(w, r, http.StatusOK, chatContextResponse{
		SystemPrompt:      metrics.ChatExpertSystemPrompt(info),
		UserPrompt:        userPrompt,
		MatchedMetric:     info.MatchedKeyword,
		ContextMetric:     info.TopicMetric,
		UsedDefaultReadme: info.UsedDefaultTopic,
		FallbackToReadme:  false,
	})
}
