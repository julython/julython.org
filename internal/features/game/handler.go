package game

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"july/internal/auth"
	"july/internal/components/layout"
	"july/internal/components/piechart"
	"july/internal/db"
	"july/internal/services"
)

type gameHandler struct {
	queries     *db.Queries
	gameService *services.GameService
}

// Register mounts home/routes on the given mux.
func Register(mux *http.ServeMux, q *db.Queries, gs *services.GameService) {
	h := &gameHandler{queries: q, gameService: gs}
	mux.HandleFunc("GET /{$}", h.Home)
	mux.HandleFunc("GET /activity", h.Activity)
	mux.HandleFunc("GET /leaders", h.Leaders)
	mux.HandleFunc("GET /leaders/projects", h.Projects)
	mux.HandleFunc("GET /leaders/languages", h.Languages)
}

func (h *gameHandler) Home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(r.Context())

	logger.Info().Msg("Loading homepage")
	game, err := h.gameService.GetActiveOrLatestGame(ctx)
	if err != nil {
		h.renderEmptyHome(w, r)
		return
	}

	stats, err := h.queries.GetCommitStats(ctx, pgtype.UUID{Bytes: game.ID, Valid: true})
	if err != nil {
		stats = db.GetCommitStatsRow{}
	}

	dailyCommits, maxDay := h.getDailyCommits(ctx, game.ID, game.StartsAt)

	recentCommits := h.getRecentCommits(ctx, game.ID, 10)

	languageBreakdown := h.getLanguageBreakdown(ctx, game.ID)

	data := HomeData{
		Game: GameStats{
			Name:          game.Name,
			TotalCommits:  int(stats.TotalCommits),
			TotalUsers:    int(stats.UniqueUsers),
			TotalProjects: int(stats.UniqueProjects),
		},
		DailyCommits:      dailyCommits,
		MaxDayCommits:     maxDay,
		RecentCommits:     recentCommits,
		LanguageBreakdown: languageBreakdown,
	}

	layout := layout.LayoutData{
		Title:       "Home",
		CurrentPath: "/",
		User:        userInfoFromContext(r),
	}

	HomePage(layout, data).Render(ctx, w)
}

func (h *gameHandler) getLanguageBreakdown(ctx context.Context, gameID uuid.UUID) []piechart.DataPoint {
	rows, err := h.queries.GetLanguageLeaderboard(ctx, db.GetLanguageLeaderboardParams{
		GameID:     gameID,
		LimitCount: 15,
	})
	if err != nil {
		return []piechart.DataPoint{}
	}

	if len(rows) == 0 {
		return []piechart.DataPoint{}
	}

	points := make([]piechart.DataPoint, len(rows))
	for i, row := range rows {
		points[i] = piechart.DataPoint{
			Label: row.LanguageName,
			Value: int(row.CommitCount),
		}
	}

	return points
}

func (h *gameHandler) getDailyCommits(ctx context.Context, gameID uuid.UUID, _ time.Time) ([]DayCommits, int) {
	days := make([]DayCommits, 31)
	maxCount := 0

	for i := 0; i < 31; i++ {
		days[i] = DayCommits{
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

func (h *gameHandler) renderEmptyHome(w http.ResponseWriter, r *http.Request) {
	days := make([]DayCommits, 31)
	for i := 0; i < 31; i++ {
		days[i] = DayCommits{Day: i + 1, Count: 0}
	}

	data := HomeData{
		Game:              GameStats{Name: "Julython"},
		DailyCommits:      days,
		MaxDayCommits:     0,
		RecentCommits:     []RecentCommit{},
		LanguageBreakdown: nil,
	}

	layout := layout.LayoutData{
		Title:       "Home",
		CurrentPath: "/",
		User:        userInfoFromContext(r),
	}

	HomePage(layout, data).Render(r.Context(), w)
}

func userInfoFromContext(r *http.Request) *layout.UserInfo {
	u := auth.UserFromContext(r.Context())
	if u == nil {
		return nil
	}
	return &layout.UserInfo{
		Username:  u.Username,
		AvatarURL: u.AvatarURL,
	}
}
