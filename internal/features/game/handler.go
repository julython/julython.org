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
	"july/internal/db"
	"july/internal/services"
)

// Handler handles home page HTTP requests.
type Handler struct {
	queries     *db.Queries
	gameService *services.GameService
}

// NewHandler creates a new home handler.
func NewHandler(q *db.Queries, gs *services.GameService) *Handler {
	return &Handler{queries: q, gameService: gs}
}

// Register mounts home routes on the given mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", h.Home)
	mux.HandleFunc("GET /activity", h.Activity)
	mux.HandleFunc("GET /leaders", h.Leaders)
	mux.HandleFunc("GET /leaders/projects", h.Projects)
	mux.HandleFunc("GET /leaders/languages", h.Languages)
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
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

	recentCommits := h.getRecentCommits(ctx, game.ID, maxDay)

	data := HomeData{
		Game: GameStats{
			Name:          game.Name,
			TotalCommits:  int(stats.TotalCommits),
			TotalUsers:    int(stats.UniqueUsers),
			TotalProjects: int(stats.UniqueProjects),
		},
		DailyCommits:  dailyCommits,
		MaxDayCommits: maxDay,
		RecentCommits: recentCommits,
	}

	layout := layout.LayoutData{
		Title:       "Home",
		CurrentPath: "/",
		User:        userInfoFromContext(r),
	}

	HomePage(layout, data).Render(ctx, w)
}

func (h *Handler) getDailyCommits(ctx context.Context, gameID uuid.UUID, _ time.Time) ([]DayCommits, int) {
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

func (h *Handler) renderEmptyHome(w http.ResponseWriter, r *http.Request) {
	days := make([]DayCommits, 31)
	for i := 0; i < 31; i++ {
		days[i] = DayCommits{Day: i + 1, Count: 0}
	}

	data := HomeData{
		Game:          GameStats{Name: "Julython"},
		DailyCommits:  days,
		MaxDayCommits: 0,
		RecentCommits: []RecentCommit{},
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
