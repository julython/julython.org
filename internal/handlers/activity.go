package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"july/internal/components"
	"july/internal/db"
	"july/internal/services"
)

type ActivityHandler struct {
	queries     *db.Queries
	gameService *services.GameService
}

func NewActivityHandler(queries *db.Queries, game *services.GameService) *ActivityHandler {
	return &ActivityHandler{queries: queries, gameService: game}
}

func (h *ActivityHandler) Activity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(r.Context())

	logger.Info().Msg("Loading activity page")

	// Get active or latest game
	game, err := h.gameService.GetActiveOrLatestGame(ctx)
	if err != nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	// Get recent commits for the activity page
	recentCommits := h.getRecentCommits(ctx, game.ID, 50)

	layout := components.LayoutData{
		Title:       "Recent Activity",
		CurrentPath: "/activity",
		User:        getUserFromContext(r),
	}

	components.ActivityPage(layout, components.ActivityData{
		Commits: recentCommits,
	}).Render(ctx, w)
}

func (h *ActivityHandler) getRecentCommits(ctx context.Context, gameID uuid.UUID, limit int) []components.RecentCommit {
	rows, err := h.queries.GetRecentCommits(ctx, db.GetRecentCommitsParams{
		GameID:     pgtype.UUID{Bytes: gameID, Valid: true},
		LimitCount: limit,
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
