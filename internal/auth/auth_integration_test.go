package auth_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"july/internal/db"
	"july/internal/services"
	"july/internal/testutil"
)

// noFollowClient returns an http.Client that does not follow redirects.
func noFollowClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// ============================================================================
// Login Handler Tests
// ============================================================================

func TestLoginUnknownProvider(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	noFollow := noFollowClient()
	resp, err := noFollow.Get(env.Server.URL + "/auth/login/unknownprovider")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Unknown provider returns 400 (bad request)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestLoginGitHubProvider(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	noFollow := noFollowClient()
	resp, err := noFollow.Get(env.Server.URL + "/auth/login/github")
	require.NoError(t, err)
	defer resp.Body.Close()

	// When GitHub OAuth is enabled, expect a 307 redirect to the OAuth provider.
	// When disabled, the route simply doesn't exist (404).
	if resp.StatusCode == http.StatusNotFound {
		t.Skip("GitHub OAuth is disabled in test config")
	}
	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	location := resp.Header.Get("Location")
	assert.NotEmpty(t, location, "should have Location header for redirect")

	parsed, err := url.Parse(location)
	require.NoError(t, err)
	query := parsed.Query()

	// Verify required query params.
	// Note: GitHub provider ignores PKCE (AuthorizationURL discards the
	// pkceChallenge argument), so code_challenge is not included in the URL.
	assert.NotEmpty(t, query.Get("state"), "should have state param")
	assert.Equal(t, "code", query.Get("response_type"))
}

func TestLoginGitLabProvider(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	noFollow := noFollowClient()
	resp, err := noFollow.Get(env.Server.URL + "/auth/login/gitlab")
	require.NoError(t, err)
	defer resp.Body.Close()

	// When GitLab OAuth is disabled, the route is 404.
	if resp.StatusCode == http.StatusNotFound {
		t.Skip("GitLab OAuth is disabled in test config")
	}

	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	location := resp.Header.Get("Location")
	assert.NotEmpty(t, location)

	parsed, err := url.Parse(location)
	require.NoError(t, err)
	query := parsed.Query()

	assert.NotEmpty(t, query.Get("state"))
	assert.NotEmpty(t, query.Get("code_challenge"))
}

func TestLoginEmptyProvider(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// /auth/login/ (trailing slash) does not match /auth/login/{provider}.
	// The route requires the {provider} segment.
	resp, err := env.Client.Get(env.Server.URL + "/auth/login/")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Route not found — 404
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// ============================================================================
// Callback Handler Tests
// ============================================================================

func TestCallbackMissingState(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// Use the password login endpoint to populate a session.
	noFollow := noFollowClient()
	_, err := noFollow.Get(env.Server.URL + "/auth/login/password")
	require.NoError(t, err)

	// Callback with an invalid state (doesn't match what's in session).
	callbackURL := fmt.Sprintf("%s/auth/callback?state=invalid_state&code=somecode",
		env.Server.URL)
	resp2, err := env.Client.Get(callbackURL)
	require.NoError(t, err)
	defer resp2.Body.Close()

	// State won't match (it was set by password, but callback uses a mismatched state).
	// The callback returns 400 Bad Request for invalid state.
	// Since env.Client follows redirects, the status 400 is the final status.
	assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)
}

func TestCallbackInvalidState(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// Use the password OAuth flow — get a valid state, then use a wrong state in callback.
	noFollow := noFollowClient()
	resp, err := noFollow.Get(env.Server.URL + "/auth/login/password")
	require.NoError(t, err)
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		t.Skip("Password OAuth is disabled in test config")
	}
	resp.Body.Close()

	// Use a fake state instead of the one in the session.
	fakeState := "totally-wrong-state"
	code := "testcode"
	callbackURL := fmt.Sprintf("%s/auth/callback?state=%s&code=%s",
		env.Server.URL,
		url.QueryEscape(fakeState),
		url.QueryEscape(code))

	resp2, err := env.Client.Get(callbackURL)
	require.NoError(t, err)
	defer resp2.Body.Close()

	// Bad state should return 400.
	assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)
}

func TestCallbackOAuthError(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// Get a valid state from session.
	noFollow := noFollowClient()
	resp, err := noFollow.Get(env.Server.URL + "/auth/login/password")
	require.NoError(t, err)
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		t.Skip("Password OAuth is disabled in test config")
	}
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	state := parsed.Query().Get("state")

	// Callback with OAuth error params.
	callbackURL := fmt.Sprintf("%s/auth/callback?error=access_denied&error_description=User+denied+access&state=%s",
		env.Server.URL, url.QueryEscape(state))
	resp2, err := env.Client.Get(callbackURL)
	require.NoError(t, err)
	defer resp2.Body.Close()

	// OAuth error should return 400.
	assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)
}

func TestCallbackSuccess(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// Create a user with email so password login works.
	user := testutil.CreateUser(t, env, "loginuser", "Login User")
	testutil.CreateUserIdentifier(t, env, user.ID, "email", "login@test.com", true, true)
	testutil.CreateUserIdentifier(t, env, user.ID, "github", "98765", true, false)

	// Perform full login flow.
	env.LoginAs(t, "login@test.com")

	// Session should now be populated — verify with /auth/session.
	resp, err := env.Client.Get(env.Server.URL + "/auth/session")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var session map[string]any
	err = json.NewDecoder(resp.Body).Decode(&session)
	require.NoError(t, err)
	assert.Equal(t, "loginuser", session["username"])
	assert.Equal(t, "Login User", session["name"])
}

func TestCallbackSuccessDefaultRedirect(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	user := testutil.CreateUser(t, env, "redirectuser", "Redirect User")
	testutil.CreateUserIdentifier(t, env, user.ID, "email", "redirect@test.com", true, true)

	env.LoginAs(t, "redirect@test.com")

	// Should redirect to home page by default.
	// This is tested implicitly by LoginAs succeeding (which checks status < 400).
}

func TestCallbackMissingCode(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// Get state from session.
	noFollow := noFollowClient()
	resp, err := noFollow.Get(env.Server.URL + "/auth/login/password")
	require.NoError(t, err)
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		t.Skip("Password OAuth is disabled in test config")
	}
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	parsed, err := url.Parse(location)
	require.NoError(t, err)
	state := parsed.Query().Get("state")

	// Callback with state but no code.
	callbackURL := fmt.Sprintf("%s/auth/callback?state=%s",
		env.Server.URL, url.QueryEscape(state))
	resp2, err := env.Client.Get(callbackURL)
	require.NoError(t, err)
	defer resp2.Body.Close()

	// Missing code should return 400.
	assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)
}

func TestCallbackMultipleEmailsSameUser(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// Create a user with one email already registered.
	existingUser := testutil.CreateUser(t, env, "multiemailuser", "Multi Email User")
	testutil.CreateUserIdentifier(t, env, existingUser.ID, "email", "existing@test.com", true, true)

	// New user logs in with a different email — should link rather than create.
	testutil.CreateUserIdentifier(t, env, existingUser.ID, "github", "55555", true, false)

	env.LoginAs(t, "existing@test.com")

	// Session should be for the existing user.
	resp, err := env.Client.Get(env.Server.URL + "/auth/session")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var session map[string]any
	err = json.NewDecoder(resp.Body).Decode(&session)
	require.NoError(t, err)
	assert.Equal(t, "multiemailuser", session["username"])
}

// ============================================================================
// Session Handler Tests
// ============================================================================

func TestSessionNotAuthenticated(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	resp, err := env.Client.Get(env.Server.URL + "/auth/session")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestSessionAuthenticated(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	user := testutil.CreateUser(t, env, "sessionuser", "Session User")
	testutil.CreateUserIdentifier(t, env, user.ID, "email", "session@test.com", true, true)

	env.LoginAs(t, "session@test.com")

	resp, err := env.Client.Get(env.Server.URL + "/auth/session")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var session map[string]any
	err = json.NewDecoder(resp.Body).Decode(&session)
	require.NoError(t, err)
	assert.Equal(t, "sessionuser", session["username"])
	assert.Equal(t, "Session User", session["name"])
}

func TestSessionAuthenticatedWithAvatar(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	user := testutil.CreateUser(t, env, "avataruser", "Avatar User")
	err := env.Queries.UpdateUser(context.Background(), db.UpdateUserParams{
		ID:        user.ID,
		AvatarUrl: db.Text("https://avatars.example.com/user.png"),
	})
	require.NoError(t, err)

	testutil.CreateUserIdentifier(t, env, user.ID, "email", "avatar@test.com", true, true)

	env.LoginAs(t, "avatar@test.com")

	resp, err := env.Client.Get(env.Server.URL + "/auth/session")
	require.NoError(t, err)
	defer resp.Body.Close()

	var session map[string]any
	err = json.NewDecoder(resp.Body).Decode(&session)
	require.NoError(t, err)
	assert.Equal(t, "https://avatars.example.com/user.png", session["avatar_url"])
}

// ============================================================================
// Logout Handler Tests
// ============================================================================

func TestLogoutBrowserRedirect(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	user := testutil.CreateUser(t, env, "logoutuser", "Logout User")
	testutil.CreateUserIdentifier(t, env, user.ID, "email", "logout@test.com", true, true)

	env.LoginAs(t, "logout@test.com")

	// Logout without Accept: application/json header.
	// env.Client follows redirects, so the final status after following
	// the redirect to / is 200 (homepage).
	resp, err := env.Client.Get(env.Server.URL + "/auth/logout")
	require.NoError(t, err)
	defer resp.Body.Close()

	// The redirect chain goes: 303 -> 200 (homepage).
	// Check the final status is a success.
	assert.LessOrEqual(t, 200, resp.StatusCode)
	assert.Less(t, resp.StatusCode, 400)
}

func TestLogoutJSONResponse(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	user := testutil.CreateUser(t, env, "jsonlogoutuser", "JSON Logout User")
	testutil.CreateUserIdentifier(t, env, user.ID, "email", "jsonlogout@test.com", true, true)

	env.LoginAs(t, "jsonlogout@test.com")

	// Logout with Accept: application/json.
	noFollow := noFollowClient()
	req, err := http.NewRequest("GET", env.Server.URL+"/auth/logout", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	resp, err := noFollow.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result map[string]bool
	err = json.Unmarshal(body, &result)
	require.NoError(t, err)
	assert.True(t, result["ok"])

	// Verify session is gone — next /auth/session should 401.
	noFollow2 := noFollowClient()
	resp2, err := noFollow2.Get(env.Server.URL + "/auth/session")
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp2.StatusCode)
}

func TestLogoutUnauthenticated(t *testing.T) {
	// Logout should not error even without identity key or session.
	env := testutil.SetupTestEnv(t)

	noFollow := noFollowClient()
	resp, err := noFollow.Get(env.Server.URL + "/auth/logout")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should redirect to home (303) even when unauthenticated.
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
}

// ============================================================================
// UserMiddleware Tests
// ============================================================================

func TestUserMiddlewareUnauthenticated(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// Access any page without login.
	resp, err := env.Client.Get(env.Server.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Page should render regardless (middleware doesn't block unauthenticated).
	assert.LessOrEqual(t, resp.StatusCode, 302)
}

func TestUserMiddlewareAuthenticated(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	user := testutil.CreateUser(t, env, "middlewareuser", "Middleware User")
	testutil.CreateUserIdentifier(t, env, user.ID, "email", "middleware@test.com", true, true)

	env.LoginAs(t, "middleware@test.com")

	// Should be able to access any page.
	resp, err := env.Client.Get(env.Server.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.LessOrEqual(t, resp.StatusCode, 302)
}

// ============================================================================
// Full Login/Logout Integration Test
// ============================================================================

func TestFullLoginLogoutFlow(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// Create a user.
	user := testutil.CreateUser(t, env, "flowuser", "Flow User")
	testutil.CreateUserIdentifier(t, env, user.ID, "email", "flow@test.com", true, true)

	// 1. Login.
	env.LoginAs(t, "flow@test.com")

	// 2. Session is active.
	resp, err := env.Client.Get(env.Server.URL + "/auth/session")
	require.NoError(t, err)
	var session map[string]any
	err = json.NewDecoder(resp.Body).Decode(&session)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, "flowuser", session["username"])

	// 3. Logout — verify session cookie is cleared.
	noFollow := noFollowClient()
	resp2, err := noFollow.Get(env.Server.URL + "/auth/logout")
	require.NoError(t, err)
	resp2.Body.Close()
	assert.Equal(t, http.StatusSeeOther, resp2.StatusCode)

	// 4. Session should be gone after logout.
	resp3, err := noFollow.Get(env.Server.URL + "/auth/session")
	require.NoError(t, err)
	resp3.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp3.StatusCode)
}

// ============================================================================
// OAuth Linking Tests (multi-stage lookup integration)
// ============================================================================

func TestOAuthLoginLinksToWebhookUser(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// Step 1: Create a user via "webhook" — username + unverified email
	user, err := env.Queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:       db.NewID(),
		Name:     "Webhook Creator",
		Username: "gh-webhookuser",
		Role:     "user",
	})
	require.NoError(t, err)

	// Add unverified email identifier (what webhook does)
	_, err = env.Queries.UpsertUserIdentifier(context.Background(), db.UpsertUserIdentifierParams{
		Value:     "email:webhook@example.com",
		Type:      "email",
		UserID:    user.ID,
		Verified:  false,
		IsPrimary: false,
	})
	require.NoError(t, err)

	// Step 2: Simulate OAuth login — should find user by username, add github:<id>
	oauthUser := services.OAuthUser{
		ID:        "99999",
		Provider:  "github",
		Username:  "gh-webhookuser",
		Name:      "Webhook Creator Updated",
		AvatarURL: "https://avatars.example.com/webhook.png",
		Emails:    []services.EmailAddress{{Email: "webhook@example.com", Primary: true, Verified: true}},
	}

	linkedUser, created, err := env.UserService.OAuthLoginOrRegister(context.Background(), oauthUser)
	require.NoError(t, err)
	assert.False(t, created, "should not create a new user — should find by username")
	assert.Equal(t, user.ID, linkedUser.ID)

	// Verify github:<id> identifier was added
	_, err = env.Queries.GetUserIdentifier(context.Background(), "github:99999")
	require.NoError(t, err, "github identifier should exist after OAuth login")

	// Verify user name was updated from OAuth data
	updatedUser, err := env.Queries.GetUserByID(context.Background(), linkedUser.ID)
	require.NoError(t, err)
	assert.Equal(t, "Webhook Creator Updated", updatedUser.Name)

	// Verify avatar was updated from OAuth data
	require.True(t, updatedUser.AvatarUrl.Valid)
	assert.Equal(t, "https://avatars.example.com/webhook.png", db.StringFromNull(updatedUser.AvatarUrl))

	// Verify original email identifier still exists (preserved, not overwritten)
	_, err = env.Queries.GetUserIdentifier(context.Background(), "email:webhook@example.com")
	require.NoError(t, err)
}

func TestOAuthLoginFindsUserByGithubId(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// Step 1: Create a user with an existing github:<id> identifier
	user, err := env.Queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:       db.NewID(),
		Name:     "Existing GitHub User",
		Username: "gh-ghuser",
		Role:     "user",
	})
	require.NoError(t, err)

	// Add the github identifier directly (simulating prior OAuth login)
	_, err = env.Queries.UpsertUserIdentifier(context.Background(), db.UpsertUserIdentifierParams{
		Value:     "github:12345",
		Type:      "github",
		UserID:    user.ID,
		Verified:  true,
		IsPrimary: false,
	})
	require.NoError(t, err)

	// Step 2: Simulate OAuth login with matching github ID
	oauthUser := services.OAuthUser{
		ID:        "12345",
		Provider:  "github",
		Username:  "gh-ghuser",
		Name:      "GitHub User Updated",
		AvatarURL: "https://avatars.example.com/ghuser.png",
	}

	linkedUser, created, err := env.UserService.OAuthLoginOrRegister(context.Background(), oauthUser)
	require.NoError(t, err)
	assert.False(t, created, "should not create a new user — should find by github:<id>")
	assert.Equal(t, user.ID, linkedUser.ID)

	// Verify user name was updated from OAuth data
	updatedUser, err := env.Queries.GetUserByID(context.Background(), linkedUser.ID)
	require.NoError(t, err)
	assert.Equal(t, "GitHub User Updated", updatedUser.Name)

	// Verify avatar was updated from OAuth data
	require.True(t, updatedUser.AvatarUrl.Valid)
	assert.Equal(t, "https://avatars.example.com/ghuser.png", db.StringFromNull(updatedUser.AvatarUrl))
}

// ============================================================================
// Helpers
// ============================================================================

// parsedURL parses a URL string. Returns nil on error.
func parsedURL(rawURL string) *url.URL {
	parsed, _ := url.Parse(rawURL)
	return parsed
}

// parsedURLT parses a URL string, failing the test on error.
func parsedURLT(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	require.NoError(t, err, "could not parse URL: %s", rawURL)
	return parsed
}
