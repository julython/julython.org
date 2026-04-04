package handlers_test

import (
	"july/internal/testutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHome(t *testing.T) {
	t.Run("empty state renders without error when no game exists", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		// No game, no users — handler falls back to renderEmptyLeaderboard.
		resp := testutil.GetJSON(t, env, "/")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "Code more during July and January. Track your commits. Compete with friends.", "🐍")
	})

	t.Run("shows participant username and game name", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		testutil.CreateGameScenario(t, env)

		resp := testutil.GetJSON(t, env, "/")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "1 commits during Test Game")
	})
}
