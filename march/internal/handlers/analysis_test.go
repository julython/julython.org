package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"july/internal/db"
	"july/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Payload fixtures ─────────────────────────────────────────────
// Readme has 5 bool fields; all true → score 10, one true → score 2.

var (
	readmeFull = map[string]any{
		"has_readme":         true,
		"readme_substantial": true,
		"readme_has_install": true,
		"readme_has_usage":   true,
		"readme_has_banners": true,
	}
	readmePartial = map[string]any{
		"has_readme": true, // 1/5 → score 2
	}

	// CI has 4 bool fields; all true → score 10.
	ciFull = map[string]any{
		"has_ci":         true,
		"has_lint_step":  true,
		"has_test_step":  true,
		"has_build_step": true,
	}
)

// ── Helpers ───────────────────────────────────────────────────────

type analysisResponse struct {
	MetricType string `json:"metricType"`
	Score      int16  `json:"score"`
	Level      int16  `json:"level"`
}

func decodeAnalysis(t *testing.T, resp *http.Response) analysisResponse {
	t.Helper()
	var out analysisResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return out
}

func analysisPath(projectID string) string {
	return "/api/projects/" + projectID + "/analysis"
}

func postAnalysis(t *testing.T, env *testutil.TestEnv, projectID, metricType, sha string, level int, data map[string]any) *http.Response {
	t.Helper()
	return testutil.PostJSON(t, env, analysisPath(projectID), map[string]any{
		"sha":        sha,
		"metricType": metricType,
		"level":      level,
		"data":       data,
	})
}

// ── Tests ─────────────────────────────────────────────────────────

func TestPostProjectAnalysis(t *testing.T) {
	t.Run("unauthenticated request is rejected", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		project := testutil.CreateProject(t, env, "gh-nobody-repo", "https://github.com/nobody/repo")

		resp := postAnalysis(t, env, project.ID.String(), "readme", "abc123", 0, readmePartial)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("unknown project returns 404", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := postAnalysis(t, env, "00000000-0000-0000-0000-000000000000", "readme", "abc123", 0, readmePartial)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("non-owner is forbidden", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		testutil.CreateProject(t, env, "gh-alice-repo", "https://github.com/alice/repo")
		bob := testutil.CreateUser(t, env, "bob", "Bob")
		testutil.CreateUserIdentifier(t, env, bob.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		// alice's project; bob must be rejected
		aliceProject := testutil.CreateProject(t, env, "gh-alice-other", "https://github.com/alice/other")
		resp := postAnalysis(t, env, aliceProject.ID.String(), "readme", "abc123", 0, readmePartial)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("partial readme payload returns score 2 at heuristic L1", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := postAnalysis(t, env, project.ID.String(), "readme", "abc123", 0, readmePartial)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body := decodeAnalysis(t, resp)
		assert.Equal(t, "readme", body.MetricType)
		assert.Equal(t, int16(2), body.Score) // 1 true / 5 bools * 10 = 2
		assert.Equal(t, int16(1), body.Level)
	})

	t.Run("full readme payload returns score 10 at L1", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := postAnalysis(t, env, project.ID.String(), "readme", "abc123", 1, readmeFull)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body := decodeAnalysis(t, resp)
		assert.Equal(t, int16(10), body.Score)
		assert.Equal(t, int16(1), body.Level)
	})

	t.Run("different metric types are scored independently", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		readmeResp := decodeAnalysis(t, must200(t, postAnalysis(t, env, project.ID.String(), "readme", "abc", 0, readmeFull)))
		ciResp := decodeAnalysis(t, must200(t, postAnalysis(t, env, project.ID.String(), "ci", "abc", 0, ciFull)))

		assert.Equal(t, "readme", readmeResp.MetricType)
		assert.Equal(t, "ci", ciResp.MetricType)
		assert.Equal(t, int16(10), readmeResp.Score)
		assert.Equal(t, int16(10), ciResp.Score)
	})

	t.Run("L2 upgrade rejected if metric not yet at L1", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		// No prior metric — handler must reject with 400.
		resp := postAnalysis(t, env, project.ID.String(), "readme", "abc123", 2, readmeFull)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("L2 upgrade succeeds after L1 is established", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		// Establish L1 via the route.
		must200(t, postAnalysis(t, env, project.ID.String(), "readme", "abc123", 1, readmeFull))

		// AI grades it to L2.
		resp := postAnalysis(t, env, project.ID.String(), "readme", "abc123", 2, readmeFull)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body := decodeAnalysis(t, resp)
		assert.Equal(t, int16(2), body.Level)
		assert.Equal(t, int16(10), body.Score)
	})

	t.Run("L2 upgrade succeeds after partial heuristic L1", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		must200(t, postAnalysis(t, env, project.ID.String(), "readme", "abc123", 0, readmePartial))

		resp := postAnalysis(t, env, project.ID.String(), "readme", "abc123", 2, readmeFull)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body := decodeAnalysis(t, resp)
		assert.Equal(t, int16(2), body.Level)
		assert.Equal(t, int16(2), body.Score)
	})

	t.Run("rescan does not downgrade an L2 metric", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		// L1 → L2 via the route.
		must200(t, postAnalysis(t, env, project.ID.String(), "readme", "abc123", 1, readmeFull))
		must200(t, postAnalysis(t, env, project.ID.String(), "readme", "abc123", 2, readmeFull))

		// Rescan with a poor payload — L2 must survive.
		resp := postAnalysis(t, env, project.ID.String(), "readme", "rescan", 0, readmePartial)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body := decodeAnalysis(t, resp)
		assert.Equal(t, int16(2), body.Level, "L2 must survive a bad rescan")
		assert.Equal(t, int16(2), body.Score, "score updates to reflect the latest scan")
	})

	t.Run("L2 is preserved even when score drops to 0", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		ctx := context.Background()
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		// Seed L2 directly for this edge-case test.
		require.NoError(t, env.Queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID: db.NewID(), ProjectID: project.ID, MetricType: "readme",
			Score: 10, Data: db.JSONMap{}, Sha: "seed", UpdatedBy: user.ID,
		}))
		require.NoError(t, env.Queries.UpdateAnalysisMetricLevel(ctx, db.UpdateAnalysisMetricLevelParams{
			ProjectID: project.ID, MetricType: "readme", Level: 2, UpdatedBy: user.ID,
		}))

		// All-false payload → score 0.
		resp := postAnalysis(t, env, project.ID.String(), "readme", "rescan", 0, map[string]any{})
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body := decodeAnalysis(t, resp)
		assert.Equal(t, int16(2), body.Level, "L2 must not be downgraded by score=0")
	})

	t.Run("invalid payload fields return 400", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		cases := []struct {
			name    string
			payload map[string]any
		}{
			{"missing sha", map[string]any{"metricType": "readme", "level": 0, "data": readmePartial}},
			{"missing metricType", map[string]any{"sha": "abc", "level": 0, "data": readmePartial}},
			{"missing data", map[string]any{"sha": "abc", "metricType": "readme", "level": 0}},
			{"unknown metricType", map[string]any{"sha": "abc", "metricType": "nope", "level": 0, "data": readmePartial}},
			{"level out of range", map[string]any{"sha": "abc", "metricType": "readme", "level": 4, "data": readmePartial}},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				resp := testutil.PostJSON(t, env, analysisPath(project.ID.String()), tc.payload)
				assert.Equal(t, http.StatusBadRequest, resp.StatusCode, tc.name)
			})
		}
	})
}

func must200(t *testing.T, resp *http.Response) *http.Response {
	t.Helper()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	return resp
}
