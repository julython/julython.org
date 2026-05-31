package profile_test

import (
	"net/http"
	"bytes"
	"testing"

	"july/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Profile /overview ───────────────────────────────────────────────

func TestProfileOverview(t *testing.T) {
	t.Run("unauthenticated request redirects to login", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		resp, err := env.Client.Get(env.Server.URL + "/profile")
		require.NoError(t, err)
		defer resp.Body.Close()
		// Redirects to GitHub login (302), which redirects to OAuth provider (307),
		// final response is 200 (OAuth login page).
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("authenticated user sees profile page", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)
		env.LoginAs(t, "test@example.com")

		resp, err := env.Client.Get(env.Server.URL + "/profile")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		// i18n translations are lowercase: "account info", "name", "emails", "username"
		testutil.BodyContains(t, resp, "Test User", "testuser", "account info")
	})
}

// ── Profile /settings ───────────────────────────────────────────────

func TestSettingsPage(t *testing.T) {
	t.Run("unauthenticated request redirects to login", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		resp, err := env.Client.Get(env.Server.URL + "/profile/settings")
		require.NoError(t, err)
		defer resp.Body.Close()
		// Redirects to GitHub login (302), which redirects to OAuth provider (307),
		// final response is 200 (the OAuth login page).
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("authenticated user sees settings page", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)
		env.LoginAs(t, "test@example.com")

		resp, err := env.Client.Get(env.Server.URL + "/profile/settings")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "Settings")
	})
}

func TestUpdateSettings(t *testing.T) {
	t.Run("unauthenticated request returns 401", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		resp := testutil.PostJSON(t, env, "/profile/settings", map[string]string{
			"name": "New Name",
		})
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("empty name returns error", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := testutil.PostJSON(t, env, "/profile/settings", map[string]string{
			"name": "",
		})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "Name cannot be empty")
	})

	t.Run("name over 120 chars returns error", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		longName := ""
		for i := 0; i < 130; i++ {
			longName += "x"
		}
		resp := testutil.PostJSON(t, env, "/profile/settings", map[string]string{
			"name": longName,
		})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "Name must be 120 characters or fewer")
	})

	t.Run("valid name updates successfully (HTMX, fragment)", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "testuser", "Old Name")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		// Simulate HTMX form submission with JSON body.
		body := []byte(`{"name":"New Name"}`)
		req, err := http.NewRequest("POST", env.Server.URL+"/profile/settings", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("HX-Request", "true")

		resp, err := env.Client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		// HTMX fragment shows "success" (lowercase i18n key: profile.saveSuccess)
		testutil.BodyContains(t, resp, "success")
	})

	t.Run("valid name updates successfully (non-HTMX, full page)", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "testuser", "Old Name")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := testutil.PostJSON(t, env, "/profile/settings", map[string]string{
			"name": "New Name",
		})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		// Non-HTMX renders full page with the form (no success message fragment).
		// The input value shows the newly saved name.
		testutil.BodyContains(t, resp, "Settings", "New Name")
	})
}

// ── Profile /webhooks (page shell) ──────────────────────────────────

func TestWebhooksPage(t *testing.T) {
	t.Run("unauthenticated request redirects to login", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		resp, err := env.Client.Get(env.Server.URL + "/profile/webhooks")
		require.NoError(t, err)
		defer resp.Body.Close()
		// Redirects to GitHub login (302), which redirects to OAuth provider (307),
		// final response is 200 (OAuth login page).
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("authenticated user sees webhooks page", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)
		env.LoginAs(t, "test@example.com")

		resp, err := env.Client.Get(env.Server.URL + "/profile/webhooks")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "Webhooks")
	})
}

// ── Profile /webhooks/repos (HTMX endpoint) ────────────────────────

func TestWebhookRepos(t *testing.T) {
	t.Run("unauthenticated request returns 401", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		resp, err := env.Client.Get(env.Server.URL + "/profile/webhooks/repos")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("logged in without GitHub token returns 401", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "testuser", "Test User")
		// No GitHub identifier / token
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp, err := env.Client.Get(env.Server.URL + "/profile/webhooks/repos")
		require.NoError(t, err)
		defer resp.Body.Close()
		// Handler checks for a GitHub OAuth token and returns 401 when absent.
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("logged in with GitHub token calls GitHub API", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)
		testutil.CreateGitHubToken(t, env, user.ID, "ghp_testtoken123")
		env.LoginAs(t, "test@example.com")

		resp, err := env.Client.Get(env.Server.URL + "/profile/webhooks/repos")
		require.NoError(t, err)
		defer resp.Body.Close()
		// A fake token returns 401 from the GitHub API, which the handler
		// translates to a 500 server error. The point of this test is to
		// confirm the request passes authentication and reaches the handler.
		// A 401 from our server would mean no token was found.
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("page parameter accepted", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)
		testutil.CreateGitHubToken(t, env, user.ID, "ghp_testtoken456")
		env.LoginAs(t, "test@example.com")

		resp, err := env.Client.Get(env.Server.URL + "/profile/webhooks/repos?page=2")
		require.NoError(t, err)
		defer resp.Body.Close()
		// Same as above — fake token → 500 from handler. Not 401 (unauthenticated).
		assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// NOTE: The following endpoints require a stored GitHub OAuth token:
//   POST /profile/webhooks/{repoID}/hooks  (AddWebhook)
//   DELETE /profile/webhooks/{repoID}/hooks/{hookID}  (DeleteWebhook)
//
// These would need a real GitHub token with an actual webhook configured on a repo,
// making them integration tests best suited for CI with a real GitHub repo.
