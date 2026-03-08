package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"july/internal/db"
	"july/internal/services"
	"july/internal/testutil"
)

func TestGameService_CreateGame(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	svc := services.NewGameService(env.Queries)
	ctx := context.Background()

	t.Run("creates game with valid dates", func(t *testing.T) {
		start := time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)
		end := time.Date(2025, 7, 31, 12, 0, 0, 0, time.UTC)

		game, err := svc.CreateGame(ctx, "Julython 2025", start, end, 1, 10, true, false)

		require.NoError(t, err)
		assert.Equal(t, "Julython 2025", game.Name)
		assert.Equal(t, int32(1), game.CommitPoints)
		assert.Equal(t, int32(10), game.ProjectPoints)
		assert.True(t, game.IsActive)
	})

	t.Run("rejects invalid date range", func(t *testing.T) {
		start := time.Date(2025, 7, 31, 0, 0, 0, 0, time.UTC)
		end := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)

		_, err := svc.CreateGame(ctx, "Invalid", start, end, 1, 10, false, false)

		assert.ErrorIs(t, err, services.ErrInvalidDates)
	})

	t.Run("deactivates other games when requested", func(t *testing.T) {
		// Create first active game
		start1 := time.Date(2024, 7, 1, 12, 0, 0, 0, time.UTC)
		end1 := time.Date(2024, 7, 31, 12, 0, 0, 0, time.UTC)
		game1, err := svc.CreateGame(ctx, "Julython 2024", start1, end1, 1, 10, true, false)
		require.NoError(t, err)
		assert.True(t, game1.IsActive)

		// Create second game with deactivate_others
		start2 := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
		end2 := time.Date(2025, 8, 31, 12, 0, 0, 0, time.UTC)
		game2, err := svc.CreateGame(ctx, "Augustathon 2025", start2, end2, 1, 10, true, true)
		require.NoError(t, err)
		assert.True(t, game2.IsActive)

		// First game should be deactivated
		updated, err := env.Queries.GetGameByID(ctx, game1.ID)
		require.NoError(t, err)
		assert.False(t, updated.IsActive)
	})
}

func TestGameService_CreateJulythonGame(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	svc := services.NewGameService(env.Queries)
	ctx := context.Background()

	t.Run("creates July game", func(t *testing.T) {
		game, err := svc.CreateJulythonGame(ctx, 2025, 7, false, false)

		require.NoError(t, err)
		assert.Equal(t, "Julython 2025", game.Name)
		assert.Equal(t, time.July, game.StartsAt.UTC().Month())
	})

	t.Run("creates January game", func(t *testing.T) {
		game, err := svc.CreateJulythonGame(ctx, 2025, 1, false, false)

		require.NoError(t, err)
		assert.Equal(t, "J(an)ulython 2025", game.Name)
		assert.Equal(t, time.January, game.StartsAt.UTC().Month())
	})

	t.Run("rejects invalid month", func(t *testing.T) {
		_, err := svc.CreateJulythonGame(ctx, 2025, 6, false, false)

		assert.Error(t, err)
	})
}

func TestGameService_GetActiveGame(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	svc := services.NewGameService(env.Queries)
	ctx := context.Background()

	t.Run("returns active game", func(t *testing.T) {
		now := time.Now().UTC()
		start := now.Add(-24 * time.Hour)
		end := now.Add(24 * time.Hour)

		created, err := svc.CreateGame(ctx, "Active Game", start, end, 1, 10, true, false)
		require.NoError(t, err)

		game, err := svc.GetActiveGame(ctx)

		require.NoError(t, err)
		assert.Equal(t, created.ID, game.ID)
	})

	t.Run("returns error when no active game", func(t *testing.T) {
		// Deactivate all games first
		_ = env.Queries.DeactivateAllGames(ctx)

		_, err := svc.GetActiveGame(ctx)

		assert.ErrorIs(t, err, services.ErrNoActiveGame)
	})
}

func TestGameService_UpsertPlayer(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	svc := services.NewGameService(env.Queries)
	ctx := context.Background()

	t.Run("creates player with correct score", func(t *testing.T) {
		// Create isolated test data (no CreateTestScenario which includes a commit)
		user := testutil.CreateUser(t, env, "scoreuser", "Score User")
		project := testutil.CreateProject(t, env, "score-project", "https://github.com/scoreuser/score-project")
		game := testutil.CreateActiveGame(t, env)

		// Create exactly 2 commits
		commit1 := testutil.CreateCommit(t, env, project.ID, "score-abc123", "First commit")
		commit2 := testutil.CreateCommit(t, env, project.ID, "score-def456", "Second commit")

		// Assign these specific commits to user and game
		_, err := env.Pool.Exec(ctx, `
			UPDATE commits SET user_id = $1, game_id = $2
			WHERE id = ANY($3)
		`, user.ID, game.ID, []any{commit1.ID, commit2.ID})
		require.NoError(t, err)

		err = svc.UpsertPlayer(ctx, game, user.ID)
		require.NoError(t, err)

		// Check player score: 2 commits * 1 point + 1 project * 10 points = 12
		player, err := env.Queries.GetPlayerByUserAndGame(ctx, db.GetPlayerByUserAndGameParams{
			UserID: user.ID,
			GameID: game.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, int32(2), player.CommitCount)
		assert.Equal(t, int32(1), player.ProjectCount)
		assert.Equal(t, int32(12), player.Points)
	})

	t.Run("updates existing player score", func(t *testing.T) {
		user := testutil.CreateUser(t, env, "updateuser", "Update User")
		project := testutil.CreateProject(t, env, "update-project", "https://github.com/updateuser/update-project")
		game := testutil.CreateActiveGame(t, env)

		// Create initial commit
		commit1 := testutil.CreateCommit(t, env, project.ID, "update-abc123", "First commit")
		_, _ = env.Pool.Exec(ctx, `UPDATE commits SET user_id = $1, game_id = $2 WHERE id = $3`, user.ID, game.ID, commit1.ID)

		err := svc.UpsertPlayer(ctx, game, user.ID)
		require.NoError(t, err)

		player1, _ := env.Queries.GetPlayerByUserAndGame(ctx, db.GetPlayerByUserAndGameParams{UserID: user.ID, GameID: game.ID})
		assert.Equal(t, int32(1), player1.CommitCount)
		assert.Equal(t, int32(11), player1.Points) // 1*1 + 1*10

		// Add another commit
		commit2 := testutil.CreateCommit(t, env, project.ID, "update-def456", "Second commit")
		_, _ = env.Pool.Exec(ctx, `UPDATE commits SET user_id = $1, game_id = $2 WHERE id = $3`, user.ID, game.ID, commit2.ID)

		err = svc.UpsertPlayer(ctx, game, user.ID)
		require.NoError(t, err)

		player2, _ := env.Queries.GetPlayerByUserAndGame(ctx, db.GetPlayerByUserAndGameParams{UserID: user.ID, GameID: game.ID})
		assert.Equal(t, int32(2), player2.CommitCount)
		assert.Equal(t, int32(12), player2.Points) // 2*1 + 1*10
	})
}

func TestGameService_ClaimOrphanCommits(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	svc := services.NewGameService(env.Queries)
	ctx := context.Background()

	t.Run("claims commits matching user email", func(t *testing.T) {
		user := testutil.CreateUser(t, env, "claimuser", "Claim User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "claim@test.com", true, true)
		project := testutil.CreateProject(t, env, "claim-project", "https://github.com/claimuser/claim-project")
		game := testutil.CreateActiveGame(t, env)

		// Create orphan commit (no user_id) with matching email
		_, err := env.Pool.Exec(ctx, `
			INSERT INTO commits (id, hash, project_id, author, email, message, url, timestamp, game_id, languages, files)
			VALUES (gen_random_uuid(), 'orphan123', $1, 'Claim User', 'claim@test.com', 'Orphan commit', 'http://test', now(), $2, '{}', '[]')
		`, project.ID, game.ID)
		require.NoError(t, err)

		claimed, err := svc.ClaimOrphanCommits(ctx, user.ID, []string{"claim@test.com"})

		require.NoError(t, err)
		assert.Equal(t, int64(1), claimed)

		// Verify player was created
		player, err := env.Queries.GetPlayerByUserAndGame(ctx, db.GetPlayerByUserAndGameParams{UserID: user.ID, GameID: game.ID})
		require.NoError(t, err)
		assert.Equal(t, int32(1), player.CommitCount)
	})

	t.Run("returns zero when no matching commits", func(t *testing.T) {
		user := testutil.CreateUser(t, env, "nomatch", "No Match")
		_ = testutil.CreateActiveGame(t, env)

		claimed, err := svc.ClaimOrphanCommits(ctx, user.ID, []string{"nomatch@test.com"})

		require.NoError(t, err)
		assert.Equal(t, int64(0), claimed)
	})
}
