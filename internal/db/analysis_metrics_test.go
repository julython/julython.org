package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"july/internal/db"
	"july/internal/testutil"
)

func TestUpsertAnalysisMetric(t *testing.T) {
	ctx := context.Background()
	env := testutil.SetupTestEnv(t)

	user := testutil.CreateUser(t, env, "testuser", "Test User")
	project := testutil.CreateProject(t, env, "test-project", "https://github.com/testuser/test-project")

	// upsert helper: maps score to level — 0→0, 1–5→1, 6–8→2, 9–10→3.
	upsert := func(metricType string, score int16) db.AnalysisMetric {
		t.Helper()
		level := int16(0)
		switch {
		case score >= 9:
			level = 3
		case score >= 6:
			level = 2
		case score >= 1:
			level = 1
		}
		require.NoError(t, env.Queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID:         db.NewID(),
			ProjectID:  project.ID,
			MetricType: metricType,
			Level:      level,
			Score:      score,
			Data:       db.JSONMap{"score": score},
			Sha:        "abc123",
			UpdatedBy:  user.ID,
		}))
		m, err := env.Queries.GetAnalysisMetric(ctx, db.GetAnalysisMetricParams{
			ProjectID:  project.ID,
			MetricType: metricType,
		})
		require.NoError(t, err)
		return m
	}

	t.Run("score 0 maps to level 0", func(t *testing.T) {
		m := upsert("readme", 0)
		assert.Equal(t, int16(0), m.Level)
		assert.Equal(t, int16(0), m.Score)
	})

	t.Run("score 1–5 maps to level 1", func(t *testing.T) {
		for _, sc := range []int16{1, 3, 5} {
			m := upsert("readme", sc)
			assert.Equal(t, int16(1), m.Level)
		}
	})

	t.Run("score 6–8 maps to level 2", func(t *testing.T) {
		for _, sc := range []int16{6, 7, 8} {
			m := upsert("readme", sc)
			assert.Equal(t, int16(2), m.Level)
		}
	})

	t.Run("score 9–10 maps to level 3", func(t *testing.T) {
		for _, sc := range []int16{9, 10} {
			m := upsert("readme", sc)
			assert.Equal(t, int16(3), m.Level)
		}
	})

	t.Run("rescan updates both score and level", func(t *testing.T) {
		m := upsert("readme", 5)
		assert.Equal(t, int16(1), m.Level)

		// Rescan to score 7 → should now be L2.
		m = upsert("readme", 7)
		assert.Equal(t, int16(2), m.Level)
		assert.Equal(t, int16(7), m.Score)
	})

	t.Run("data and sha update on every upsert", func(t *testing.T) {
		require.NoError(t, env.Queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID:         db.NewID(),
			ProjectID:  project.ID,
			MetricType: "readme",
			Level:      1,
			Score:      5,
			Data:       db.JSONMap{"hasReadme": true, "sizeBytes": 1200},
			Sha:        "def456",
			UpdatedBy:  user.ID,
		}))
		m, err := env.Queries.GetAnalysisMetric(ctx, db.GetAnalysisMetricParams{
			ProjectID: project.ID, MetricType: "readme",
		})
		require.NoError(t, err)
		assert.Equal(t, "def456", m.Sha)
	})

	t.Run("separate metric types are independent", func(t *testing.T) {
		ci := upsert("ci", 10)
		assert.Equal(t, int16(3), ci.Level)

		readme, err := env.Queries.GetAnalysisMetric(ctx, db.GetAnalysisMetricParams{
			ProjectID: project.ID, MetricType: "readme",
		})
		require.NoError(t, err)
		assert.Equal(t, int16(5), readme.Score, "readme unaffected by ci upsert")
	})
}

func TestGetProjectTotalScore(t *testing.T) {
	ctx := context.Background()
	env := testutil.SetupTestEnv(t)

	user := testutil.CreateUser(t, env, "testuser", "Test User")
	project := testutil.CreateProject(t, env, "test-project", "https://github.com/testuser/test-project")

	// upsert helper: maps score to level — 0→0, 1–5→1, 6–8→2, 9–10→3.
	upsert := func(metricType string, score int16) {
		t.Helper()
		level := int16(0)
		switch {
		case score >= 9:
			level = 3
		case score >= 6:
			level = 2
		case score >= 1:
			level = 1
		}
		require.NoError(t, env.Queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID:         db.NewID(),
			ProjectID:  project.ID,
			MetricType: metricType,
			Level:      level,
			Score:      score,
			Data:       db.JSONMap{},
			Sha:        "abc123",
			UpdatedBy:  user.ID,
		}))
	}

	t.Run("zero with no metrics", func(t *testing.T) {
		total, err := env.Queries.GetProjectTotalScore(ctx, project.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(0), total)
	})

	t.Run("score 0 contributes nothing", func(t *testing.T) {
		upsert("readme", 0) // level 0 → 0 * 0 = 0
		total, err := env.Queries.GetProjectTotalScore(ctx, project.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(0), total)
	})

	t.Run("score 1–5 (L1) contributes raw score", func(t *testing.T) {
		upsert("readme", 5) // L1 = 5
		total, err := env.Queries.GetProjectTotalScore(ctx, project.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(5), total) // 5 * 1
	})

	t.Run("score 6–8 (L2) doubles contribution", func(t *testing.T) {
		upsert("readme", 8) // L2 = 8 → 8 * 2 = 16
		total, err := env.Queries.GetProjectTotalScore(ctx, project.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(16), total)
	})

	t.Run("score 9–10 (L3) triples contribution", func(t *testing.T) {
		upsert("readme", 10) // L3 = 10 → 10 * 3 = 30
		total, err := env.Queries.GetProjectTotalScore(ctx, project.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(30), total)
	})

	t.Run("sums across multiple metrics", func(t *testing.T) {
		upsert("readme", 8)  // L2 = 16
		upsert("tests", 10)  // L3 = 30
		upsert("ci", 3)      // L1 = 3
		total, err := env.Queries.GetProjectTotalScore(ctx, project.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(49), total) // 16 + 30 + 3
	})
}
