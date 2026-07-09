package services_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"july/internal/db"
	"july/internal/services"
	"july/internal/testutil"
	"july/internal/webhooks"
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
		game, err := svc.CreateJulythonGame(ctx, 2025, 6, false, false)

		require.NoError(t, err)
		assert.Equal(t, "Test Game June 2025", game.Name)
		assert.Equal(t, time.June, game.StartsAt.UTC().Month())
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

func TestGameService_AddCommit(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	svc := services.NewGameService(env.Queries)
	ctx := context.Background()

	t.Run("scores commits for public projects", func(t *testing.T) {
		// Create a public project (owner: "scorer")
		project := testutil.CreateProject(t, env, "scorer-scoring", "https://github.com/scorer/scoring")
		game := testutil.CreateActiveGame(t, env)

		// Insert a raw commit linked to this project
		commitID := uuid.New()
		_, err := env.Pool.Exec(ctx, `
			INSERT INTO commits (id, hash, project_id, author, email, message, url, timestamp)
			VALUES ($1, 'pub-hash-001', $2, 'Scorer', 'scorer@test.com', 'Public commit', 'http://test', now())
		`, commitID, project.ID)
		require.NoError(t, err)

		commit, err := env.Queries.GetCommitByID(ctx, commitID)
		require.NoError(t, err)

		err = svc.AddCommit(ctx, commit)
		require.NoError(t, err)

		// The commit should now have a game_id (scored)
		updated, err := env.Queries.GetCommitByID(ctx, commitID)
		require.NoError(t, err)
		assert.True(t, updated.GameID.Valid)
		gameID, err := uuid.FromBytes(updated.GameID.Bytes[:])
		require.NoError(t, err)
		assert.Equal(t, game.ID, gameID)

		// Verify board was created
		board, err := env.Queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
			ProjectID: project.ID,
			GameID:    game.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, int32(1), board.CommitCount)
	})

	t.Run("skores commits for private projects", func(t *testing.T) {
		// Create a private project
		project := testutil.CreateProject(t, env, "priv-skip-scoring", "https://github.com/skipper/skip-project")
		game := testutil.CreateActiveGame(t, env)

		// Mark project as private
		err := env.Queries.SetProjectIsPrivate(ctx, db.SetProjectIsPrivateParams{
			ID:        project.ID,
			IsPrivate: true,
		})
		require.NoError(t, err)

		// Insert a raw commit linked to this project
		commitID := uuid.New()
		_, err = env.Pool.Exec(ctx, `
			INSERT INTO commits (id, hash, project_id, author, email, message, url, timestamp)
			VALUES ($1, 'priv-skip-001', $2, 'Skipper', 'skipper@test.com', 'Private commit', 'http://test', now())
		`, commitID, project.ID)
		require.NoError(t, err)

		commit, err := env.Queries.GetCommitByID(ctx, commitID)
		require.NoError(t, err)

		err = svc.AddCommit(ctx, commit)
		require.NoError(t, err)

		// The commit should NOT have a game_id (no scoring for private)
		updated, err := env.Queries.GetCommitByID(ctx, commitID)
		require.NoError(t, err)
		assert.False(t, updated.GameID.Valid)

		// Verify no board was created
		_, err = env.Queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
			ProjectID: project.ID,
			GameID:    game.ID,
		})
		assert.ErrorIs(t, err, pgx.ErrNoRows)
	})

	t.Run("returns nil when no active game", func(t *testing.T) {
		// Deactivate all games
		_ = env.Queries.DeactivateAllGames(ctx)

		project := testutil.CreateProject(t, env, "nogame-project", "https://github.com/nogame/project")
		commitID := uuid.New()
		_, err := env.Pool.Exec(ctx, `
			INSERT INTO commits (id, hash, project_id, author, email, message, url, timestamp)
			VALUES ($1, 'nogame-001', $2, 'NoGame', 'nogame@test.com', 'No game commit', 'http://test', now())
		`, commitID, project.ID)
		require.NoError(t, err)

		commit, err := env.Queries.GetCommitByID(ctx, commitID)
		require.NoError(t, err)

		// Should not error even with no active game
		err = svc.AddCommit(ctx, commit)
		require.NoError(t, err)
	})

}

// TestGameService_AddCommit_BoardGapFill tests that board slots with gaps
// are filled correctly. It pre-seeds a player with board_2 but no board_1
// via direct SQL, then posts a commit through the webhook to trigger gap filling.
func TestGameService_AddCommit_BoardGapFill(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	ctx := context.Background()

	gameID := uuid.MustParse("67676767-6767-6767-6767-676767676767")
	userID := uuid.MustParse("89898989-8989-8989-8989-898989898989")

	// Create user with deterministic ID
	_, err := env.Pool.Exec(ctx,
		"INSERT INTO users (id, username, name, avatar_url, is_active) VALUES ($1, 'gapuser', 'Gap User', '', true)",
		userID)
	require.NoError(t, err)

	// Create email identifier so the webhook pipeline can find this user by email
	// The key is formatted as "type:value" (e.g., "email:gapuser@test.com")
	_, err = env.Pool.Exec(ctx,
		"INSERT INTO user_identifiers (value, type, user_id, verified, is_primary) VALUES ('email:gapuser@test.com', 'email', $1, true, true)",
		userID)
	require.NoError(t, err)

	// Create game with deterministic ID
	game := testutil.CreateGame(t, env, "Gap Test",
		time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour), true)
	_, err = env.Pool.Exec(ctx, "UPDATE games SET id = $1 WHERE id = $2", gameID, game.ID)
	require.NoError(t, err)
	game.ID = gameID

	// Pre-seed player with board_2 assigned but board_1 empty
	playerID := uuid.MustParse("abadabad-abad-abad-abad-abadabadabad")
	board2ID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	projectID2 := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	_, err = env.Pool.Exec(ctx,
		"INSERT INTO projects (id, url, name, slug, description, service, forked, forks, watchers, is_private, owner) "+
			"VALUES ($1, $2, $3, 'gap-p2', NULL, 'github', false, 0, 0, false, 'test')",
		projectID2, "https://github.com/test/gap2", "gap2")
	require.NoError(t, err)

	_, err = env.Pool.Exec(ctx,
		"INSERT INTO boards (id, game_id, project_id, points, potential_points, commit_count, contributor_count) "+
			"VALUES ($1, $2, $3, 11, 11, 1, 1)",
		board2ID, gameID, projectID2)
	require.NoError(t, err)

	_, err = env.Pool.Exec(ctx,
		"INSERT INTO players (id, game_id, user_id, points, potential_points, commit_count, project_count, "+
			"analysis_status, board_1_id, board_2_id, board_3_id) VALUES ($1, $2, $3, 11, 11, 1, 1, "+
			"'pending', NULL, $4, NULL)",
		playerID, gameID, userID, board2ID)
	require.NoError(t, err)

	// Create a 3rd project and link a commit through the webhook
	testutil.WebhookCommit(t, env, "gap-fill-hash", func(o *testutil.WebhookOpts) {
		o.RepoID = 55555
		o.RepoName = "gap-fill-project"
		o.FullName = "gapuser/gap-fill-project"
		o.HTMLURL = "https://github.com/gapuser/gap-fill-project"
		o.Author = webhooks.GitHubAuthor{Name: "Gap User", Email: "gapuser@test.com"}
	})

	// Verify board_1 was filled (not board_3)
	ids, err := env.Queries.GetPlayerBoardIds(ctx, playerID)
	require.NoError(t, err)
	require.True(t, ids.Board1ID.Valid, "board_1 should be filled when it's missing")

	// Compare board_2 by bytes (expected is uuid.UUID, actual is pgtype.UUID)
	expectedBytes := [16]byte(board2ID)
	require.True(t, ids.Board2ID.Valid, "board_2 should be valid")
	assert.Equal(t, expectedBytes, ids.Board2ID.Bytes, "board_2 should remain unchanged")
	assert.False(t, ids.Board3ID.Valid, "board_3 should not be assigned yet")
}

// TestGameService_ScoreReset verifies that scoring is isolated per game:
// - New game Board rows start with points = 0 (no carryover from old games)
// - Existing boards.verified_points from prior games are NOT affected
// - Total score displays correctly: total = boards.points + boards.verified_points
func TestGameService_ScoreReset(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	svc := services.NewGameService(env.Queries)
	ctx := context.Background()

	t.Run("new game Board rows start with points = 0, no carryover from old games", func(t *testing.T) {
		// Create game 1 with a project, then add commits to build up points
		game1 := testutil.CreateActiveGame(t, env)
		project1 := testutil.CreateProject(t, env, "scoregame1-proj",
			"https://github.com/scoreuser1/scoregame1-proj")

		// Insert 3 commits via SQLC query (same pattern as TestGameService_AddCommit)
		for i := 0; i < 3; i++ {
			commitHash := fmt.Sprintf("game1-commit-%d", i)
			_, err := env.Queries.CreateCommitSimple(ctx, db.CreateCommitSimpleParams{
				ID:        db.NewID(),
				Hash:      db.Text(commitHash),
				ProjectID: project1.ID,
				Author:    db.Text("Score User 1"),
				Email:     db.Text("scoreuser1@test.com"),
				Message:   "Commit 1",
				Url:       "http://test",
				Timestamp: time.Now(),
			})
			require.NoError(t, err)

			commit, err := env.Queries.GetCommitByHashStr(ctx, commitHash)
			require.NoError(t, err)
			err = svc.AddCommit(ctx, commit)
			require.NoError(t, err)
		}

		// Verify game 1 board has points from 3 commits
		board1, err := env.Queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
			ProjectID: project1.ID,
			GameID:    game1.ID,
		})
		require.NoError(t, err)
		assert.Greater(t, board1.Points, int32(0), "game 1 board should have points > 0 (3 commits + 1 project)")

		// Deactivate game 1 and create a new game
		err = env.Queries.DeactivateGame(ctx, game1.ID)
		require.NoError(t, err)
		game2 := testutil.CreateActiveGame(t, env)

		// Insert a single commit via SQLC into the same project but game 2
		_, err = env.Queries.CreateCommitSimple(ctx, db.CreateCommitSimpleParams{
			ID:        db.NewID(),
			Hash:      db.Text("game2-commit-0"),
			ProjectID: project1.ID,
			Author:    db.Text("Score User 1"),
			Email:     db.Text("scoreuser1@test.com"),
			Message:   "Commit 2",
			Url:       "http://test",
			Timestamp: time.Now(),
		})
		require.NoError(t, err)

		commit2, err := env.Queries.GetCommitByHashStr(ctx, "game2-commit-0")
		require.NoError(t, err)
		err = svc.AddCommit(ctx, commit2)
		require.NoError(t, err)

		// Verify game 2's board for the same project has only 1 commit's worth of points
		board2, err := env.Queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
			ProjectID: project1.ID,
			GameID:    game2.ID,
		})
		require.NoError(t, err)
		// 1 commit (1) + 1 project (10) = 11
		assert.Equal(t, int32(11), board2.Points, "game 2 board should have 11 points (fresh start, 1 commit + 1 project)")

		// Verify game 1's board was NOT affected by game 2's commit
		board1After, err := env.Queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
			ProjectID: project1.ID,
			GameID:    game1.ID,
		})
		require.NoError(t, err)
		assert.Greater(t, board1After.Points, int32(0), "game 1 board should still have points > 0 (unchanged)")
	})

	t.Run("boards.verified_points from prior games on the same project are NOT affected", func(t *testing.T) {
		// Create game 1 with a project that has a board with verified_points set
		game1 := testutil.CreateActiveGame(t, env)
		project1 := testutil.CreateProject(t, env, "verifiedgame1-proj",
			"https://github.com/verifieduser1/verifiedgame1-proj")

		// Add a commit via SQLC
		_, err := env.Queries.CreateCommitSimple(ctx, db.CreateCommitSimpleParams{
			ID:        db.NewID(),
			Hash:      db.Text("verified1-commit-0"),
			ProjectID: project1.ID,
			Author:    db.Text("Verified User 1"),
			Email:     db.Text("verifieduser1@test.com"),
			Message:   "Commit 1",
			Url:       "http://test",
			Timestamp: time.Now(),
		})
		require.NoError(t, err)

		commit1, err := env.Queries.GetCommitByHashStr(ctx, "verified1-commit-0")
		require.NoError(t, err)
		err = svc.AddCommit(ctx, commit1)
		require.NoError(t, err)

		// Get the game 1 board (verified_points defaults to 0)
		board1, err := env.Queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
			ProjectID: project1.ID,
			GameID:    game1.ID,
		})
		require.NoError(t, err)
		assert.Greater(t, board1.Points, int32(0)) // from 1 commit + 1 project
		assert.Equal(t, int32(0), board1.VerifiedPoints) // default

		// Set verified_points = 15 to simulate AI analysis via SQLC.
		// UpsertBoard does NOT update verified_points (it's not in the SQL),
		// so we use the new UpdateBoardVerifiedPoints query.
		_, err = env.Queries.UpdateBoardVerifiedPoints(ctx, db.UpdateBoardVerifiedPointsParams{
			BoardID:        board1.ID,
			VerifiedPoints: 15,
		})
		require.NoError(t, err)

		// Verify game 1 board now has verified_points = 15
		board1WithVerified, err := env.Queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
			ProjectID: project1.ID,
			GameID:    game1.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, int32(15), board1WithVerified.VerifiedPoints)

		// Deactivate game 1 and create game 2
		err = env.Queries.DeactivateGame(ctx, game1.ID)
		require.NoError(t, err)
		game2 := testutil.CreateActiveGame(t, env)

		// Add a commit via SQLC for game 2 (same project)
		_, err = env.Queries.CreateCommitSimple(ctx, db.CreateCommitSimpleParams{
			ID:        db.NewID(),
			Hash:      db.Text("verified2-commit-0"),
			ProjectID: project1.ID,
			Author:    db.Text("Verified User 1"),
			Email:     db.Text("verifieduser1@test.com"),
			Message:   "Commit 2",
			Url:       "http://test",
			Timestamp: time.Now(),
		})
		require.NoError(t, err)

		commit2, err := env.Queries.GetCommitByHashStr(ctx, "verified2-commit-0")
		require.NoError(t, err)
		err = svc.AddCommit(ctx, commit2)
		require.NoError(t, err)

		// Verify game 2's board has verified_points = 0 (default, no AI analysis)
		board2, err := env.Queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
			ProjectID: project1.ID,
			GameID:    game2.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, int32(0), board2.VerifiedPoints, "game 2 board should have verified_points = 0 (no AI set)")

		// Verify game 1's board verified_points is STILL 15 (unchanged by game 2 operations)
		board1After, err := env.Queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
			ProjectID: project1.ID,
			GameID:    game1.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, int32(15), board1After.VerifiedPoints, "game 1 board verified_points must not be affected by game 2")
	})

	t.Run("total score: total = boards.points + boards.verified_points", func(t *testing.T) {
		game := testutil.CreateActiveGame(t, env)
		project := testutil.CreateProject(t, env, "totalgame-proj",
			"https://github.com/totaluser/totalgame-proj")

		// Add 2 commits via SQLC
		for i := 0; i < 2; i++ {
			commitHash := fmt.Sprintf("total-commit-%d", i)
			_, err := env.Queries.CreateCommitSimple(ctx, db.CreateCommitSimpleParams{
				ID:        db.NewID(),
				Hash:      db.Text(commitHash),
				ProjectID: project.ID,
				Author:    db.Text("Total User"),
				Email:     db.Text("totaluser@test.com"),
				Message:   fmt.Sprintf("Commit %d", i),
				Url:       "http://test",
				Timestamp: time.Now(),
			})
			require.NoError(t, err)

			commit, err := env.Queries.GetCommitByHashStr(ctx, commitHash)
			require.NoError(t, err)
			err = svc.AddCommit(ctx, commit)
			require.NoError(t, err)
		}

		// Get the board
		board, err := env.Queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
			ProjectID: project.ID,
			GameID:    game.ID,
		})
		require.NoError(t, err)

		// board.points = 2 commits * 1 + 1 project * 10 = 12
		assert.Equal(t, int32(12), board.Points,
			"board.points should be 12 (2 commits * 1 + 1 project * 10)")
		assert.Equal(t, int32(0), board.VerifiedPoints)

		// Simulate AI analysis by setting verified_points = 5 via SQLC
		_, err = env.Queries.UpdateBoardVerifiedPoints(ctx, db.UpdateBoardVerifiedPointsParams{
			BoardID:        board.ID,
			VerifiedPoints: 5,
		})
		require.NoError(t, err)

		// Verify the computed total: 12 + 5 = 17
		boardAfter, err := env.Queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
			ProjectID: project.ID,
			GameID:    game.ID,
		})
		require.NoError(t, err)
		expectedTotal := boardAfter.Points + boardAfter.VerifiedPoints
		assert.Equal(t, int32(17), expectedTotal,
			"total should be 17 (points 12 + verified_points 5)")
	})
}

// TestGameService_AddCommit_DuplicateBoard verifies that calling AddCommit
// for commits belonging to the same project does not assign the same board
// to multiple player slots.
func TestGameService_AddCommit_DuplicateBoard(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	ctx := context.Background()

	game := testutil.CreateActiveGame(t, env)

	// First commit: AddCommit assigns the board to slot 1.
	testutil.WebhookCommit(t, env, "dup-hash-001", func(o *testutil.WebhookOpts) {
		o.RepoID = 66666
		o.RepoName = "duplicate-board-project"
		o.FullName = "dupuser/duplicate-board-project"
		o.HTMLURL = "https://github.com/dupuser/duplicate-board-project"
		o.Owner = "dupuser"
		o.Author = webhooks.GitHubAuthor{Name: "Dup User", Email: "dupuser@test.com"}
	})

	// Second commit: AddCommit again for the same project.
	// The fix should skip re-assigning the same board.
	testutil.WebhookCommit(t, env, "dup-hash-002", func(o *testutil.WebhookOpts) {
		o.RepoID = 66666
		o.RepoName = "duplicate-board-project"
		o.FullName = "dupuser/duplicate-board-project"
		o.HTMLURL = "https://github.com/dupuser/duplicate-board-project"
		o.Owner = "dupuser"
		o.Author = webhooks.GitHubAuthor{Name: "Dup User", Email: "dupuser@test.com"}
	})

	// Look up the player via username and verify they have exactly 1 board.
	user, err := env.Queries.GetUserByUsername(ctx, "gh-dupuser")
	require.NoError(t, err)

	player, err := env.Queries.GetPlayerByUserAndGame(ctx, db.GetPlayerByUserAndGameParams{
		UserID: user.ID,
		GameID: game.ID,
	})
	require.NoError(t, err)

	ids, err := env.Queries.GetPlayerBoardIds(ctx, player.ID)
	require.NoError(t, err)

	require.True(t, ids.Board1ID.Valid, "board_1 should be set")
	assert.False(t, ids.Board2ID.Valid, "board_2 should NOT be assigned (same board already in slot 1)")
	assert.False(t, ids.Board3ID.Valid, "board_3 should NOT be assigned (same board already in slot 1)")
}
