package game

import (
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"july/internal/components/layout"
	"july/internal/db"
)

func (h *gameHandler) Leaders(w http.ResponseWriter, r *http.Request) {
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

	entries := make([]LeaderboardEntry, len(rows))
	for i, row := range rows {
		avatarURL := ""
		if row.AvatarUrl.Valid {
			avatarURL = row.AvatarUrl.String
		}
		entries[i] = LeaderboardEntry{
			Rank:         offset + i + 1,
			UserID:       row.UserID.String(),
			Username:     row.Username,
			Name:       row.Name,
			AvatarURL:    avatarURL,
			CommitCount:  int(row.CommitCount),
			ProjectCount: int(row.ProjectCount),
			Points:       int(row.Points),
		}
	}

	gameStats := GameStats{
		Name:         game.Name,
		TotalCommits: int(stats.TotalCommits),
		TotalUsers:   int(stats.UniqueUsers),
	}

	data := LeaderboardData{
		Game:       gameStats,
		Entries:    entries,
		Offset:     offset,
		Limit:      limit,
		HasMore:    hasMore,
		TotalUsers: int(stats.UniqueUsers),
	}

	layout := layout.LayoutData{
		Title:       "Leaderboard",
		CurrentPath: "/leaders",
		User:        userInfoFromContext(r),
	}

	if r.Header.Get("HX-Request") == "true" {
		LeaderboardTable(data).Render(ctx, w)
	} else {
		LeaderboardPage(layout, data).Render(ctx, w)
	}
}

func (h *gameHandler) Projects(w http.ResponseWriter, r *http.Request) {
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

	entries := make([]ProjectLeaderboardEntry, len(rows))
	for i, row := range rows {
		entries[i] = ProjectLeaderboardEntry{
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

	data := ProjectLeaderboardData{
		Game:    GameStats{Name: game.Name},
		Entries: entries,
		HasMore: hasMore,
	}

	layout := layout.LayoutData{
		Title:       "Project Leaderboard",
		CurrentPath: "/leaders/projects",
		User:        userInfoFromContext(r),
	}

	if r.Header.Get("HX-Request") == "true" {
		ProjectLeaderboardTable(data).Render(ctx, w)
	} else {
		ProjectLeaderboardPage(layout, data).Render(ctx, w)
	}
}

func (h *gameHandler) Languages(w http.ResponseWriter, r *http.Request) {
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

	entries := make([]LanguageLeaderboardEntry, len(rows))
	for i, row := range rows {
		entries[i] = LanguageLeaderboardEntry{
			Rank:         i + 1,
			LanguageID:   row.LanguageID.String(),
			LanguageName: row.LanguageName,
			Points:       int(row.Points),
			CommitCount:  int(row.CommitCount),
		}
	}

	data := LanguageLeaderboardData{
		Game:    GameStats{Name: game.Name},
		Entries: entries,
		HasMore: hasMore,
	}

	layout := layout.LayoutData{
		Title:       "Language Leaderboard",
		CurrentPath: "/leaders/languages",
		User:        userInfoFromContext(r),
	}

	if r.Header.Get("HX-Request") == "true" {
		LanguageLeaderboardTable(data).Render(ctx, w)
	} else {
		LanguageLeaderboardPage(layout, data).Render(ctx, w)
	}
}

func (h *gameHandler) renderEmptyLeaderboard(w http.ResponseWriter, r *http.Request) {
	data := LeaderboardData{
		Game:    GameStats{Name: "Julython"},
		Entries: []LeaderboardEntry{},
	}
	layout := layout.LayoutData{
		Title:       "Leaderboard",
		CurrentPath: "/leaders",
		User:        userInfoFromContext(r),
	}
	LeaderboardPage(layout, data).Render(r.Context(), w)
}

func (h *gameHandler) renderEmptyProjectLeaderboard(w http.ResponseWriter, r *http.Request) {
	data := ProjectLeaderboardData{
		Game:    GameStats{Name: "Julython"},
		Entries: []ProjectLeaderboardEntry{},
	}
	layout := layout.LayoutData{
		Title:       "Project Leaderboard",
		CurrentPath: "/leaders/projects",
		User:        userInfoFromContext(r),
	}
	ProjectLeaderboardPage(layout, data).Render(r.Context(), w)
}

func (h *gameHandler) renderEmptyLanguageLeaderboard(w http.ResponseWriter, r *http.Request) {
	data := LanguageLeaderboardData{
		Game:    GameStats{Name: "Julython"},
		Entries: []LanguageLeaderboardEntry{},
	}
	layout := layout.LayoutData{
		Title:       "Language Leaderboard",
		CurrentPath: "/leaders/languages",
		User:        userInfoFromContext(r),
	}
	LanguageLeaderboardPage(layout, data).Render(r.Context(), w)
}
