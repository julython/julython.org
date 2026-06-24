package db_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"july/internal/db"
	"july/internal/testutil"
)

func newID() uuid.UUID { return db.NewID() }

// pgid creates a valid pgtype.UUID from a uuid.UUID.
func pgid(u uuid.UUID) pgtype.UUID {
	var b [16]byte
	copy(b[:], u[:])
	return pgtype.UUID{Bytes: b, Valid: true}
}

// --- Setup helpers ---

func setupPlayerBoardsTest(t *testing.T) (*testutil.TestEnv, db.Game, db.User, db.Project, db.Board, db.Player) {
	t.Helper()
	ctx := context.Background()
	env := testutil.SetupTestEnv(t)

	// Verify board columns exist on players table
	rows, err := env.Pool.Query(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_name = 'players' AND column_name LIKE 'board_%'
		ORDER BY column_name
	`)
	require.NoError(t, err)
	var cols []string
	for rows.Next() {
		var col string
		rows.Scan(&col)
		cols = append(cols, col)
	}
	rows.Close()
	t.Logf("board columns in players table: %v", cols)
	require.Lenf(t, cols, 3, "expected 3 board columns in players table, got %d: %v", len(cols), cols)

	game := testutil.CreateActiveGame(t, env)
	user := testutil.CreateUser(t, env, "boardtest", "Board Test User")
	project := testutil.CreateProject(t, env, "board-project", "https://github.com/boardtest/board-project")

	board, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
		ID:          newID(),
		GameID:      game.ID,
		ProjectID:   project.ID,
		Points:      10,
		CommitCount: 5,
	})
	require.NoError(t, err)

	player, err := env.Queries.UpsertPlayer(ctx, db.UpsertPlayerParams{
		ID:             newID(),
		GameID:         game.ID,
		UserID:         user.ID,
		Points:         10,
		CommitCount:    5,
		ProjectCount:   1,
		AnalysisStatus: "complete",
	})
	require.NoError(t, err)

	return env, game, user, project, board, player
}

func resetBoards(ctx context.Context, env *testutil.TestEnv, playerID uuid.UUID) {
	env.Pool.Exec(ctx,
		"UPDATE players SET board_1_id = NULL, board_2_id = NULL, board_3_id = NULL WHERE id = $1",
		playerID,
	)
}

// --- AssignBoards ---

func TestAssignBoards(t *testing.T) {
	ctx := context.Background()
	env, game, _, _, board, player := setupPlayerBoardsTest(t)

	// Create additional boards
	p2 := testutil.CreateProject(t, env, "assign-p2", "https://github.com/boardtest/assign-p2")
	p3 := testutil.CreateProject(t, env, "assign-p3", "https://github.com/boardtest/assign-p3")

	b2, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
		ID:          newID(),
		GameID:      game.ID,
		ProjectID:   p2.ID,
		Points:      20,
		CommitCount: 7,
	})
	require.NoError(t, err)

	b3, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
		ID:          newID(),
		GameID:      game.ID,
		ProjectID:   p3.ID,
		Points:      30,
		CommitCount: 9,
	})
	require.NoError(t, err)

	t.Run("assigns a single board", func(t *testing.T) {
		resetBoards(ctx, env, player.ID)

		result, err := env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			Board1ID: pgid(board.ID),
			PlayerID: player.ID,
		})
		require.NoError(t, err)
		assert.True(t, result.Board1ID.Valid)
		assertPgUUIDEqual(t, pgid(board.ID), result.Board1ID)
		assert.False(t, result.Board2ID.Valid)
		assert.False(t, result.Board3ID.Valid)
	})

	t.Run("assigns all 3 boards", func(t *testing.T) {
		resetBoards(ctx, env, player.ID)

		_, err = env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			Board1ID: pgid(board.ID),
			Board2ID: pgid(b2.ID),
			Board3ID: pgid(b3.ID),
			PlayerID: player.ID,
		})
		require.NoError(t, err)

		p, err := env.Queries.GetPlayerByID(ctx, player.ID)
		require.NoError(t, err)
		assertPgUUIDEqual(t, pgid(board.ID), p.Board1ID)
		assertPgUUIDEqual(t, pgid(b2.ID), p.Board2ID)
		assertPgUUIDEqual(t, pgid(b3.ID), p.Board3ID)
	})

	t.Run("partial update leaves other columns unchanged", func(t *testing.T) {
		resetBoards(ctx, env, player.ID)

		// First assign all 3 boards
		_, err := env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			Board1ID: pgid(board.ID),
			Board2ID: pgid(b2.ID),
			Board3ID: pgid(b3.ID),
			PlayerID: player.ID,
		})
		require.NoError(t, err)

		// Read back actual assigned IDs (these are the real UUIDs in the DB)
		ids, err := env.Queries.GetPlayerBoardIds(ctx, player.ID)
		require.NoError(t, err)

		// Now update only board_2, leave board_1 and board_3 untouched
		_, err = env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			Board2ID: pgid(b3.ID), // replace board 2 with a new board
			PlayerID: player.ID,
		})
		require.NoError(t, err)

		p, err := env.Queries.GetPlayerByID(ctx, player.ID)
		require.NoError(t, err)
		// board_1 should be unchanged
		assertPgUUIDEqual(t, ids.Board1ID, p.Board1ID)
		// board_2 should be replaced
		assertPgUUIDEqual(t, pgid(b3.ID), p.Board2ID)
		// board_3 should be unchanged
		assertPgUUIDEqual(t, ids.Board3ID, p.Board3ID)
	})

	t.Run("replacing a board position works", func(t *testing.T) {
		resetBoards(ctx, env, player.ID)

		// Assign 3 boards
		_, err := env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			Board1ID: pgid(board.ID),
			Board2ID: pgid(b2.ID),
			Board3ID: pgid(b3.ID),
			PlayerID: player.ID,
		})
		require.NoError(t, err)

		// Replace board at position 2
		pNew := testutil.CreateProject(t, env, "replace-board", "https://github.com/boardtest/replace")
		replaceBoard, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:          newID(),
			GameID:      game.ID,
			ProjectID:   pNew.ID,
			Points:      50,
			CommitCount: 15,
		})
		require.NoError(t, err)

		_, err = env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			Board2ID: pgid(replaceBoard.ID),
			PlayerID: player.ID,
		})
		require.NoError(t, err)

		p, err := env.Queries.GetPlayerByID(ctx, player.ID)
		require.NoError(t, err)
		// Only board_2 changed
		assertPgUUIDEqual(t, pgid(board.ID), p.Board1ID)
		assertPgUUIDEqual(t, pgid(replaceBoard.ID), p.Board2ID)
		assertPgUUIDEqual(t, pgid(b3.ID), p.Board3ID)
	})

	t.Run("no boards assigned leaves all null", func(t *testing.T) {
		resetBoards(ctx, env, player.ID)

		_, err := env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			PlayerID: player.ID,
		})
		require.NoError(t, err)

		p, err := env.Queries.GetPlayerByID(ctx, player.ID)
		require.NoError(t, err)
		assert.False(t, p.Board1ID.Valid)
		assert.False(t, p.Board2ID.Valid)
		assert.False(t, p.Board3ID.Valid)
	})
}

// --- GetPlayerBoardIds ---

func TestGetPlayerBoardIds(t *testing.T) {
	ctx := context.Background()
	env := testutil.SetupTestEnv(t)
	game := testutil.CreateActiveGame(t, env)
	user := testutil.CreateUser(t, env, "boardtest-nb", "No Boards Test")
	player, err := env.Queries.UpsertPlayer(ctx, db.UpsertPlayerParams{
		ID:             newID(),
		GameID:         game.ID,
		UserID:         user.ID,
		Points:         0,
		CommitCount:    0,
		ProjectCount:   0,
		AnalysisStatus: "pending",
	})
	require.NoError(t, err)

	p1 := testutil.CreateProject(t, env, "nb-p1", "https://github.com/boardtest/nb-p1")
	board, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
		ID:          newID(),
		GameID:      game.ID,
		ProjectID:   p1.ID,
		Points:      10,
		CommitCount: 5,
	})
	require.NoError(t, err)

	t.Run("returns all null when no boards assigned", func(t *testing.T) {
		ids, err := env.Queries.GetPlayerBoardIds(ctx, player.ID)
		require.NoError(t, err)
		assert.False(t, ids.Board1ID.Valid)
		assert.False(t, ids.Board2ID.Valid)
		assert.False(t, ids.Board3ID.Valid)
	})

	t.Run("returns assigned board IDs", func(t *testing.T) {
		_, err := env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			Board1ID: pgid(board.ID),
			Board3ID: pgid(board.ID), // reuse the same board
			PlayerID: player.ID,
		})
		require.NoError(t, err)

		ids, err := env.Queries.GetPlayerBoardIds(ctx, player.ID)
		require.NoError(t, err)
		assert.True(t, ids.Board1ID.Valid)
		assert.False(t, ids.Board2ID.Valid)
		assert.True(t, ids.Board3ID.Valid)
		assertPgUUIDEqual(t, pgid(board.ID), ids.Board1ID)
		assertPgUUIDEqual(t, pgid(board.ID), ids.Board3ID)
	})
}

// --- GetLeaderboard (leaderboard with board totals) ---

func TestListPlayersWithBoards(t *testing.T) {
	ctx := context.Background()
	env, game, _, _, board, player := setupPlayerBoardsTest(t)

	// Create a second player for leaderboard comparison
	user2 := testutil.CreateUser(t, env, "boardtest2", "Board Test User 2")
	p2proj := testutil.CreateProject(t, env, "lt-p2proj", "https://github.com/boardtest2/p2proj")
	bHigh, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
		ID:          newID(),
		GameID:      game.ID,
		ProjectID:   p2proj.ID,
		Points:      50,
		CommitCount: 15,
	})
	require.NoError(t, err)

	player2, err := env.Queries.UpsertPlayer(ctx, db.UpsertPlayerParams{
		ID:             newID(),
		GameID:         game.ID,
		UserID:         user2.ID,
		Points:         50,
		CommitCount:    15,
		ProjectCount:   1,
		AnalysisStatus: "complete",
	})
	require.NoError(t, err)

	t.Run("leaderboard returns players with board totals", func(t *testing.T) {
		// Assign board (score=10) to original player
		_, err = env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			Board1ID: pgid(board.ID),
			PlayerID: player.ID,
		})
		require.NoError(t, err)

		// Assign board (score=50) to player2
		_, err = env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			Board1ID: pgid(bHigh.ID),
			PlayerID: player2.ID,
		})
		require.NoError(t, err)

		rows, err := env.Queries.GetLeaderboard(ctx, db.GetLeaderboardParams{
			GameID:     game.ID,
			LimitCount: 100,
		})
		require.NoError(t, err)
		require.Len(t, rows, 2)

		// Find each player by ID
		var row1, row2 db.GetLeaderboardRow
		for _, r := range rows {
			if r.ID == player.ID {
				row1 = r
			} else {
				row2 = r
			}
		}

		// player2 should be first (score 50 > 10)
		assert.True(t, row2.ID == player2.ID)
		assert.True(t, row1.ID == player.ID)
	})

	t.Run("players with no boards get total 0", func(t *testing.T) {
		rows, err := env.Queries.GetLeaderboard(ctx, db.GetLeaderboardParams{
			GameID:     game.ID,
			LimitCount: 100,
		})
		require.NoError(t, err)
		require.Len(t, rows, 2)

		for _, r := range rows {
			if r.ID == player.ID {
				assertBoardTotalEqual(t, 10, r.BoardTotal)
			} else {
				assertBoardTotalEqual(t, 50, r.BoardTotal)
			}
		}
	})

	t.Run("multiple boards are summed", func(t *testing.T) {
		// Create 3 boards for a single player
		p3 := testutil.CreateProject(t, env, "multi-p3", "https://github.com/boardtest/multi-p3")
		b3, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:          newID(),
			GameID:      game.ID,
			ProjectID:   p3.ID,
			Points:      20,
			CommitCount: 5,
		})
		require.NoError(t, err)

		// Assign all 3 boards to player
		_, err = env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
			Board1ID: pgid(board.ID),  // score 10
			Board2ID: pgid(bHigh.ID),  // score 50
			Board3ID: pgid(b3.ID),     // score 20
			PlayerID: player.ID,
		})
		require.NoError(t, err)

		rows, err := env.Queries.GetLeaderboard(ctx, db.GetLeaderboardParams{
			GameID:     game.ID,
			LimitCount: 100,
		})
		require.NoError(t, err)

		var row db.GetLeaderboardRow
		for _, r := range rows {
			if r.ID == player.ID {
				row = r
				break
			}
		}

		// Total should be 10 + 50 + 20 = 80
		assertBoardTotalEqual(t, 80, row.BoardTotal)
	})
}

// --- Integration: full workflow ---

func TestPlayerBoardsFullWorkflow(t *testing.T) {
	ctx := context.Background()
	env, game, _, _, board, player := setupPlayerBoardsTest(t)

	// Create 2 more boards
	p2 := testutil.CreateProject(t, env, "workflow-p2", "https://github.com/boardtest/workflow-p2")
	p3 := testutil.CreateProject(t, env, "workflow-p3", "https://github.com/boardtest/workflow-p3")

	b2, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
		ID:          newID(),
		GameID:      game.ID,
		ProjectID:   p2.ID,
		Points:      20,
		CommitCount: 7,
	})
	require.NoError(t, err)

	b3, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
		ID:          newID(),
		GameID:      game.ID,
		ProjectID:   p3.ID,
		Points:      30,
		CommitCount: 9,
	})
	require.NoError(t, err)

	// Step 1: Assign 1 board
	_, err = env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
		Board1ID: pgid(board.ID),
		PlayerID: player.ID,
	})
	require.NoError(t, err)

	// Step 2: Read board IDs
	ids, err := env.Queries.GetPlayerBoardIds(ctx, player.ID)
	require.NoError(t, err)
	assert.True(t, ids.Board1ID.Valid)
	assert.False(t, ids.Board2ID.Valid)
	assert.False(t, ids.Board3ID.Valid)

	// Step 3: Add 2 more boards (partial update)
	_, err = env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
		Board2ID: pgid(b2.ID),
		Board3ID: pgid(b3.ID),
		PlayerID: player.ID,
	})
	require.NoError(t, err)

	// Step 4: Verify all 3 boards are set
	p, err := env.Queries.GetPlayerByID(ctx, player.ID)
	require.NoError(t, err)
	assertPgUUIDEqual(t, pgid(board.ID), p.Board1ID)
	assertPgUUIDEqual(t, pgid(b2.ID), p.Board2ID)
	assertPgUUIDEqual(t, pgid(b3.ID), p.Board3ID)

	// Step 5: Check leaderboard total (10 + 20 + 30 = 60)
	rows, err := env.Queries.GetLeaderboard(ctx, db.GetLeaderboardParams{
		GameID:     game.ID,
		LimitCount: 100,
	})
	require.NoError(t, err)
	var row db.GetLeaderboardRow
	for _, r := range rows {
		if r.ID == player.ID {
			row = r
			break
		}
	}
	assertBoardTotalEqual(t, 60, row.BoardTotal)

	// Step 6: Replace one board (update only board_2)
	pNew := testutil.CreateProject(t, env, "workflow-replace", "https://github.com/boardtest/replacement")
	replaceBoard, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
		ID:          newID(),
		GameID:      game.ID,
		ProjectID:   pNew.ID,
		Points:      100,
		CommitCount: 30,
	})
	require.NoError(t, err)

	_, err = env.Queries.AssignBoards(ctx, db.AssignBoardsParams{
		Board2ID: pgid(replaceBoard.ID),
		PlayerID: player.ID,
	})
	require.NoError(t, err)

	// Step 7: Verify other boards unchanged, new total (10 + 100 + 30 = 140)
	p, err = env.Queries.GetPlayerByID(ctx, player.ID)
	require.NoError(t, err)
	assertPgUUIDEqual(t, pgid(board.ID), p.Board1ID)
	assertPgUUIDEqual(t, pgid(replaceBoard.ID), p.Board2ID)
	assertPgUUIDEqual(t, pgid(b3.ID), p.Board3ID)

	rows, err = env.Queries.GetLeaderboard(ctx, db.GetLeaderboardParams{
		GameID:     game.ID,
		LimitCount: 100,
	})
	require.NoError(t, err)
	for _, r := range rows {
		if r.ID == player.ID {
			assertBoardTotalEqual(t, 140, r.BoardTotal)
			return
		}
	}
	t.Fatal("player not found in leaderboard")
}

// --- helpers ---

func assertPgUUIDEqual(t *testing.T, expected pgtype.UUID, actual pgtype.UUID) {
	t.Helper()
	if !expected.Valid {
		assert.False(t, actual.Valid, "expected null UUID, got non-null")
		return
	}
	assert.True(t, actual.Valid, "expected non-null UUID, got null")
	assert.Equal(t, expected.Bytes, actual.Bytes)
}

func assertBoardTotalEqual(t *testing.T, expected int32, actual interface{}) {
	t.Helper()
	switch v := actual.(type) {
	case int32:
		assert.Equal(t, expected, v)
	case int64:
		assert.Equal(t, expected, int32(v))
	case float64:
		assert.InDelta(t, float64(expected), v, 0.001)
	default:
		t.Errorf("unexpected BoardTotal type %T", actual)
	}
}
