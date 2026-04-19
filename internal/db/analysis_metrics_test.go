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

	upsert := func(metricType string, score int16) db.AnalysisMetric {
		t.Helper()
		require.NoError(t, env.Queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID:         db.NewID(),
			ProjectID:  project.ID,
			MetricType: metricType,
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

	elevate := func(metricType string, level int16) {
		t.Helper()
		require.NoError(t, env.Queries.UpdateAnalysisMetricLevel(ctx, db.UpdateAnalysisMetricLevelParams{
			ProjectID:  project.ID,
			MetricType: metricType,
			Level:      level,
			UpdatedBy:  user.ID,
		}))
	}

	t.Run("partial score promotes to heuristic L1", func(t *testing.T) {
		m := upsert("readme", 7)
		assert.Equal(t, int16(1), m.Level)
		assert.Equal(t, int16(7), m.Score)
	})

	t.Run("full score promotes to L1", func(t *testing.T) {
		m := upsert("readme", 10)
		assert.Equal(t, int16(1), m.Level)
		assert.Equal(t, int16(10), m.Score)
	})

	t.Run("rescan below 10 stays heuristic L1", func(t *testing.T) {
		m := upsert("readme", 8)
		assert.Equal(t, int16(1), m.Level)
		assert.Equal(t, int16(8), m.Score)
	})

	t.Run("L2 is preserved when rescan comes back full", func(t *testing.T) {
		// Restore to L1 first, then elevate to L2.
		upsert("readme", 10)
		elevate("readme", 2)

		m := upsert("readme", 10)
		assert.Equal(t, int16(2), m.Level, "L2 should survive a clean rescan")
		assert.Equal(t, int16(10), m.Score)
	})

	t.Run("L2 is preserved even when score drops below 10", func(t *testing.T) {
		m := upsert("readme", 7)
		assert.Equal(t, int16(2), m.Level, "L2 is never downgraded by a rescan")
		assert.Equal(t, int16(7), m.Score)
	})

	t.Run("L3 is preserved when rescan comes back full", func(t *testing.T) {
		upsert("readme", 10)
		elevate("readme", 3)

		m := upsert("readme", 10)
		assert.Equal(t, int16(3), m.Level, "L3 should survive a clean rescan")
	})

	t.Run("data and sha update on every upsert", func(t *testing.T) {
		require.NoError(t, env.Queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID:         db.NewID(),
			ProjectID:  project.ID,
			MetricType: "readme",
			Score:      10,
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
		assert.Equal(t, int16(1), ci.Level)

		readme, err := env.Queries.GetAnalysisMetric(ctx, db.GetAnalysisMetricParams{
			ProjectID: project.ID, MetricType: "readme",
		})
		require.NoError(t, err)
		assert.Equal(t, int16(10), readme.Score, "readme unaffected by ci upsert")
	})
}

func TestGetProjectTotalScore(t *testing.T) {
	ctx := context.Background()
	env := testutil.SetupTestEnv(t)

	user := testutil.CreateUser(t, env, "testuser", "Test User")
	project := testutil.CreateProject(t, env, "test-project", "https://github.com/testuser/test-project")

	upsert := func(metricType string, score int16) {
		t.Helper()
		require.NoError(t, env.Queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID:         db.NewID(),
			ProjectID:  project.ID,
			MetricType: metricType,
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

	t.Run("partial scores contribute at heuristic L1", func(t *testing.T) {
		upsert("readme", 7) // L1 — score * level = 7 * 1 = 7
		total, err := env.Queries.GetProjectTotalScore(ctx, project.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(7), total)
	})

	t.Run("full score at L1 contributes 10 points", func(t *testing.T) {
		upsert("readme", 10) // promotes to L1 — 10 * 1 = 10
		total, err := env.Queries.GetProjectTotalScore(ctx, project.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(10), total)
	})

	t.Run("L2 doubles the score", func(t *testing.T) {
		require.NoError(t, env.Queries.UpdateAnalysisMetricLevel(ctx, db.UpdateAnalysisMetricLevelParams{
			ProjectID:  project.ID,
			MetricType: "readme",
			Level:      2,
			UpdatedBy:  user.ID,
		}))
		total, err := env.Queries.GetProjectTotalScore(ctx, project.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(20), total) // 10 * 2
	})

	t.Run("L3 triples the score", func(t *testing.T) {
		require.NoError(t, env.Queries.UpdateAnalysisMetricLevel(ctx, db.UpdateAnalysisMetricLevelParams{
			ProjectID:  project.ID,
			MetricType: "readme",
			Level:      3,
			UpdatedBy:  user.ID,
		}))
		total, err := env.Queries.GetProjectTotalScore(ctx, project.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(30), total) // 10 * 3
	})

	t.Run("sums across multiple metrics", func(t *testing.T) {
		upsert("tests", 10) // L1 = 10
		upsert("ci", 10)    // L1 = 10
		total, err := env.Queries.GetProjectTotalScore(ctx, project.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(50), total) // readme(30) + tests(10) + ci(10)
	})
}
