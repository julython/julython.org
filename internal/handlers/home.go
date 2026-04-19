package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"july/internal/components"
	"july/internal/db"
	"july/internal/services"
)

type HomeHandler struct {
	queries     *db.Queries
	gameService *services.GameService
}

func NewHomeHandler(queries *db.Queries, game *services.GameService) *HomeHandler {
	return &HomeHandler{queries: queries, gameService: game}
}

func (h *HomeHandler) Home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(r.Context())

	logger.Info().Msg("Loading homepage")
	// Get active or latest game
	game, err := h.gameService.GetActiveOrLatestGame(ctx)
	if err != nil {
		h.renderEmptyHome(w, r)
		return
	}

	// Get commit stats
	stats, err := h.queries.GetCommitStats(ctx, pgtype.UUID{Bytes: game.ID, Valid: true})
	if err != nil {
		stats = db.GetCommitStatsRow{}
	}

	// Get daily commit counts for histogram
	dailyCommits, maxDay := h.getDailyCommits(ctx, game.ID, game.StartsAt)

	// Get recent commits
	recentCommits := h.getRecentCommits(ctx, game.ID)

	gameStats := components.GameStats{
		Name:          game.Name,
		TotalCommits:  int(stats.TotalCommits),
		TotalUsers:    int(stats.UniqueUsers),
		TotalProjects: int(stats.UniqueProjects),
	}

	data := components.HomeData{
		Game:          gameStats,
		DailyCommits:  dailyCommits,
		MaxDayCommits: maxDay,
		RecentCommits: recentCommits,
	}

	layout := components.LayoutData{
		Title:       "Home",
		CurrentPath: "/",
		User:        getUserFromContext(r),
	}

	components.HomePage(layout, data).Render(ctx, w)
}

func (h *HomeHandler) getDailyCommits(ctx context.Context, gameID uuid.UUID, startAt time.Time) ([]components.DayCommits, int) {
	days := make([]components.DayCommits, 31)
	maxCount := 0

	for i := 0; i < 31; i++ {
		days[i] = components.DayCommits{
			Day:   i + 1,
			Count: 0,
		}
	}

	rows, err := h.queries.GetDailyCommitCounts(ctx, pgtype.UUID{Bytes: gameID, Valid: true})
	if err != nil {
		return days, maxCount
	}

	for _, row := range rows {
		dayNum := row.CommitDate.Time.Day()
		if dayNum >= 1 && dayNum <= 31 {
			days[dayNum-1].Count = int(row.CommitCount)
			if int(row.CommitCount) > maxCount {
				maxCount = int(row.CommitCount)
			}
		}
	}

	return days, maxCount
}

func (h *HomeHandler) getRecentCommits(ctx context.Context, gameID uuid.UUID) []components.RecentCommit {
	rows, err := h.queries.GetRecentCommits(ctx, db.GetRecentCommitsParams{
		GameID:     pgtype.UUID{Bytes: gameID, Valid: true},
		LimitCount: 10,
	})
	if err != nil {
		return []components.RecentCommit{}
	}

	commits := make([]components.RecentCommit, len(rows))
	for i, row := range rows {
		username := row.Author
		avatarURL := ""
		if row.Username.Valid {
			username = pgtype.Text{String: row.Username.String}
		}
		if row.AvatarUrl.Valid {
			avatarURL = row.AvatarUrl.String
		}

		commits[i] = components.RecentCommit{
			Username:  username.String,
			AvatarURL: avatarURL,
			Message:   row.Message,
			Project:   row.ProjectSlug,
			TimeAgo:   timeAgo(row.Timestamp),
		}
	}

	return commits
}

func timeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2")
	}
}

func (h *HomeHandler) renderEmptyHome(w http.ResponseWriter, r *http.Request) {
	days := make([]components.DayCommits, 31)
	for i := 0; i < 31; i++ {
		days[i] = components.DayCommits{Day: i + 1, Count: 0}
	}

	data := components.HomeData{
		Game:          components.GameStats{Name: "Julython"},
		DailyCommits:  days,
		MaxDayCommits: 0,
		RecentCommits: []components.RecentCommit{},
	}

	layout := components.LayoutData{
		Title:       "Home",
		CurrentPath: "/",
		User:        getUserFromContext(r),
	}

	components.HomePage(layout, data).Render(r.Context(), w)
}

func getUserFromContext(r *http.Request) *components.UserInfo {
	u := UserFromContext(r.Context())
	if u == nil {
		return nil
	}
	return &components.UserInfo{
		Username:  u.Username,
		AvatarURL: u.AvatarURL,
	}
}
