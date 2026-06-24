package game_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"july/internal/db"
	"july/internal/testutil"

	"github.com/google/uuid"
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

func must200(t *testing.T, resp *http.Response) *http.Response {
	t.Helper()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	return resp
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

	t.Run("excludes private projects from leaderboard", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		_, projectPublic, game, _ := testutil.CreateGameScenario(t, env)
		_, projectPrivate := testutil.CreateGameScenarioForUser(t, env, game, "privuser", "Private User")

		err := env.Queries.SetProjectIsPrivate(context.Background(), db.SetProjectIsPrivateParams{
			ID:        projectPrivate.ID,
			IsPrivate: true,
		})
		require.NoError(t, err)

		resp := testutil.GetJSON(t, env, "/leaders/projects")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body := testutil.DecodeBody(t, resp)
		assert.Contains(t, body, projectPublic.Name)
		assert.Contains(t, body, projectPublic.Slug)
		assert.NotContains(t, body, projectPrivate.Name)
		assert.NotContains(t, body, projectPrivate.Slug)
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

func TestLeaderboardWithBoardAssignments(t *testing.T) {
	// This test verifies that the user leaderboard correctly reflects the
	// sum of points from all 3 assigned project boards — not just the
	// initial unverified score stored in the players table.
	t.Run("leaderboard scores reflect assigned project boards", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)

		// Create a game
		game := testutil.CreateActiveGame(t, env)
		ctx := context.Background()
		sql := env.Queries

		// ── Player A: single board = 10 pts ──────────────────
		userA := testutil.CreateUser(t, env, "agamer", "A Player")
		testutil.CreateUserIdentifier(t, env, userA.ID, "email", "agamer@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, userA.ID, "github", "10001", true, false)
		projA := testutil.CreateProject(t, env, "agamer-proj", "https://github.com/agamer/proj")
		boardA, err := sql.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			ProjectID:   projA.ID,
			Points:      10,
			CommitCount: 10,
		})
		require.NoError(t, err)
		_, err = sql.UpsertPlayer(ctx, db.UpsertPlayerParams{
			ID:             db.NewID(),
			GameID:         game.ID,
			UserID:         userA.ID,
			Points:         10,
			CommitCount:    10,
			ProjectCount:   1,
			AnalysisStatus: "pending",
		})
		require.NoError(t, err)
		aPlayerID := getPlayerID(t, env, userA.ID, game.ID)

		// ── Player B: two boards = 25 + 30 = 55 pts ───────────
		userB := testutil.CreateUser(t, env, "bgamer", "B Player")
		testutil.CreateUserIdentifier(t, env, userB.ID, "email", "bgamer@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, userB.ID, "github", "10002", true, false)
		projB1 := testutil.CreateProject(t, env, "bgamer-b1", "https://github.com/bgamer/b1")
		boardB1, err := sql.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			ProjectID:   projB1.ID,
			Points:      25,
			CommitCount: 25,
		})
		require.NoError(t, err)
		projB2 := testutil.CreateProject(t, env, "bgamer-b2", "https://github.com/bgamer/b2")
		boardB2, err := sql.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			ProjectID:   projB2.ID,
			Points:      30,
			CommitCount: 30,
		})
		require.NoError(t, err)
		_, err = sql.UpsertPlayer(ctx, db.UpsertPlayerParams{
			ID:             db.NewID(),
			GameID:         game.ID,
			UserID:         userB.ID,
			Points:         25,
			CommitCount:    25,
			ProjectCount:   1,
			AnalysisStatus: "pending",
		})
		require.NoError(t, err)
		bPlayerID := getPlayerID(t, env, userB.ID, game.ID)

		// ── Player C: three boards = 5 + 15 + 40 = 60 pts ─────
		userC := testutil.CreateUser(t, env, "cgamer", "C Player")
		testutil.CreateUserIdentifier(t, env, userC.ID, "email", "cgamer@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, userC.ID, "github", "10003", true, false)
		projC1 := testutil.CreateProject(t, env, "cgamer-c1", "https://github.com/cgamer/c1")
		boardC1, err := sql.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			ProjectID:   projC1.ID,
			Points:      5,
			CommitCount: 5,
		})
		require.NoError(t, err)
		projC2 := testutil.CreateProject(t, env, "cgamer-c2", "https://github.com/cgamer/c2")
		boardC2, err := sql.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			ProjectID:   projC2.ID,
			Points:      15,
			CommitCount: 15,
		})
		require.NoError(t, err)
		projC3 := testutil.CreateProject(t, env, "cgamer-c3", "https://github.com/cgamer/c3")
		boardC3, err := sql.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			ProjectID:   projC3.ID,
			Points:      40,
			CommitCount: 40,
		})
		require.NoError(t, err)
		_, err = sql.UpsertPlayer(ctx, db.UpsertPlayerParams{
			ID:             db.NewID(),
			GameID:         game.ID,
			UserID:         userC.ID,
			Points:         5,
			CommitCount:    5,
			ProjectCount:   1,
			AnalysisStatus: "pending",
		})
		require.NoError(t, err)
		cPlayerID := getPlayerID(t, env, userC.ID, game.ID)

		// Now assign all boards to each player
		_, err = sql.AssignBoards(ctx, db.AssignBoardsParams{
			PlayerID: aPlayerID,
			Board1ID: db.UUID(boardA.ID),
		})
		require.NoError(t, err)

		_, err = sql.AssignBoards(ctx, db.AssignBoardsParams{
			PlayerID: bPlayerID,
			Board1ID: db.UUID(boardB1.ID),
			Board2ID: db.UUID(boardB2.ID),
		})
		require.NoError(t, err)

		_, err = sql.AssignBoards(ctx, db.AssignBoardsParams{
			PlayerID: cPlayerID,
			Board1ID: db.UUID(boardC1.ID),
			Board2ID: db.UUID(boardC2.ID),
			Board3ID: db.UUID(boardC3.ID),
		})
		require.NoError(t, err)

		// Trigger analysis to set verified_points on each player.
		aTotal, _ := sql.GetPlayerBoardTotal(ctx, db.GetPlayerBoardTotalParams{
			Board1ID: boardA.ID,
		})
		_ = sql.UpdatePlayerAnalysis(ctx, db.UpdatePlayerAnalysisParams{
			ID:             aPlayerID,
			VerifiedPoints: aTotal,
			AnalysisStatus: "completed",
		})

		bTotal, _ := sql.GetPlayerBoardTotal(ctx, db.GetPlayerBoardTotalParams{
			Board1ID: boardB1.ID,
			Board2ID: boardB2.ID,
		})
		_ = sql.UpdatePlayerAnalysis(ctx, db.UpdatePlayerAnalysisParams{
			ID:             bPlayerID,
			VerifiedPoints: bTotal,
			AnalysisStatus: "completed",
		})

		cTotal, _ := sql.GetPlayerBoardTotal(ctx, db.GetPlayerBoardTotalParams{
			Board1ID: boardC1.ID,
			Board2ID: boardC2.ID,
			Board3ID: boardC3.ID,
		})
		_ = sql.UpdatePlayerAnalysis(ctx, db.UpdatePlayerAnalysisParams{
			ID:             cPlayerID,
			VerifiedPoints: cTotal,
			AnalysisStatus: "completed",
		})

		// Fetch the leaderboard
		resp := testutil.GetJSON(t, env, "/leaders")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body := testutil.DecodeBody(t, resp)

		// Player C should be rank 1 (60 pts), B rank 2 (55 pts), A rank 3 (10 pts)
		cPos := strings.Index(body, "cgamer")
		bPos := strings.Index(body, "bgamer")
		aPos := strings.Index(body, "agamer")

		assert.True(t, cPos < bPos, "Player C (60 pts) should rank above Player B (55 pts); C pos=%d, B pos=%d", cPos, bPos)
		assert.True(t, bPos < aPos, "Player B (55 pts) should rank above Player A (10 pts); B pos=%d, A pos=%d", bPos, aPos)

		// Verify the correct scores appear in the leaderboard (each player's board total).
		// The Points column renders as: <span class="font-bold text-july-400">{score}</span>
		assert.Contains(t, body, `text-july-400">60</span>`, "leaderboard should show 60 pts for Player C (5+15+40)")
		assert.Contains(t, body, `text-july-400">55</span>`, "leaderboard should show 55 pts for Player B (25+30)")
		assert.Contains(t, body, `text-july-400">10</span>`, "leaderboard should show 10 pts for Player A")
	})
}

// getPlayerID returns the player row for the given user + game.
func getPlayerID(t *testing.T, env *testutil.TestEnv, userID, gameID uuid.UUID) uuid.UUID {
	ctx := context.Background()
	player, err := env.Queries.GetPlayerByUserAndGame(ctx, db.GetPlayerByUserAndGameParams{
		UserID: userID,
		GameID: gameID,
	})
	require.NoError(t, err)
	return player.ID
}
