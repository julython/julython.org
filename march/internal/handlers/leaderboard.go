package handlers

import (
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"july/internal/components"
	"july/internal/db"
	"july/internal/services"
)

type LeaderboardHandler struct {
	queries     *db.Queries
	gameService *services.GameService
}

func NewLeaderboardHandler(q *db.Queries, gs *services.GameService) *LeaderboardHandler {
	return &LeaderboardHandler{
		queries:     q,
		gameService: gs,
	}
}

func (h *LeaderboardHandler) Leaders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := 25
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		offset, _ = strconv.Atoi(o)
	}

	game, err := h.gameService.GetActiveOrLatestGame(ctx)
	if err != nil {
		h.renderEmptyLeaderboard(w, r)
		return
	}

	stats, _ := h.queries.GetCommitStats(ctx, pgtype.UUID{Bytes: game.ID, Valid: true})

	rows, err := h.queries.GetLeaderboard(ctx, db.GetLeaderboardParams{
		GameID:      game.ID,
		LimitCount:  int32(limit + 1),
		OffsetCount: int32(offset),
	})
	if err != nil {
		http.Error(w, "failed to load leaderboard", http.StatusInternalServerError)
		return
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	entries := make([]components.LeaderboardEntry, len(rows))
	for i, row := range rows {
		avatarURL := ""
		if row.AvatarUrl.Valid {
			avatarURL = row.AvatarUrl.String
		}
		entries[i] = components.LeaderboardEntry{
			Rank:         offset + i + 1,
			UserID:       row.UserID.String(),
			Username:     row.Username,
			AvatarURL:    avatarURL,
			CommitCount:  int(row.CommitCount),
			ProjectCount: int(row.ProjectCount),
			Points:       int(row.Points),
		}
	}

	gameStats := components.GameStats{
		Name:         game.Name,
		TotalCommits: int(stats.TotalCommits),
		TotalUsers:   int(stats.UniqueUsers),
	}

	data := components.LeaderboardData{
		Game:       gameStats,
		Entries:    entries,
		Offset:     offset,
		Limit:      limit,
		HasMore:    hasMore,
		TotalUsers: int(stats.UniqueUsers),
	}

	layout := components.LayoutData{
		Title:       "Leaderboard",
		CurrentPath: "/leaders",
		User:        getUserFromContext(r),
	}

	if r.Header.Get("HX-Request") == "true" {
		components.LeaderboardTable(data).Render(ctx, w)
	} else {
		components.LeaderboardPage(layout, data).Render(ctx, w)
	}
}

func (h *LeaderboardHandler) Projects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := 25

	game, err := h.gameService.GetActiveOrLatestGame(ctx)
	if err != nil {
		h.renderEmptyProjectLeaderboard(w, r)
		return
	}

	rows, err := h.queries.GetProjectLeaderboard(ctx, db.GetProjectLeaderboardParams{
		GameID:     game.ID,
		LimitCount: int32(limit + 1),
	})
	if err != nil {
		http.Error(w, "failed to load project leaderboard", http.StatusInternalServerError)
		return
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	entries := make([]components.ProjectLeaderboardEntry, len(rows))
	for i, row := range rows {
		entries[i] = components.ProjectLeaderboardEntry{
			Rank:             i + 1,
			ProjectID:        row.ProjectID.String(),
			ProjectName:      row.ProjectName,
			ProjectURL:       row.ProjectUrl,
			Slug:             row.Slug,
			Points:           int(row.Points),
			PotentialPoints:  int(row.PotentialPoints),
			VerifiedPoints:   int(row.VerifiedPoints),
			CommitCount:      int(row.CommitCount),
			ContributorCount: int(row.ContributorCount),
		}
	}

	data := components.ProjectLeaderboardData{
		Game:    components.GameStats{Name: game.Name},
		Entries: entries,
		HasMore: hasMore,
	}

	layout := components.LayoutData{
		Title:       "Project Leaderboard",
		CurrentPath: "/leaders/projects",
		User:        getUserFromContext(r),
	}

	if r.Header.Get("HX-Request") == "true" {
		components.ProjectLeaderboardTable(data).Render(ctx, w)
	} else {
		components.ProjectLeaderboardPage(layout, data).Render(ctx, w)
	}
}

func (h *LeaderboardHandler) Languages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := 25

	game, err := h.gameService.GetActiveOrLatestGame(ctx)
	if err != nil {
		h.renderEmptyLanguageLeaderboard(w, r)
		return
	}

	rows, err := h.queries.GetLanguageLeaderboard(ctx, db.GetLanguageLeaderboardParams{
		GameID:     game.ID,
		LimitCount: int32(limit + 1),
	})
	if err != nil {
		http.Error(w, "failed to load language leaderboard", http.StatusInternalServerError)
		return
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	entries := make([]components.LanguageLeaderboardEntry, len(rows))
	for i, row := range rows {
		entries[i] = components.LanguageLeaderboardEntry{
			Rank:         i + 1,
			LanguageID:   row.LanguageID.String(),
			LanguageName: row.LanguageName,
			Points:       int(row.Points),
			CommitCount:  int(row.CommitCount),
		}
	}

	data := components.LanguageLeaderboardData{
		Game:    components.GameStats{Name: game.Name},
		Entries: entries,
		HasMore: hasMore,
	}

	layout := components.LayoutData{
		Title:       "Language Leaderboard",
		CurrentPath: "/leaders/languages",
		User:        getUserFromContext(r),
	}

	if r.Header.Get("HX-Request") == "true" {
		components.LanguageLeaderboardTable(data).Render(ctx, w)
	} else {
		components.LanguageLeaderboardPage(layout, data).Render(ctx, w)
	}
}

func (h *LeaderboardHandler) renderEmptyLeaderboard(w http.ResponseWriter, r *http.Request) {
	data := components.LeaderboardData{
		Game:    components.GameStats{Name: "Julython"},
		Entries: []components.LeaderboardEntry{},
	}
	layout := components.LayoutData{
		Title:       "Leaderboard",
		CurrentPath: "/leaders",
		User:        getUserFromContext(r),
	}
	components.LeaderboardPage(layout, data).Render(r.Context(), w)
}

func (h *LeaderboardHandler) renderEmptyProjectLeaderboard(w http.ResponseWriter, r *http.Request) {
	data := components.ProjectLeaderboardData{
		Game:    components.GameStats{Name: "Julython"},
		Entries: []components.ProjectLeaderboardEntry{},
	}
	layout := components.LayoutData{
		Title:       "Project Leaderboard",
		CurrentPath: "/leaders/projects",
		User:        getUserFromContext(r),
	}
	components.ProjectLeaderboardPage(layout, data).Render(r.Context(), w)
}

func (h *LeaderboardHandler) renderEmptyLanguageLeaderboard(w http.ResponseWriter, r *http.Request) {
	data := components.LanguageLeaderboardData{
		Game:    components.GameStats{Name: "Julython"},
		Entries: []components.LanguageLeaderboardEntry{},
	}
	layout := components.LayoutData{
		Title:       "Language Leaderboard",
		CurrentPath: "/leaders/languages",
		User:        getUserFromContext(r),
	}
	components.LanguageLeaderboardPage(layout, data).Render(r.Context(), w)
}
