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

func NewRouter(pool *pgxpool.Pool, cfg *config.Config, logger zerolog.Logger) http.Handler {
	mux := http.NewServeMux()

	// Session manager with postgres store
	sessionMgr := services.NewSessionManager(pool, cfg.Session, cfg.IsProduction())
	sessionMgr.StartCleanup(cfg.Session.CleanupInterval)

	// Services
	queries := db.New(pool)
	userSvc := services.NewUserService(queries)
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
	webhookHandler := webhooks.NewHandler(queries, gameSvc)
	projectHandler := handlers.NewProjectHandler(queries, gameSvc)
	profileHandler := handlers.NewProfileHandler(userSvc, sessionMgr.SessionManager, cfg.Webhooks.GitHub)
	blogHandler := handlers.NewBlogHandler()
	helpHandler := handlers.NewHelpHandler()

	// Routes
	mux.HandleFunc("GET /favicon.svg", handlers.FaviconHandler)
	mux.HandleFunc("GET /auth/login/{provider}", authHandler.Login)
	mux.HandleFunc("GET /auth/callback", authHandler.Callback)
	mux.HandleFunc("GET /auth/session", authHandler.Session)
	mux.HandleFunc("GET /auth/logout", authHandler.Logout)

	mux.HandleFunc("GET /{$}", homeHandler.Home) // anchored to root only
	mux.HandleFunc("GET /leaders", leaderboardHandler.Leaders)
	mux.HandleFunc("GET /leaders/projects", leaderboardHandler.Projects)
	mux.HandleFunc("GET /leaders/languages", leaderboardHandler.Languages)
	mux.HandleFunc("GET /projects", projectHandler.List)
	mux.HandleFunc("GET /projects/{slug}", projectHandler.Detail)
	mux.HandleFunc("GET /set-language", i18n.SetLanguage)

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
	mux.HandleFunc("POST /api/v1/github", webhookHandler.HandleGitHubWebhook)

	// Static files
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", web.AssetsHandler()))
	mux.Handle("GET /static/", http.StripPrefix("/static/", web.StaticHandler()))

	// Explicit 404 for anything unmatched so ErrorMiddleware can intercept it
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	})

	// Apply middleware stack (outermost = runs first on each request):
	// LoggingMiddleware → RecoveryMiddleware → ErrorMiddleware → i18n → LoadAndSave → UserMiddleware → mux
	var handler http.Handler = mux
	handler = authHandler.UserMiddleware(handler)
	handler = sessionMgr.LoadAndSave(handler)
	handler = RecoveryMiddleware(handler)             // writes 500 into ErrorMiddleware's buffer
	handler = ErrorMiddleware(handler)                // intercepts 4xx/5xx and renders pretty page
	handler = LoggingMiddleware(logger, cfg)(handler) // injects logger + records access log
	handler = i18n.Middleware(handler)

	return handler
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
	// Split off the options ";o=1"
	parts := strings.SplitN(h, ";", 2)
	// Split trace/span
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
