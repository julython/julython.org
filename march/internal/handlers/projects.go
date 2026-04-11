package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"july/internal/components"
	"july/internal/db"
	"july/internal/services"
)

type ProjectHandler struct {
	queries     *db.Queries
	gameService *services.GameService
	userService *services.UserService
	l1Scanner   *services.L1Scanner
}

func NewProjectHandler(q *db.Queries, gs *services.GameService, us *services.UserService, l1 *services.L1Scanner) *ProjectHandler {
	return &ProjectHandler{queries: q, gameService: gs, userService: us, l1Scanner: l1}
}

// Order matches metrics.Parse and the project detail board UI.
var analysisBoardSpec = []struct {
	key     string
	i18nKey string
}{
	{"readme", "projects.MetricReadme"},
	{"tests", "projects.MetricTests"},
	{"ci", "projects.MetricCI"},
	{"structure", "projects.MetricStructure"},
	{"linting", "projects.MetricLinting"},
	{"deps", "projects.MetricDeps"},
	{"docs", "projects.MetricDocs"},
	{"ai_ready", "projects.MetricAIReady"},
}

const analysisBoardMaxPts = 480

const projectPageSize = 25

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	search := q.Get("search")
	service := q.Get("service")
	cursorStr := q.Get("cursor")

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
		http.Error(w, "failed to load projects", http.StatusInternalServerError)
		return
	}

	hasMore := len(projects) > projectPageSize
	if hasMore {
		projects = projects[:projectPageSize]
	}

	entries := make([]components.ProjectEntry, len(projects))
	for i, p := range projects {
		desc := ""
		if p.Description.Valid {
			desc = p.Description.String
		}
		entries[i] = components.ProjectEntry{
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

	data := components.ProjectListData{
		Entries:    entries,
		Search:     search,
		Service:    service,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}

	if r.Header.Get("HX-Request") == "true" {
		components.ProjectListItems(data).Render(ctx, w)
		return
	}

	layout := components.LayoutData{
		Title:       "Projects",
		CurrentPath: "/projects",
		User:        getUserFromContext(r),
	}
	components.ProjectListPage(layout, data).Render(ctx, w)
}

func (h *ProjectHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	entry := components.ProjectEntry{
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
		http.Error(w, "failed to load analysis metrics", http.StatusInternalServerError)
		return
	}
	levelByType := make(map[string]int16, len(analysisRows))
	scoreByType := make(map[string]int16, len(analysisRows))
	for _, row := range analysisRows {
		levelByType[row.MetricType] = row.Level
		scoreByType[row.MetricType] = row.Score
	}

	showMetricAI := false
	if sess := UserFromContext(ctx); sess != nil {
		if u, err := h.userService.FindByID(ctx, sess.ID); err == nil && canEditProject(&u, project) && project.Service == "github" {
			if !project.IsPrivate && h.l1Scanner.IsConfigured() {
				showMetricAI = true
			}
		}
	}

	tiles := make([]components.ProjectAnalysisTile, 0, len(analysisBoardSpec))
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
		tiles = append(tiles, components.ProjectAnalysisTile{
			MetricKey:    spec.key,
			Level:        level,
			Score:        score,
			I18nKey:      spec.i18nKey,
			ShowMetricAI: showMetricAI,
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
	analysisBoard := components.ProjectAnalysisBoard{
		Tiles:            tiles,
		EarnedPts:        earned,
		MaxPts:           analysisBoardMaxPts,
		AnalysisRunCount: len(shaDistinct),
	}
	if haveMetricAt {
		analysisBoard.LastAnalyzedAgo = timeAgo(lastMetricAt)
	}
	if sess := UserFromContext(ctx); sess != nil {
		if u, err := h.userService.FindByID(ctx, sess.ID); err == nil && canEditProject(&u, project) && project.Service == "github" {
			analysisBoard.RescanL1Slug = project.Slug
			switch {
			case project.IsPrivate:
				analysisBoard.RescanL1Disabled = true
				analysisBoard.RescanL1DisabledReason = "private"
			case !h.l1Scanner.IsConfigured():
				analysisBoard.RescanL1Disabled = true
				analysisBoard.RescanL1DisabledReason = "no_token"
			}
		}
	}

	gameActivity := components.ProjectGameActivitySummary{
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
			gameActivity.Board = &components.ProjectBoardStats{
				CommitCount:      int(board.CommitCount),
				ContributorCount: int(board.ContributorCount),
			}
		} else if !errors.Is(bErr, pgx.ErrNoRows) {
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
		http.Error(w, "failed to load commits", http.StatusInternalServerError)
		return
	}

	hasMore := len(commits) > limit
	if hasMore {
		commits = commits[:limit]
	}

	commitEntries := make([]components.CommitEntry, len(commits))
	for i, c := range commits {

		flagReason := ""
		if c.FlagReason.Valid {
			flagReason = c.FlagReason.String
		}
		commitEntries[i] = components.CommitEntry{
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

	data := components.ProjectDetailData{
		Project:       entry,
		AnalysisBoard: analysisBoard,
		GameActivity:  gameActivity,
		Commits:       commitEntries,
		Offset:        offset,
		Limit:         limit,
		HasMore:       hasMore,
	}

	layout := components.LayoutData{
		Title:       project.Name,
		CurrentPath: fmt.Sprintf("/projects/%s", slug),
		User:        getUserFromContext(r),
	}

	if r.Header.Get("HX-Request") == "true" {
		components.ProjectCommitList(data).Render(ctx, w)
		return
	}
	components.ProjectDetailPage(layout, data).Render(ctx, w)
}
