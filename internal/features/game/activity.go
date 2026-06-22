package game

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"july/internal/components/layout"
	"july/internal/db"
	"july/internal/services"
	"july/internal/shared"
)

func (h *gameHandler) Activity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(r.Context())

	logger.Info().Msg("Loading activity page")

	// Get active or latest game
	game, err := h.gameService.GetActiveOrLatestGame(ctx)
	if err != nil {
		if errors.Is(err, services.ErrNoActiveGame) {
			// No game yet - render empty activity page
			layout := layout.LayoutData{
				Title:       "Recent Activity",
				CurrentPath: "/activity",
				User:        userInfoFromContext(r),
			}

			ActivityPage(layout, ActivityData{
				Commits: []RecentCommit{},
			}).Render(ctx, w)
			return
		}
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	// Get recent commits for the activity page
	recentCommits := h.getRecentCommits(ctx, game.ID, 10)

	layout := layout.LayoutData{
		Title:       "Recent Activity",
		CurrentPath: "/activity",
		User:        userInfoFromContext(r),
	}

	ActivityPage(layout, ActivityData{
		Commits: recentCommits,
	}).Render(ctx, w)
}

func (h *gameHandler) getRecentCommits(ctx context.Context, gameID uuid.UUID, limit int) []RecentCommit {
	rows, err := h.queries.GetRecentCommits(ctx, db.GetRecentCommitsParams{
		GameID:     pgtype.UUID{Bytes: gameID, Valid: true},
		LimitCount: limit,
	})
	if err != nil {
		return []RecentCommit{}
	}

	commits := make([]RecentCommit, len(rows))
	for i, row := range rows {
		username := row.Author.String
		avatarURL := ""
		if row.Username.Valid {
			username = row.Username.String
		}
		if row.AvatarUrl.Valid {
			avatarURL = row.AvatarUrl.String
		}

		commits[i] = RecentCommit{
			Username:    username,
			Author:      row.Author.String,
			AvatarURL:   avatarURL,
			Message:     row.Message,
			Project:     row.ProjectSlug,
			ProjectName: row.ProjectName,
			TimeAgo:     shared.TimeAgo(row.Timestamp),
		}
	}

	return commits
}
