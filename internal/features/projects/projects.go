package projects

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/invopop/ctxi18n/i18n"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"july/internal/auth"
	"july/internal/components/analysis"
	"july/internal/components/layout"
	"july/internal/db"
	"july/internal/metrics"
	"july/internal/services"
	"july/internal/shared"
)

type projectHandler struct {
	queries     *db.Queries
	gameService *services.GameService
	userService *services.UserService
	scanner     *metrics.Scanner
}

// Register mounts all project routes on the given mux.
func Register(mux *http.ServeMux, q *db.Queries, gs *services.GameService, us *services.UserService, l1 *metrics.Scanner) {
	h := &projectHandler{queries: q, gameService: gs, userService: us, scanner: l1}
	mux.HandleFunc("GET /projects", h.List)
	mux.HandleFunc("GET /projects/{slug}", h.Detail)
	mux.HandleFunc("POST /projects/{slug}/analysis/l1", h.PostProjectRescanL1)
	mux.HandleFunc("POST /api/projects/{projectID}/analysis", h.PostProjectAnalysis)
	mux.HandleFunc("POST /api/projects/{projectID}/analysis/chat-context", h.PostProjectChatContext)
	mux.HandleFunc("GET /api/projects/{projectID}/analysis/metrics/{metricType}/llm-context", h.GetProjectMetricLLMContext)
}

const projectPageSize = 25

func (h *projectHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(ctx)
	q := r.URL.Query()

	search := q.Get("search")
	service := q.Get("service")
	cursorStr := q.Get("cursor")
	logger.Info().Str("search", search).Str("service", service).Str("cursor", cursorStr).Msg("GET /projects")

	params := db.SearchActiveProjectsParams{
		LimitCount: int32(projectPageSize + 1),
	}
	if search != "" {
		params.Search = db.Text(search)
	}
	if service != "" {
		params.Service = db.Text(service)
	}
	if cursorStr != "" {
		var uid pgtype.UUID
		if err := uid.Scan(cursorStr); err == nil {
			params.Cursor = db.UUID(uid.Bytes)
		}
	}

	projects, err := h.queries.SearchActiveProjects(ctx, params)
	if err != nil {
		logger.Error().Err(err).Msg("SearchActiveProjects failed")
		http.Error(w, "failed to load projects", http.StatusInternalServerError)
		return
	}
	logger.Info().Int("count", len(projects)).Msg("SearchActiveProjects returned")

	hasMore := len(projects) > projectPageSize
	if hasMore {
		projects = projects[:projectPageSize]
	}

	entries := make([]ProjectEntry, len(projects))
	for i, p := range projects {
		desc := ""
		if p.Description.Valid {
			desc = p.Description.String
		}
		entries[i] = ProjectEntry{
			ID:          p.ID.String(),
			Name:        p.Name,
			Slug:        p.Slug,
			URL:         p.Url,
			Description: desc,
			Service:     p.Service,
			Forks:       int(p.Forks),
			Watchers:    int(p.Watchers),
			Forked:      p.Forked,
		}
	}

	// Next cursor is the ID of the last item returned.
	nextCursor := ""
	if hasMore && len(entries) > 0 {
		nextCursor = entries[len(entries)-1].ID
	}

	data := ProjectListData{
		Entries:    entries,
		Search:     search,
		Service:    service,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}

	if r.Header.Get("HX-Request") == "true" {
		logger.Info().Bool("htmx", true).Int("entries", len(entries)).Msg("Rendering ProjectListItems")
		if err := ProjectListItems(data).Render(ctx, w); err != nil {
			logger.Error().Err(err).Msg("ProjectListItems render failed")
			http.Error(w, "render failed", http.StatusInternalServerError)
			return
		}
		return
	}

	logger.Info().Bool("htmx", false).Int("entries", len(entries)).Msg("Rendering ProjectListPage")
	layout := layout.LayoutData{
		Title:       "Projects",
		CurrentPath: "/projects",
		User:        layout.UserInfoFromContext(r),
	}
	if err := ProjectListPage(layout, data).Render(ctx, w); err != nil {
		logger.Error().Err(err).Msg("ProjectListPage render failed")
		http.Error(w, "render failed", http.StatusInternalServerError)
		return
	}
}

func (h *projectHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(ctx)

	slug := r.PathValue("slug")

	limit := 25
	offset := 0

	project, err := h.queries.GetProjectBySlug(ctx, slug)
	if err != nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	desc := ""
	if project.Description.Valid {
		desc = project.Description.String
	}

	entry := ProjectEntry{
		ID:          project.ID.String(),
		Name:        project.Name,
		Slug:        project.Slug,
		URL:         project.Url,
		Description: desc,
		Service:     project.Service,
		Forks:       int(project.Forks),
		Watchers:    int(project.Watchers),
		Forked:      project.Forked,
	}

	analysisRows, err := h.queries.GetAnalysisMetricsByProject(ctx, project.ID)
	if err != nil {
		logger.Error().Err(err).Msg("GetAnalysisMetricsByProject failed")
		http.Error(w, "failed to load analysis metrics", http.StatusInternalServerError)
		return
	}
	levelByType := make(map[string]int16, len(analysisRows))
	scoreByType := make(map[string]int16, len(analysisRows))
	for _, row := range analysisRows {
		levelByType[row.MetricType] = row.Level
		scoreByType[row.MetricType] = row.Score
	}

	showMetricAI := project.Service == "github" && !project.IsPrivate && h.scanner.IsConfigured()

	var analysisBoardSpec = []struct {
		key     string
		i18nKey string
	}{
		{"readme", i18n.T(ctx, "projects.MetricReadme")},
		{"tests", i18n.T(ctx, "projects.MetricTests")},
		{"ci", i18n.T(ctx, "projects.MetricCI")},
		{"structure", i18n.T(ctx, "projects.MetricStructure")},
		{"linting", i18n.T(ctx, "projects.MetricLinting")},
		{"deps", i18n.T(ctx, "projects.MetricDeps")},
		{"docs", i18n.T(ctx, "projects.MetricDocs")},
		{"ai_ready", i18n.T(ctx, "projects.MetricAIReady")},
	}

	tiles := make([]analysis.AnalysisTile, 0, len(analysisBoardSpec))
	earned := 0
	for _, spec := range analysisBoardSpec {
		level := levelByType[spec.key]
		if level < 0 {
			level = 0
		}
		if level > 3 {
			level = 3
		}
		score := scoreByType[spec.key]
		// Points align score (0–10) with level (0–3): max 10*3*2 = 60 per metric.
		earned += int(score) * int(level) * 2
		tiles = append(tiles, analysis.AnalysisTile{
			MetricKey: spec.key,
			Level:     level,
			Score:     score,
			I18nKey:   spec.i18nKey,
		})
	}
	shaDistinct := make(map[string]struct{})
	var lastMetricAt time.Time
	var haveMetricAt bool
	for _, row := range analysisRows {
		if row.Sha != "" {
			shaDistinct[row.Sha] = struct{}{}
		}
		if !haveMetricAt || row.UpdatedAt.After(lastMetricAt) {
			lastMetricAt = row.UpdatedAt
			haveMetricAt = true
		}
	}
	analysisBoard := ProjectAnalysisBoard{
		Tiles:            tiles,
		EarnedPts:        earned,
		MaxPts:           analysis.AnalysisBoardMaxPts,
		AnalysisRunCount: len(shaDistinct),
		MetricAIEnabled:  showMetricAI,
	}
	if haveMetricAt {
		analysisBoard.LastAnalyzedAgo = shared.TimeAgo(lastMetricAt)
	}
	if sess := auth.UserFromContext(ctx); sess != nil {
		if u, err := h.userService.FindByID(ctx, sess.ID); err == nil && canEditProject(&u, project) && project.Service == "github" {
			analysisBoard.RescanL1Slug = project.Slug
			switch {
			case project.IsPrivate:
				analysisBoard.RescanL1Disabled = true
				analysisBoard.RescanL1DisabledReason = "private"
			case !h.scanner.IsConfigured():
				analysisBoard.RescanL1Disabled = true
				analysisBoard.RescanL1DisabledReason = "no_token"
			}
		}
	}

	gameActivity := ProjectGameActivitySummary{
		HasGame: false,
	}

	game, err := h.gameService.GetActiveOrLatestGame(ctx)
	if err == nil {
		gameActivity.HasGame = true
		gid := db.UUID(game.ID)
		now := time.Now().UTC()
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		weekStart := now.Add(-7 * 24 * time.Hour)

		agg, aggErr := h.queries.GetProjectGameActivityAggregates(ctx, db.GetProjectGameActivityAggregatesParams{
			ProjectID:  project.ID,
			GameID:     gid,
			WeekStart:  weekStart,
			MonthStart: monthStart,
		})
		if aggErr == nil {
			gameActivity.CommitsThisMonth = int(agg.CommitsThisMonth)
			gameActivity.CommitsThisWeek = int(agg.CommitsThisWeek)
			gameActivity.FileTouchCount = int(agg.FileTouchCount)
			gameActivity.UniqueDirs = int(agg.UniqueDirs)
		}

		board, bErr := h.queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
			ProjectID: project.ID,
			GameID:    game.ID,
		})
		if bErr == nil {
			gameActivity.Board = &analysis.BoardStats{
				CommitCount:      int(board.CommitCount),
				ContributorCount: int(board.ContributorCount),
			}
		} else if !errors.Is(bErr, pgx.ErrNoRows) {
			logger.Error().Err(bErr).Msg("GetBoardByProjectAndGame failed")
			http.Error(w, "failed to load game board", http.StatusInternalServerError)
			return
		}
	}

	// Get commits
	commits, err := h.queries.GetCommitsByProject(ctx, db.GetCommitsByProjectParams{
		ProjectID:   project.ID,
		OffsetCount: int32(offset),
		LimitCount:  int32(limit + 1),
	})
	if err != nil {
		logger.Error().Err(err).Msg("GetCommitsByProject failed")
		http.Error(w, "failed to load commits", http.StatusInternalServerError)
		return
	}

	hasMore := len(commits) > limit
	if hasMore {
		commits = commits[:limit]
	}

	commitEntries := make([]CommitEntry, len(commits))
	for i, c := range commits {

		flagReason := ""
		if c.FlagReason.Valid {
			flagReason = c.FlagReason.String
		}
		commitEntries[i] = CommitEntry{
			ID:         c.ID.String(),
			Hash:       c.Hash.String,
			Message:    c.Message,
			Author:     c.Author.String,
			URL:        c.Url,
			Timestamp:  c.Timestamp,
			Languages:  c.Languages,
			IsVerified: c.IsVerified,
			IsFlagged:  c.IsFlagged,
			FlagReason: flagReason,
		}
	}

	data := ProjectDetailData{
		Project:       entry,
		AnalysisBoard: analysisBoard,
		GameActivity:  gameActivity,
		Commits:       commitEntries,
		Offset:        offset,
		Limit:         limit,
		HasMore:       hasMore,
	}

	layout := layout.LayoutData{
		Title:       project.Name,
		CurrentPath: fmt.Sprintf("/projects/%s", slug),
		User:        layout.UserInfoFromContext(r),
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := ProjectCommitList(data).Render(ctx, w); err != nil {
			logger.Error().Err(err).Msg("ProjectCommitList render failed")
			http.Error(w, "render failed", http.StatusInternalServerError)
			return
		}
		return
	}
	if err := ProjectDetailPage(layout, data).Render(ctx, w); err != nil {
		logger.Error().Err(err).Msg("ProjectDetailPage render failed")
		http.Error(w, "render failed", http.StatusInternalServerError)
	}
}
