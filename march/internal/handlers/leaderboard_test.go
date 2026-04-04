package handlers_test

import (
	"net/http"
	"strings"
	"testing"

	"july/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeaderboard(t *testing.T) {
	t.Run("empty state renders without error when no game exists", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		// No game, no users — handler falls back to renderEmptyLeaderboard.
		resp := testutil.GetJSON(t, env, "/leaders")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "Julython", "🏆")
	})

	t.Run("shows participant username and game name", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user, _, game, _ := testutil.CreateGameScenario(t, env)

		resp := testutil.GetJSON(t, env, "/leaders")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, game.Name, user.Username)
	})

	t.Run("top 3 entries get medal highlight class", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		_, _, game, _ := testutil.CreateGameScenario(t, env)

		// Add two more participants so rank 1–3 are all present.
		testutil.CreateGameScenarioForUser(t, env, game, "silver", "Silver User")
		testutil.CreateGameScenarioForUser(t, env, game, "bronze", "Bronze User")

		resp := testutil.GetJSON(t, env, "/leaders")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body := testutil.DecodeBody(t, resp)

		// templ emits bg-july-500/10 for rank <= 3
		assert.GreaterOrEqual(t, strings.Count(body, "bg-july-500/10"), 3,
			"expected at least 3 highlighted rows for top-3 ranks")
	})

	t.Run("HTMX request returns table fragment only", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		testutil.CreateGameScenario(t, env)

		req, err := http.NewRequest(http.MethodGet, env.Server.URL+"/leaders", nil)
		require.NoError(t, err)
		req.Header.Set("HX-Request", "true")

		resp, err := env.Client.Do(req)
		require.NoError(t, err)
		body := testutil.DecodeBody(t, resp)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		// Fragment must contain the table but NOT the full page layout.
		assert.True(t, strings.Contains(body, "<tbody>"), "expected table body in fragment")
		assert.False(t, strings.Contains(body, "<html"), "HTMX response must not include full page")
	})

	t.Run("pagination: offset moves the rank window", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		_, _, game, _ := testutil.CreateGameScenario(t, env)
		testutil.CreateGameScenarioForUser(t, env, game, "second", "Second User")

		// Page 1 should not contain second-page content; page 2 (offset=1) should.
		p1 := testutil.DecodeBody(t, must200(t, testutil.GetJSON(t, env, "/leaders?offset=0")))
		p2 := testutil.DecodeBody(t, must200(t, testutil.GetJSON(t, env, "/leaders?offset=1")))

		// Offset 1 means rank starts at 2 — medal for rank 1 only on page 1.
		assert.True(t, strings.Contains(p1, "🥇"), "page 1 must show gold medal")
		assert.False(t, strings.Contains(p2, "🥇"), "page 2 must not show gold medal")
	})
}

func TestProjectLeaderboard(t *testing.T) {
	t.Run("empty state renders without error when no game exists", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		resp := testutil.GetJSON(t, env, "/leaders/projects")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "Julython", "📦")
	})

	t.Run("shows project name and slug", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		_, project, _, _ := testutil.CreateGameScenario(t, env)

		resp := testutil.GetJSON(t, env, "/leaders/projects")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, project.Name, project.Slug)
	})

	t.Run("HTMX request returns table fragment only", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		testutil.CreateGameScenario(t, env)

		req, err := http.NewRequest(http.MethodGet, env.Server.URL+"/leaders/projects", nil)
		require.NoError(t, err)
		req.Header.Set("HX-Request", "true")

		resp, err := env.Client.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body := testutil.DecodeBody(t, resp)
		assert.True(t, strings.Contains(body, "<tbody>"))
		assert.False(t, strings.Contains(body, "<html"))
	})
}

func TestLanguageLeaderboard(t *testing.T) {
	t.Run("empty state renders without error when no game exists", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		resp := testutil.GetJSON(t, env, "/leaders/languages")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "Julython", "💻")
	})

	t.Run("shows language names from commits", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		// CreateGameScenario commits with Languages: []string{"Go", "Python"}
		testutil.CreateGameScenario(t, env)

		resp := testutil.GetJSON(t, env, "/leaders/languages")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "Go", "Python")
	})
}
