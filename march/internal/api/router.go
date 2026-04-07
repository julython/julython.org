package api

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"july/internal/config"
	"july/internal/db"
	"july/internal/handlers"
	"july/internal/i18n"
	"july/internal/services"
	"july/internal/webhooks"
	"july/web"
)

// buildMux constructs the ServeMux and all dependencies.
// Returns the mux and the two objects applyMiddleware needs.
func buildMux(pool *pgxpool.Pool, cfg *config.Config, logger zerolog.Logger) (
	*http.ServeMux, *services.SessionManager, *handlers.AuthHandler,
) {
	mux := http.NewServeMux()

	// Session manager with postgres store
	sessionMgr := services.NewSessionManager(pool, cfg.Session, cfg.IsProduction())
	sessionMgr.StartCleanup(cfg.Session.CleanupInterval)

	// Services
	queries := db.New(pool)
	userSvc := services.MustNewUserService(queries, cfg.Database.EncKey)
	gameSvc := services.NewGameService(queries)

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
	log.Info().
		Strs("providers", enabled).
		Str("callback", cfg.OAuth.CallbackURL()).
		Msg("oauth configured")

	// Handlers
	authHandler := handlers.NewAuthHandler(userSvc, gameSvc, sessionMgr.SessionManager, providers)
	homeHandler := handlers.NewHomeHandler(queries, gameSvc)
	leaderboardHandler := handlers.NewLeaderboardHandler(queries, gameSvc)
	l1Scanner := services.NewL1Scanner(queries, pool, cfg.GitHubToken)
	webhookHandler := webhooks.NewHandler(queries, pool, gameSvc, l1Scanner)
	projectHandler := handlers.NewProjectHandler(queries, gameSvc, userSvc, l1Scanner)
	profileHandler := handlers.NewProfileHandler(userSvc, sessionMgr.SessionManager, cfg.Webhooks.GitHub)
	blogHandler := handlers.NewBlogHandler()
	helpHandler := handlers.NewHelpHandler()
	proxyHandler := handlers.NewGitHubProxyHandler(userSvc, sessionMgr.SessionManager)

	// Routes
	mux.HandleFunc("GET /favicon.svg", handlers.FaviconHandler)
	mux.HandleFunc("GET /auth/login/{provider}", authHandler.Login)
	mux.HandleFunc("GET /auth/callback", authHandler.Callback)
	mux.HandleFunc("GET /auth/session", authHandler.Session)
	mux.HandleFunc("GET /auth/logout", authHandler.Logout)

	mux.HandleFunc("GET /{$}", homeHandler.Home)
	mux.HandleFunc("GET /leaders", leaderboardHandler.Leaders)
	mux.HandleFunc("GET /leaders/projects", leaderboardHandler.Projects)
	mux.HandleFunc("GET /leaders/languages", leaderboardHandler.Languages)
	mux.HandleFunc("GET /projects", projectHandler.List)
	mux.HandleFunc("POST /projects/{slug}/analysis/l1", projectHandler.PostProjectRescanL1)
	mux.HandleFunc("GET /projects/{slug}", projectHandler.Detail)
	mux.HandleFunc("GET /set-language", i18n.SetLanguage)

	// Projects
	mux.HandleFunc("POST /api/projects/{projectID}/analysis", projectHandler.PostProjectAnalysis)

	// Profiles
	mux.HandleFunc("GET /profile", profileHandler.Overview)
	mux.HandleFunc("GET /profile/webhooks", profileHandler.Webhooks)
	mux.HandleFunc("GET /profile/webhooks/repos", profileHandler.WebhookRepos)
	mux.HandleFunc("POST /profile/webhooks/{repoID}/hooks", profileHandler.AddWebhook)
	mux.HandleFunc("DELETE /profile/webhooks/{repoID}/hooks/{hookID}", profileHandler.DeleteWebhook)
	mux.HandleFunc("GET /profile/settings", profileHandler.Settings)
	mux.HandleFunc("POST /profile/settings", profileHandler.UpdateSettings)

	// Help
	mux.HandleFunc("GET /help", helpHandler.Help)
	mux.HandleFunc("GET /about", helpHandler.About)
	mux.HandleFunc("GET /privacy", helpHandler.Privacy)

	// Blog
	mux.HandleFunc("GET /blog", blogHandler.List)
	mux.HandleFunc("GET /blog/{slug}", blogHandler.Detail)

	// Webhooks
	mux.HandleFunc("GET /api/v1/gh/{path...}", proxyHandler.Proxy)
	mux.HandleFunc("POST /api/v1/github", webhookHandler.HandleGitHubWebhook)

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
	authHandler *handlers.AuthHandler,
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
	mux, sessionMgr, authHandler := buildMux(pool, cfg, logger)
	return applyMiddleware(mux, sessionMgr, authHandler, logger, cfg)
}

// LoggingMiddleware injects a request-scoped zerolog logger (with request ID)
// into the context and logs each request on completion.
func LoggingMiddleware(logger zerolog.Logger, cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get("X-Request-Id")
			if reqID == "" {
				reqID = generateRequestID()
			}
			w.Header().Set("X-Request-Id", reqID)

			ctx := logger.With().
				Str("request_id", reqID).
				Str("method", r.Method).
				Str("path", r.URL.Path)

			if cfg.IsProduction() {
				traceID, spanID := parseTraceHeader(r.Header.Get("X-Cloud-Trace-Context"))
				trace := fmt.Sprintf("projects/%s/traces/%s", cfg.ProjectID, traceID)
				ctx = ctx.
					Str("logging.googleapis.com/trace", trace).
					Str("logging.googleapis.com/spanId", spanID).
					Dict("logging.googleapis.com/labels", zerolog.Dict().
						Str("request_id", reqID),
					)
			}

			reqLogger := ctx.Logger()
			r = r.WithContext(reqLogger.WithContext(r.Context()))

			start := time.Now()
			wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			reqLogger.Info().
				Int("status", wrapped.status).
				Dur("duration", time.Since(start)).
				Msgf("%s %s", r.Method, r.URL.RequestURI())
		})
	}
}

// parseTraceHeader parses "X-Cloud-Trace-Context: TRACE_ID/SPAN_ID;o=1"
func parseTraceHeader(h string) (traceID, spanID string) {
	if h == "" {
		return "", ""
	}
	parts := strings.SplitN(h, ";", 2)
	ids := strings.SplitN(parts[0], "/", 2)
	traceID = ids[0]
	if len(ids) == 2 {
		spanID = ids[1]
	}
	return traceID, spanID
}

// RecoveryMiddleware catches panics, logs them with the request-scoped logger,
// and prints the stack trace to stderr in dev-friendly format.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Ctx(r.Context()).Error().
					Interface("error", err).
					Msg("panic recovered")
				fmt.Fprintf(os.Stderr, "\n%s\n", debug.Stack())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return fmt.Sprintf("%x", b)
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
