package services

import (
	"context"
	"net/http"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"july/internal/config"
)

type SessionManager struct {
	*scs.SessionManager
	pool        *pgxpool.Pool
	stopCleanup chan struct{}
}

func NewSessionManager(pool *pgxpool.Pool, cfg config.Session, isProd bool) *SessionManager {
	store := pgxstore.New(pool)

	sm := scs.New()
	sm.Store = store
	sm.Lifetime = cfg.Lifetime
	sm.Cookie.Name = cfg.CookieName
	sm.Cookie.HttpOnly = true
	sm.Cookie.Secure = isProd
	sm.Cookie.SameSite = http.SameSiteLaxMode

	return &SessionManager{
		SessionManager: sm,
		pool:           pool,
		stopCleanup:    make(chan struct{}),
	}
}

// StartCleanup runs periodic cleanup of expired sessions
func (sm *SessionManager) StartCleanup(interval time.Duration) {
	log.Info().Dur("interval", interval).Msg("starting session cleanup job")

	// Run once at startup
	go sm.cleanup()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				sm.cleanup()
			case <-sm.stopCleanup:
				log.Info().Msg("stopping session cleanup job")
				return
			}
		}
	}()
}

// StopCleanup stops the cleanup goroutine
func (sm *SessionManager) StopCleanup() {
	close(sm.stopCleanup)
}

func (sm *SessionManager) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := sm.pool.Exec(ctx, "DELETE FROM sessions WHERE expiry < now()")
	if err != nil {
		log.Error().Err(err).Msg("session cleanup failed")
		return
	}

	if n := result.RowsAffected(); n > 0 {
		log.Info().Int64("deleted", n).Msg("session cleanup completed")
	}
}
