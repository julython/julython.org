package api

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"july/internal/auth"
	"july/internal/config"
	"july/internal/db"
	"july/internal/features/assets"
	"july/internal/features/blog"
	"july/internal/features/game"
	"july/internal/features/help"
	"july/internal/features/players"
	"july/internal/features/profile"
	"july/internal/features/projects"
	"july/internal/features/proxy"
	"july/internal/i18n"
	"july/internal/metrics"
	"july/internal/services"
	"july/internal/webhooks"
	"july/web"
)

// buildMux constructs the ServeMux and all dependencies.
// Returns the mux and the two objects applyMiddleware needs.
func buildMux(pool *pgxpool.Pool, cfg *config.Config) (
	*http.ServeMux, *services.SessionManager, *auth.AuthHandler,
) {
	mux := http.NewServeMux()

	// Session manager with postgres store
	sessionMgr := services.NewSessionManager(pool, cfg.Session, cfg.IsProduction())
	sessionMgr.StartCleanup(cfg.Session.CleanupInterval)

	// Services
	queries := db.New(pool)
	userSvc := services.MustNewUserService(queries, cfg.Database.EncKey)
	gameSvc := services.NewGameService(queries)
	scanner := metrics.NewScanner(queries, pool, gameSvc, cfg.GitHubToken)

	// OAuth providers
	providers := make(map[string]services.OAuthProvider)
	if cfg.OAuth.GitHub.Enabled {
		providers["github"] = services.NewGitHubOAuth(
			cfg.OAuth.GitHub.ClientID,
			cfg.OAuth.GitHub.ClientSecret,
			cfg.OAuth.CallbackURL(),
		)
	}
	if cfg.OAuth.GitLab.Enabled {
		providers["gitlab"] = services.NewGitLabOAuth(
			cfg.OAuth.GitLab.ClientID,
			cfg.OAuth.GitLab.ClientSecret,
			cfg.OAuth.CallbackURL(),
		)
	}
	if cfg.OAuth.Password.Enabled {
		providers["password"] = services.NewPasswordOAuth(queries, cfg.OAuth.BaseURL)
	}

	enabled := make([]string, 0, len(providers))
	for name := range providers {
		enabled = append(enabled, name)
	}

	// Handlers
	authHandler := auth.NewAuthHandler(userSvc, gameSvc, sessionMgr.SessionManager, providers)

	// Auth Routes
	auth.Register(mux, authHandler)
	mux.HandleFunc("GET /set-language", i18n.SetLanguage)

	// Game Routes
	game.Register(mux, queries, gameSvc)

	// Players (per-user player boards)
	players.Register(mux, queries, gameSvc)

	// Project routes
	projects.Register(mux, queries, gameSvc, userSvc, scanner)

	// Help routes
	help.Register(mux)

	// Profiles
	profile.Register(mux, userSvc, sessionMgr.SessionManager, cfg.Webhooks.GitHub)

	// Assets (favicon, etc.)
	assets.Register(mux)

	// Blog
	blog.Register(mux)

	// Proxy
	proxy.Register(mux, userSvc, sessionMgr.SessionManager)

	// Webhooks
	webhooks.Register(mux, queries, pool, gameSvc, scanner)

	// Static files
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", web.AssetsHandler()))
	mux.Handle("GET /static/", http.StripPrefix("/static/", web.StaticHandler()))

	// Explicit 404 for anything unmatched so ErrorMiddleware can intercept it
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	})

	return mux, sessionMgr, authHandler
}

// applyMiddleware wraps a handler with the full middleware stack.
// Order (outermost = runs first): LoggingMiddleware → i18n → ErrorMiddleware → RecoveryMiddleware → LoadAndSave → UserMiddleware → h
func applyMiddleware(
	h http.Handler,
	sessionMgr *services.SessionManager,
	authHandler *auth.AuthHandler,
	logger zerolog.Logger,
	cfg *config.Config,
) http.Handler {
	h = authHandler.UserMiddleware(h)
	h = sessionMgr.LoadAndSave(h)
	h = RecoveryMiddleware(h)
	h = ErrorMiddleware(h)
	h = LoggingMiddleware(logger, cfg)(h)
	h = i18n.Middleware(h)
	return h
}

// NewRouter is the production entry point.
func NewRouter(pool *pgxpool.Pool, cfg *config.Config, logger zerolog.Logger) http.Handler {
	mux, sessionMgr, authHandler := buildMux(pool, cfg)
	return applyMiddleware(mux, sessionMgr, authHandler, logger, cfg)
}
