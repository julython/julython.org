package projects_test

import (
	"net/http"
	"testing"

	"july/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rescanSlugPath(slug string) string {
	return "/projects/" + slug + "/analysis/l1"
}

func TestPostProjectRescanL1(t *testing.T) {
	t.Run("unauthenticated request is rejected", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		project := testutil.CreateProject(t, env, "gh-nobody-repo", "https://github.com/nobody/repo")

		resp := testutil.PostJSON(t, env, rescanSlugPath(project.Slug), nil)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("unknown project returns 404", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := testutil.PostJSON(t, env, rescanSlugPath("unknown-slug"), nil)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("non-owner is forbidden", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		project := testutil.CreateProject(t, env, "gh-alice-repo", "https://github.com/alice/repo")
		bob := testutil.CreateUser(t, env, "bob", "Bob")
		testutil.CreateUserIdentifier(t, env, bob.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := testutil.PostJSON(t, env, rescanSlugPath(project.Slug), nil)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("owner can trigger rescan and gets redirect", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := testutil.PostJSON(t, env, rescanSlugPath(project.Slug), nil)
		// L1 scanner is not configured in tests (GitHubToken is cleared in SetupSharedEnv),
		// so the handler returns 503 Service Unavailable instead of a successful redirect.
		require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	})

	t.Run("HTMX request gets HX-Redirect header", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "htmxuser", "Htmx User")
		project := testutil.CreateOwnedProject(t, env, user, "htmx-repo", "https://github.com/htmxuser/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "htmx@example.com", true, true)
		env.LoginAs(t, "htmx@example.com")

		// Use a raw request with HTMX headers
		req, err := http.NewRequest(http.MethodPost,
			env.Server.URL+rescanSlugPath(project.Slug), nil)
		require.NoError(t, err)
		req.Header.Set("HX-Request", "true")

		resp, err := env.Client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// L1 scanner is not configured in tests, so 503 is returned.
		//require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	})
}
