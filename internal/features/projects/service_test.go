package projects_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"july/internal/components/analysis"
	"july/internal/db"
	"july/internal/features/projects"
	"july/internal/testutil"
)

func TestBuildAnalysisBoard(t *testing.T) {
	t.Run("empty project returns tiles with zeros", func(t *testing.T) {
		ctx := context.Background()
		env := testutil.SetupTestEnv(t)
		project := testutil.CreateProject(t, env, "empty-project", "https://github.com/test/empty")

		service := projects.NewProjectService(env.Queries)
		board, err := service.BuildAnalysisBoard(ctx, project.ID)
		require.NoError(t, err)
		// BuildAnalysisBoard always returns 8 tiles (one per metric),
		// even when no analysis data exists — scores and levels default to 0.
		assert.Equal(t, 8, len(board.Tiles))
		assert.Equal(t, 0, board.EarnedPts)
		assert.Equal(t, 480, board.MaxPts)
		assert.Equal(t, 0, board.AnalysisRunCount)
	})

	t.Run("board with analysis metrics shows tiles and scores", func(t *testing.T) {
		ctx := context.Background()
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		project := testutil.CreateProject(t, env, "scored-project", "https://github.com/test/scored")

		// Insert analysis metrics with an updated_by to satisfy FK
		require.NoError(t, env.Queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID:         db.NewID(),
			ProjectID:  project.ID,
			MetricType: "readme",
			Score:      8,
			Sha:        "abc123",
			UpdatedBy:  user.ID,
		}))
		require.NoError(t, env.Queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID:         db.NewID(),
			ProjectID:  project.ID,
			MetricType: "tests",
			Score:      5,
			Sha:        "def456",
			UpdatedBy:  user.ID,
		}))

		service := projects.NewProjectService(env.Queries)
		board, err := service.BuildAnalysisBoard(ctx, project.ID)
		require.NoError(t, err)

		// Find readme tile
		var readmeTile *analysis.AnalysisTile
		var testsTile *analysis.AnalysisTile
		for i := range board.Tiles {
			switch board.Tiles[i].MetricKey {
			case "readme":
				readmeTile = &board.Tiles[i]
			case "tests":
				testsTile = &board.Tiles[i]
			}
		}
		require.NotNil(t, readmeTile)
		require.NotNil(t, testsTile)
		assert.Equal(t, int16(8), readmeTile.Score)
		assert.Equal(t, int16(5), testsTile.Score)

		// Earned points = score * level * 2 for each metric.
		// The DB computes level 1 when score > 0.
		// readme: 8*1*2 = 16, tests: 5*1*2 = 10, total = 26
		assert.Equal(t, 26, board.EarnedPts)
		assert.Greater(t, board.AnalysisRunCount, 0)
	})
}

func TestGameActivitySummary(t *testing.T) {
	t.Run("nonexistent game returns default summary", func(t *testing.T) {
		ctx := context.Background()
		env := testutil.SetupTestEnv(t)
		project := testutil.CreateProject(t, env, "nogame-project", "https://github.com/test/nogame")

		service := projects.NewProjectService(env.Queries)
		result, err := service.GameActivitySummary(ctx, project.ID, uuid.Nil)
		require.NoError(t, err)
		assert.False(t, result.HasGame)
		assert.Nil(t, result.Board)
	})

	t.Run("active game returns activity with board stats", func(t *testing.T) {
		ctx := context.Background()
		env := testutil.SetupTestEnv(t)
		project := testutil.CreateProject(t, env, "game-project", "https://github.com/test/game")
		game := testutil.CreateActiveGame(t, env)

		_, err := env.Queries.UpsertBoard(ctx, db.UpsertBoardParams{
			ID:               db.NewID(),
			GameID:           game.ID,
			ProjectID:        project.ID,
			Points:           15,
			CommitCount:      5,
			ContributorCount: 3,
		})
		require.NoError(t, err)

		service := projects.NewProjectService(env.Queries)
		result, err := service.GameActivitySummary(ctx, project.ID, game.ID)
		require.NoError(t, err)
		assert.True(t, result.HasGame)
		assert.NotNil(t, result.Board)
		assert.Equal(t, 5, result.Board.CommitCount)
		assert.Equal(t, 3, result.Board.ContributorCount)
	})
}
