package players_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"july/internal/db"
	"july/internal/testutil"
)

func TestPlayerRoute(t *testing.T) {
	t.Run("nonexistent user shows empty state", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)
		env.LoginAs(t, "test@example.com")

		resp, err := env.Client.Get(env.Server.URL + "/player/nonexistent")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
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
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("authenticated user with no boards renders empty state", func(t *testing.T) {
		ctx := context.Background()
		env := testutil.SetupTestEnv(t)

		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)

		game := testutil.CreateActiveGame(t, env)

		// Create a player with no boards assigned
		_, err := env.Queries.UpsertPlayer(ctx, db.UpsertPlayerParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			UserID:      user.ID,
			Points:      0,
			CommitCount: 0,
		})
		require.NoError(t, err)

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

		board1, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			ProjectID:   project.ID,
			Points:      15,
			CommitCount: 5,
		})
		require.NoError(t, err)
		board2, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			ProjectID:   project.ID,
			Points:      15,
			CommitCount: 5,
		})
		require.NoError(t, err)
		board3, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			ProjectID:   project.ID,
			Points:      15,
			CommitCount: 5,
		})
		require.NoError(t, err)

		player, err := env.Queries.UpsertPlayer(ctx, db.UpsertPlayerParams{
			ID:             db.NewID(),
			GameID:         game.ID,
			UserID:         user.ID,
			Points:         15,
			CommitCount:    5,
			ProjectCount:   1,
			AnalysisStatus: "pending",
		})
		require.NoError(t, err)

		_, err = env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			Board1ID: db.UUID(board1.ID),
			Board2ID: db.UUID(board2.ID),
			Board3ID: db.UUID(board3.ID),
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

func TestUpdatePlayer(t *testing.T) {
	t.Run("unauthenticated request returns 401", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		resp, err := env.Client.Post(env.Server.URL+"/player/testuser", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("forbidden when swapping another user's boards", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)

		owner := testutil.CreateUser(t, env, "owner", "Owner User")
		testutil.CreateUserIdentifier(t, env, owner.ID, "email", "owner@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, owner.ID, "github", "99999", true, false)

		other := testutil.CreateUser(t, env, "other", "Other User")
		testutil.CreateUserIdentifier(t, env, other.ID, "email", "other@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, other.ID, "github", "88888", true, false)

		game := testutil.CreateActiveGame(t, env)

		ctx := context.Background()
		project := testutil.CreateProject(t, env, "my-project", "https://github.com/owner/my-project")
		board, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			ProjectID:   project.ID,
			Points:      10,
			CommitCount: 3,
		})
		require.NoError(t, err)

		player, err := env.Queries.UpsertPlayer(ctx, db.UpsertPlayerParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			UserID:      owner.ID,
			Points:      10,
			CommitCount: 3,
		})
		require.NoError(t, err)

		// Assign board_1
		b1id := db.UUID(board.ID)
		_, err = env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			Board1ID: b1id,
			PlayerID: player.ID,
		})
		require.NoError(t, err)

		env.LoginAs(t, "other@example.com")

		form := url.Values{}
		form.Set("board_1", board.ID.String())
		resp, err := env.Client.Post(env.Server.URL+"/player/owner", "application/x-www-form-urlencoded", bytes.NewReader([]byte(form.Encode())))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("forbidden when trying to delete another player's board", func(t *testing.T) {
		ctx := context.Background()
		env := testutil.SetupTestEnv(t)

		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)

		game := testutil.CreateActiveGame(t, env)

		// Create a board for a different game
		otherGame := testutil.CreateGame(t, env, "Other Game", game.StartsAt.Add(-48*time.Hour), game.EndsAt.Add(-24*time.Hour), false)
		project := testutil.CreateProject(t, env, "other-project", "https://github.com/testuser/other-project")

		board, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:          db.NewID(),
			GameID:      otherGame.ID,
			ProjectID:   project.ID,
			Points:      10,
			CommitCount: 3,
		})
		require.NoError(t, err)

		// Create a player for this user
		_, err = env.Queries.UpsertPlayer(ctx, db.UpsertPlayerParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			UserID:      user.ID,
			Points:      0,
			CommitCount: 0,
		})
		require.NoError(t, err)

		env.LoginAs(t, "test@example.com")

		// Try to delete a board that isn't in the player's slots → returns 400 (not found)
		form := url.Values{}
		form.Set("delete_board", board.ID.String())
		resp, err := env.Client.Post(env.Server.URL+"/player/testuser", "application/x-www-form-urlencoded", bytes.NewReader([]byte(form.Encode())))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})


	t.Run("delete unlinks a board from the player", func(t *testing.T) {
		ctx := context.Background()
		env := testutil.SetupTestEnv(t)

		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)

		game := testutil.CreateActiveGame(t, env)

		project := testutil.CreateProject(t, env, "my-project", "https://github.com/testuser/my-project")
		board, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			ProjectID:   project.ID,
			Points:      10,
			CommitCount: 3,
		})
		require.NoError(t, err)

		player, err := env.Queries.UpsertPlayer(ctx, db.UpsertPlayerParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			UserID:      user.ID,
			Points:      0,
			CommitCount: 0,
		})
		require.NoError(t, err)

		// Assign board_1
		_, err = env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			PlayerID: player.ID,
			Board1ID: db.UUID(board.ID),
		})
		require.NoError(t, err)

		// Verify board is assigned
		updated, err := env.Queries.GetPlayerByID(ctx, player.ID)
		require.NoError(t, err)
		assert.True(t, updated.Board1ID.Valid)

		env.LoginAs(t, "test@example.com")

		// Delete: unlink the board
		form := url.Values{}
		form.Set("delete_board", board.ID.String())
		resp, err := env.Client.Post(env.Server.URL+"/player/testuser", "application/x-www-form-urlencoded", bytes.NewReader([]byte(form.Encode())))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify board was unlinked (board_1 should be NULL)
		updated, err = env.Queries.GetPlayerByID(ctx, player.ID)
		require.NoError(t, err)
		assert.False(t, updated.Board1ID.Valid, "Board should have been unlinked")
	})
	t.Run("valid delete removes a board from the player", func(t *testing.T) {
		ctx := context.Background()
		env := testutil.SetupTestEnv(t)

		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)

		game := testutil.CreateActiveGame(t, env)

		// Create 3 projects and boards
		var projects []db.Project
		var boards []db.Board
		for i := 1; i <= 3; i++ {
			slug := fmt.Sprintf("project-%d", i)
			project := testutil.CreateProject(t, env, slug, "https://github.com/testuser/"+slug)
			projects = append(projects, project)

			b, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
				ID:          db.NewID(),
				GameID:      game.ID,
				ProjectID:   project.ID,
				Points:      int32(i * 10),
				CommitCount: 3,
			})
			require.NoError(t, err)
			boards = append(boards, b)
		}

		player, err := env.Queries.UpsertPlayer(ctx, db.UpsertPlayerParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			UserID:      user.ID,
			Points:      0,
			CommitCount: 0,
		})
		require.NoError(t, err)

		// Assign all 3 boards
		_, err = env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			PlayerID: player.ID,
			Board1ID: db.UUID(boards[0].ID),
			Board2ID: db.UUID(boards[1].ID),
			Board3ID: db.UUID(boards[2].ID),
		})
		require.NoError(t, err)

		env.LoginAs(t, "test@example.com")

		// Delete board 2 (middle slot)
		form := url.Values{}
		form.Set("delete_board", boards[1].ID.String())
		resp, err := env.Client.Post(env.Server.URL+"/player/testuser", "application/x-www-form-urlencoded", bytes.NewReader([]byte(form.Encode())))
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		testutil.BodyContains(t, resp, "project-1", "project-3")

		// Verify board 2 was unlinked
		updated, err := env.Queries.GetPlayerByID(ctx, player.ID)
		require.NoError(t, err)
		assert.True(t, updated.Board1ID.Valid)
		assert.False(t, updated.Board2ID.Valid, "Board 2 should be unlinked")
		assert.True(t, updated.Board3ID.Valid)
		assert.Equal(t, boards[0].ID, uuid.UUID(updated.Board1ID.Bytes))
		assert.Equal(t, boards[2].ID, uuid.UUID(updated.Board3ID.Bytes))
	})
}
