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

var (
	errL1PrivateRepo   = errors.New("l1: private repository")
	errL1NoGitHubToken = errors.New("l1: GITHUB_TOKEN not configured")
)

// performL1Scan runs server-side L1 (push webhook and manual rescan use this path).
func (h *ProjectHandler) performL1Scan(ctx context.Context, project db.Project, updatedBy uuid.UUID) error {
	if project.IsPrivate {
		return errL1PrivateRepo
	}
	if !h.l1Scanner.IsConfigured() {
		return errL1NoGitHubToken
	}

	scanCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if err := h.l1Scanner.RunL1Scan(scanCtx, project, updatedBy); err != nil {
		return err
	}
	return nil
}

// POST /projects/{slug}/analysis/l1
// HTMX-triggered L1 rescan from the project page; redirects back on success.
func (h *ProjectHandler) PostProjectRescanL1(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slug := r.PathValue("slug")

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

	project, err := h.queries.GetProjectBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("slug", slug).Msg("get project for rescan")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if !canEditProject(&user, project) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	err = h.performL1Scan(ctx, project, user.ID)
	if errors.Is(err, errL1PrivateRepo) {
		http.Error(w, "server L1 is not available for private repositories", http.StatusBadRequest)
		return
	}
	if errors.Is(err, errL1NoGitHubToken) {
		http.Error(w, "GITHUB_TOKEN is not configured", http.StatusServiceUnavailable)
		return
	}
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("slug", slug).Msg("L1 scan")
		http.Error(w, "L1 scan failed", http.StatusBadGateway)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", fmt.Sprintf("/projects/%s", slug))
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/projects/%s", slug), http.StatusSeeOther)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

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

func writeMetricLLMJSONError(w http.ResponseWriter, status int, body metricLLMErrorResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// GET /api/projects/{projectID}/analysis/metrics/{metricType}/llm-context
// Returns a compact, metric-focused prompt bundle for browser-side WebLLM (after server L1 exists).
func (h *ProjectHandler) GetProjectMetricLLMContext(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := r.PathValue("projectID")
	metricType := r.PathValue("metricType")

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
		writeMetricLLMJSONError(w, http.StatusBadRequest, metricLLMErrorResponse{
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
		writeMetricLLMJSONError(w, http.StatusBadRequest, metricLLMErrorResponse{
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
	writeJSON(w, metricLLMContextResponse{
		MetricType:   metricType,
		RepoName:     repoFull,
		L1Score:      row.Score,
		Level:        row.Level,
		SHA:          row.Sha,
		SystemPrompt: metrics.MetricLLMSystemPrompt,
		UserPrompt:   userPrompt,
	})
}

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
			// No L1 row for this topic: do not attach README file content as if it were CI/tests/etc.
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
	writeJSON(w, chatContextResponse{
		SystemPrompt:      metrics.ChatExpertSystemPrompt(info),
		UserPrompt:        userPrompt,
		MatchedMetric:     info.MatchedKeyword,
		ContextMetric:     info.TopicMetric,
		UsedDefaultReadme: info.UsedDefaultTopic,
		FallbackToReadme:  false,
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
