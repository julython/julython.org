package handlers

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/rs/zerolog/log"

	"july/internal/services"
)

const githubAPIBase = "https://api.github.com/"

// allowedPathPrefixes restricts the proxy to read-only endpoints
// needed by Majordomo for repo analysis. All are GET-only.
var allowedPathPrefixes = []string{
	"repos/",     // /repos/{owner}/{repo}/...
	"user/repos", // /user/repos (list authed user's repos)
}

// forwardedRateLimitHeaders are passed back to the client so it
// can observe quota without us having to parse them.
var forwardedRateLimitHeaders = []string{
	"X-RateLimit-Limit",
	"X-RateLimit-Remaining",
	"X-RateLimit-Reset",
	"X-RateLimit-Used",
	"X-RateLimit-Resource",
}

type GitHubProxyHandler struct {
	userSvc *services.UserService
	sm      *scs.SessionManager
	client  *http.Client
}

func NewGitHubProxyHandler(userSvc *services.UserService, sm *scs.SessionManager) *GitHubProxyHandler {
	return &GitHubProxyHandler{
		userSvc: userSvc,
		sm:      sm,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Proxy forwards GET requests to the GitHub API, injecting the logged-in
// user's OAuth token so we don't hit the unauthenticated rate limit (60 req/hr)
// at the server level. The token scope only covers public repos, which is fine
// for Majordomo's use case.
//
// Route: GET /api/v1/gh/{path...}
// Example: GET /api/v1/gh/repos/python/cpython/git/trees/main?recursive=1
func (h *GitHubProxyHandler) Proxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(ctx)

	user := UserFromContext(ctx)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	githubPath := r.PathValue("path")
	if !isAllowedPath(githubPath) {
		http.Error(w, "path not allowed", http.StatusForbidden)
		return
	}

	// Preserve query params (e.g. ?recursive=1, ?per_page=100)
	upstream := githubAPIBase + githubPath
	if r.URL.RawQuery != "" {
		upstream += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstream, nil)
	if err != nil {
		http.Error(w, "failed to build upstream request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	// Inject the user's token. If they somehow logged in without GitHub
	// (e.g. GitLab-only), we fall back to unauthenticated — still works
	// for public repos, just at the lower rate limit.
	token, err := h.userSvc.GetOAuthToken(ctx, user.ID, services.IdentifierGitHub)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("user_id", user.ID.String()).
			Msg("no github token for proxy; falling back to unauthenticated")
	} else {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		logger.Error().Err(err).Str("upstream", upstream).Msg("github proxy upstream failed")
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Forward content type and rate limit headers so the client
	// can observe quota and handle 403/429 itself.
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	for _, name := range forwardedRateLimitHeaders {
		if v := resp.Header.Get(name); v != "" {
			w.Header().Set(name, v)
		}
	}

	// Let browsers and CDNs cache public responses briefly.
	// The client can always bust with a cache-busting query param.
	if resp.StatusCode == http.StatusOK {
		w.Header().Set("Cache-Control", "public, max-age=60")
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		logger.Error().Err(err).Msg("github proxy body copy failed")
	}
}

func isAllowedPath(path string) bool {
	for _, prefix := range allowedPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
