package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"july/internal/services"
)

func init() {
	gob.Register(SessionUser{})
}

const (
	SessionKeyUser          = "user"
	sessionKeyIdentityKey   = "identity_key"
	sessionKeyOAuthState    = "oauth_state"
	sessionKeyOAuthProvider = "oauth_provider"
	sessionKeyPKCEVerifier  = "pkce_verifier"
)

// SessionUser is stored in the session after login
type SessionUser struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url,omitempty"`
}

type AuthHandler struct {
	users     *services.UserService
	game      *services.GameService
	session   *scs.SessionManager
	providers map[string]services.OAuthProvider
}

func NewAuthHandler(
	users *services.UserService,
	game *services.GameService,
	session *scs.SessionManager,
	providers map[string]services.OAuthProvider,
) *AuthHandler {
	return &AuthHandler{
		users:     users,
		game:      game,
		session:   session,
		providers: providers,
	}
}

// Login initiates the OAuth flow - GET /auth/login/{provider}
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(ctx)

	providerName := r.PathValue("provider")
	provider, ok := h.providers[providerName]
	if !ok {
		logger.Error().Msgf("Unknown provider: %s", providerName)
		http.Error(w, "Unknown provider", http.StatusBadRequest)
		return
	}
	logger.Info().Str("provider", providerName).Msg("Handling auth login")

	state, _ := generateRandomString(16)
	verifier, challenge, _ := services.GeneratePKCE()

	h.session.Put(r.Context(), sessionKeyOAuthState, state)
	h.session.Put(r.Context(), sessionKeyOAuthProvider, providerName)
	h.session.Put(r.Context(), sessionKeyPKCEVerifier, verifier)

	authURL := provider.AuthorizationURL(state, challenge)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// Callback handles the OAuth callback - GET /auth/callback
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(ctx)

	// Validate state
	state := r.URL.Query().Get("state")
	expectedState := h.session.PopString(ctx, sessionKeyOAuthState)
	if state == "" || state != expectedState {
		logger.Error().Msg("Invalid state")
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	// Get provider
	providerName := h.session.GetString(ctx, sessionKeyOAuthProvider)
	provider, ok := h.providers[providerName]
	if !ok {
		logger.Error().Msg("Invalid provider")
		http.Error(w, "Invalid provider", http.StatusBadRequest)
		return
	}

	// Check for OAuth error
	if errCode := r.URL.Query().Get("error"); errCode != "" {
		errDesc := r.URL.Query().Get("error_description")
		logger.Warn().Str("error", errDesc).Msg("OAuth Error")
		http.Error(w, errDesc, http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		logger.Warn().Msg("Missing code")
		http.Error(w, "Missing code", http.StatusBadRequest)
		return
	}

	// Exchange code for tokens
	verifier := h.session.PopString(ctx, sessionKeyPKCEVerifier)
	tokens, err := provider.ExchangeCode(ctx, code, verifier)
	if err != nil {
		logger.Error().Err(err).Msg("failed to exchange authorization code for tokens")
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}

	// Get user info from provider
	oauthUser, err := provider.GetUser(ctx, tokens)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get user info from provider")
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Login or register
	user, created, err := h.users.OAuthLoginOrRegister(ctx, oauthUser)
	if err != nil {
		logger.Error().Err(err).Str("username", oauthUser.Username).Msg("failed to login or register oauth user")
		http.Error(w, "Login failed", http.StatusInternalServerError)
		return
	}

	// If new user or new identity, claim orphan commits in background
	if created {
		go h.claimOrphanCommits(user.ID, oauthUser)
	}

	// Store session
	h.session.Put(ctx, SessionKeyUser, SessionUser{
		ID:        user.ID,
		Username:  user.Username,
		Name:      user.Name,
		AvatarURL: stringFromNull(user.AvatarUrl),
	})
	h.session.Put(ctx, sessionKeyIdentityKey, oauthUser.Key())

	// Redirect to original destination or home
	redirectTo := h.session.PopString(ctx, "redirect_after_login")
	if redirectTo == "" {
		redirectTo = "/"
	}
	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}

// Session returns the current user session - GET /auth/session
func (h *AuthHandler) Session(w http.ResponseWriter, r *http.Request) {
	user := h.GetCurrentUser(r)
	if user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	respondJSON(w, r, http.StatusOK, user)
}

// Logout clears the session - GET /auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Optionally revoke token with provider
	identityKey := h.session.GetString(ctx, sessionKeyIdentityKey)
	if identityKey != "" {
		// Could look up stored tokens and revoke, but often not worth it
	}

	_ = h.session.Destroy(ctx)

	// Return JSON for API calls, redirect for browser
	if r.Header.Get("Accept") == "application/json" {
		respondJSON(w, r, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// GetCurrentUser returns the logged-in user from session, or nil
func (h *AuthHandler) GetCurrentUser(r *http.Request) *SessionUser {
	user, ok := h.session.Get(r.Context(), SessionKeyUser).(SessionUser)
	if !ok {
		return nil
	}
	return &user
}

// RequireAuth middleware redirects unauthenticated requests
func (h *AuthHandler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.GetCurrentUser(r) == nil {
			// API requests get 401, browser requests get redirected
			if r.Header.Get("Accept") == "application/json" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			h.session.Put(r.Context(), "redirect_after_login", r.URL.Path)
			http.Redirect(w, r, "/auth/login/github", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *AuthHandler) claimOrphanCommits(userID uuid.UUID, oauth services.OAuthUser) {
	var emails []string
	for _, e := range oauth.Emails {
		if e.Verified {
			emails = append(emails, e.Email)
		}
	}
	if len(emails) > 0 && h.game != nil {
		h.game.ClaimOrphanCommits(context.Background(), userID, emails)
	}
}

func generateRandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func stringFromNull(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

type userCtxKey struct{}

// UserMiddleware adds the current user to context for all requests
func (h *AuthHandler) UserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u := h.GetCurrentUser(r); u != nil {
			ctx := context.WithValue(r.Context(), userCtxKey{}, u)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// UserFromContext gets user from context (can be called anywhere)
func UserFromContext(ctx context.Context) *SessionUser {
	u, _ := ctx.Value(userCtxKey{}).(*SessionUser)
	return u
}
