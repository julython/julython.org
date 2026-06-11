package projects_test

import (
	"net/http"
	"testing"

	"july/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListEmptyProjects(t *testing.T) {
	t.Run("unauthenticated list returns 200 with no projects message", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)

		resp, err := env.Client.Get(env.Server.URL + "/projects")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "No projects")
	})

	t.Run("authenticated user list returns 200 with no projects message", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "user", "User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp, err := env.Client.Get(env.Server.URL + "/projects")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "No projects")
	})

	t.Run("HTMX list returns 200 with no projects message", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)

		req, err := http.NewRequest(http.MethodGet, env.Server.URL+"/projects", nil)
		require.NoError(t, err)
		req.Header.Set("HX-Request", "true")

		resp, err := env.Client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "No projects")
	})
}
