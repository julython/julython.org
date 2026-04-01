package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"july/internal/config"
	"july/internal/db"
	"july/internal/handlers"
)

// NewTestRouter wraps the real router and adds a /test/login endpoint that
// sets a session cookie without going through OAuth.
// Never use this outside of testutil.
func NewTestRouter(pool *pgxpool.Pool, cfg *config.Config, logger zerolog.Logger) http.Handler {
	mux, sessionMgr, authHandler := buildMux(pool, cfg, logger)

	queries := db.New(pool)

	mux.HandleFunc("GET /test/login", func(w http.ResponseWriter, r *http.Request) {
		rawID := r.URL.Query().Get("userID")
		if rawID == "" {
			http.Error(w, "userID required", http.StatusBadRequest)
			return
		}
		userID, err := uuid.Parse(rawID)
		if err != nil {
			http.Error(w, "invalid userID", http.StatusBadRequest)
			return
		}
		user, err := queries.GetUserByID(r.Context(), userID)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		// Store the same SessionUser shape the real auth callback stores so
		// that UserMiddleware's type assertion succeeds.
		sessionMgr.SessionManager.Put(r.Context(), handlers.SessionKeyUser, handlers.SessionUser{
			ID:        user.ID,
			Username:  user.Username,
			Name:      user.Name,
			AvatarURL: user.AvatarUrl.String,
		})
	})

	return applyMiddleware(mux, sessionMgr, authHandler, logger, cfg)
}
