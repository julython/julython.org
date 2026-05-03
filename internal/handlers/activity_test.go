package handlers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"july/internal/db"
	"july/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivity(t *testing.T) {
	t.Run("renders activity page successfully when no game exists", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		resp := testutil.GetJSON(t, env, "/activity")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body := testutil.DecodeBody(t, resp)
		assert.Contains(t, body, "Recent Activity")
	})

	t.Run("renders activity page with recent commits when game exists", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		// Create a game and some commits to test with
		user, project, game, _ := testutil.CreateGameScenario(t, env)

		// Add a commit to the project
		ctx := context.Background()
		_, err := env.Queries.CreateCommit(ctx, db.CreateCommitParams{
			ID:        db.NewID(),
			Hash:      db.Text("abc123"),
			ProjectID: project.ID,
			UserID:    db.UUID(user.ID),
			GameID:    db.UUID(game.ID),
			Author:    db.Text("testuser"),
			Email:     db.Text("test@example.com"),
			Message:   "Initial commit",
			Url:       "",
			Timestamp: time.Now(),
			Languages: []string{},
			Files:     []byte("[]"),
		})
		require.NoError(t, err)

		resp := testutil.GetJSON(t, env, "/activity")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body := testutil.DecodeBody(t, resp)
		assert.Contains(t, body, "Recent Activity")
		assert.Contains(t, body, "Initial commit")
		assert.Contains(t, body, user.Username)
	})

	t.Run("renders activity page with multiple recent commits", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		// Create a game and some commits to test with
		user, project, game, _ := testutil.CreateGameScenario(t, env)

		// Add multiple commits to the project
		ctx := context.Background()
		for i := 0; i < 5; i++ {
			_, err := env.Queries.CreateCommit(ctx, db.CreateCommitParams{
				ID:        db.NewID(),
				Hash:      db.Text("sha" + string(rune(i+'0'))),
				ProjectID: project.ID,
				UserID:    db.UUID(user.ID),
				GameID:    db.UUID(game.ID),
				Author:    db.Text("testuser"),
				Email:     db.Text("test@example.com"),
				Message:   "Commit message " + string(rune(i+'0')),
				Url:       "",
				Timestamp: time.Now(),
				Languages: []string{},
				Files:     []byte("[]"),
			})
			require.NoError(t, err)
		}

		resp := testutil.GetJSON(t, env, "/activity")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body := testutil.DecodeBody(t, resp)
		assert.Contains(t, body, "Recent Activity")
		// Should contain at least some of the commit messages
		assert.Contains(t, body, "Commit message 0")
		assert.Contains(t, body, "Commit message 1")
	})

	t.Run("renders activity page with proper layout data", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		resp := testutil.GetJSON(t, env, "/activity")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Check that the page has the correct title and layout
		body := testutil.DecodeBody(t, resp)
		assert.Contains(t, body, "<title>Recent Activity | Julython</title>")
		assert.Contains(t, body, "Recent Activity")
	})
}