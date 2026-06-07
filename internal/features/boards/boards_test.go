package boards_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"july/internal/db"
	"july/internal/testutil"
)

func newID() uuid.UUID { return db.NewID() }

func pgid(u uuid.UUID) pgtype.UUID {
	var b [16]byte
	copy(b[:], u[:])
	return pgtype.UUID{Bytes: b, Valid: true}
}

func TestPlayerRoute(t *testing.T) {
	t.Run("unauthenticated redirects to login", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		resp, err := env.Client.Get(env.Server.URL + "/player/testuser")
		require.NoError(t, err)
		defer resp.Body.Close()
		// Redirects to GitHub login (302) → OAuth provider (307) → 200 (login page).
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("missing username returns 404", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)
		env.LoginAs(t, "test@example.com")

		resp, err := env.Client.Get(env.Server.URL + "/player/")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("nonexistent user shows empty state", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)
		env.LoginAs(t, "test@example.com")

		resp, err := env.Client.Get(env.Server.URL + "/player/nonexistent")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "nonexistent")
	})

	t.Run("authenticated user without a player record shows empty state", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)
		// No game, no player record.
		env.LoginAs(t, "test@example.com")

		resp, err := env.Client.Get(env.Server.URL + "/player/testuser")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "No boards yet")
	})

	t.Run("authenticated user with boards renders project info", func(t *testing.T) {
		ctx := context.Background()
		env := testutil.SetupTestEnv(t)

		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)

		game := testutil.CreateActiveGame(t, env)
		project := testutil.CreateProject(t, env, "my-project", "https://github.com/testuser/my-project")

		board, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:          newID(),
			GameID:      game.ID,
			ProjectID:   project.ID,
			Points:      15,
			CommitCount: 5,
		})
		require.NoError(t, err)

		player, err := env.Queries.UpsertPlayer(ctx, db.UpsertPlayerParams{
			ID:             newID(),
			GameID:         game.ID,
			UserID:         user.ID,
			Points:         15,
			CommitCount:    5,
			ProjectCount:   1,
			AnalysisStatus: "pending",
		})
		require.NoError(t, err)

		// Assign board_1
		_, err = env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			Board1ID: pgid(board.ID),
			PlayerID: player.ID,
		})
		require.NoError(t, err)

		env.LoginAs(t, "test@example.com")

		resp, err := env.Client.Get(env.Server.URL + "/player/testuser")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "testuser", "my-project")
	})
}
