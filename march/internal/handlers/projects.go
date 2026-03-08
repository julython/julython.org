package handlers

import (
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"july/internal/components"
	"july/internal/db"
	"july/internal/services"
)

type ProjectHandler struct {
	queries     *db.Queries
	gameService *services.GameService
}

func NewProjectHandler(q *db.Queries, gs *services.GameService) *ProjectHandler {
	return &ProjectHandler{queries: q, gameService: gs}
}

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

	// Get board stats for the active game (best-effort)
	var boardStats *components.ProjectBoardStats
	game, err := h.gameService.GetActiveOrLatestGame(ctx)
	if err == nil {
		board, err := h.queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
			ProjectID: project.ID,
			GameID:    game.ID,
		})
		if err == nil {
			boardStats = &components.ProjectBoardStats{
				Points:           int(board.Points),
				PotentialPoints:  int(board.PotentialPoints),
				VerifiedPoints:   int(board.VerifiedPoints),
				CommitCount:      int(board.CommitCount),
				ContributorCount: int(board.ContributorCount),
			}
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
		Project:    entry,
		BoardStats: boardStats,
		Commits:    commitEntries,
		Offset:     offset,
		Limit:      limit,
		HasMore:    hasMore,
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
