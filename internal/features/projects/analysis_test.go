package projects_test

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
// Readme has 6 bool fields (including ReadmeHasCodeBlocks);
// 5 true (full) → score 8, 1 true (partial) → score 1.

var (
	readmeFull = map[string]any{
		"has_readme":         true,
		"readme_substantial": true,
		"readme_has_install": true,
		"readme_has_usage":   true,
		"readme_has_banners": true,
	}
	readmePartial = map[string]any{
		"has_readme": true, // 1/6 → score 1
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

func postAnalysis(t *testing.T, env *testutil.TestEnv, projectID, metricType, sha string, data map[string]any) *http.Response {
	t.Helper()
	return testutil.PostJSON(t, env, analysisPath(projectID), map[string]any{
		"sha":        sha,
		"metricType": metricType,
		"data":       data,
	})
}

// ── Tests ─────────────────────────────────────────────────────────

func TestPostProjectAnalysis(t *testing.T) {
	t.Run("unauthenticated request is rejected", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		project := testutil.CreateProject(t, env, "gh-nobody-repo", "https://github.com/nobody/repo")

		resp := postAnalysis(t, env, project.ID.String(), "readme", "abc123", readmePartial)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("unknown project returns 404", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := postAnalysis(t, env, "00000000-0000-0000-0000-000000000000", "readme", "abc123", readmePartial)
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
		resp := postAnalysis(t, env, aliceProject.ID.String(), "readme", "abc123", readmePartial)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("partial readme payload maps to L1 (score 1–5)", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := postAnalysis(t, env, project.ID.String(), "readme", "abc123", readmePartial)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body := decodeAnalysis(t, resp)
		assert.Equal(t, "readme", body.MetricType)
		assert.Equal(t, int16(1), body.Score) // 1 true / 6 bools * 10 = 1
		assert.Equal(t, int16(1), body.Level)  // score 1 → level 1 (1–5 range)
	})

	t.Run("full readme payload maps to L2 (score 6–8)", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := postAnalysis(t, env, project.ID.String(), "readme", "abc123", readmeFull)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body := decodeAnalysis(t, resp)
		assert.Equal(t, int16(8), body.Score)  // 5 true / 6 bools * 10 = 8
		assert.Equal(t, int16(2), body.Level)  // score 8 → level 2 (6–8 range)
	})

	t.Run("score 9–10 maps to L3", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		// CI metric: 4 true / 4 bools → score 10
		resp := postAnalysis(t, env, project.ID.String(), "ci", "abc123", ciFull)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body := decodeAnalysis(t, resp)
		assert.Equal(t, "ci", body.MetricType)
		assert.Equal(t, int16(10), body.Score)
		assert.Equal(t, int16(3), body.Level) // score 10 → level 3 (9–10 range)
	})

	t.Run("score 0 maps to level 0", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := postAnalysis(t, env, project.ID.String(), "readme", "abc123", map[string]any{})
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body := decodeAnalysis(t, resp)
		assert.Equal(t, int16(0), body.Score)
		assert.Equal(t, int16(0), body.Level)
	})

	t.Run("score changes on rescan (no preservation)", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		// First: L2 (score 8)
		resp := must200(t, postAnalysis(t, env, project.ID.String(), "readme", "abc", readmeFull))
		body := decodeAnalysis(t, resp)
		assert.Equal(t, int16(8), body.Score)
		assert.Equal(t, int16(2), body.Level)

		// Rescan: partial (score 1)
		resp = postAnalysis(t, env, project.ID.String(), "readme", "rescan", readmePartial)
		body = decodeAnalysis(t, resp)
		assert.Equal(t, int16(1), body.Score)
		assert.Equal(t, int16(1), body.Level) // re-scoring to L1 (no preservation)
	})

	t.Run("different metric types are scored independently", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		readmeResp := decodeAnalysis(t, must200(t, postAnalysis(t, env, project.ID.String(), "readme", "abc", readmeFull)))
		ciResp := decodeAnalysis(t, must200(t, postAnalysis(t, env, project.ID.String(), "ci", "abc", ciFull)))

		assert.Equal(t, "readme", readmeResp.MetricType)
		assert.Equal(t, "ci", ciResp.MetricType)
		assert.Equal(t, int16(8), readmeResp.Score) // 5/6 = 8, L2
		assert.Equal(t, int16(10), ciResp.Score)    // 4/4 = 10, L3
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
			{"missing sha", map[string]any{"metricType": "readme", "data": readmePartial}},
			{"missing metricType", map[string]any{"sha": "abc", "data": readmePartial}},
			{"missing data", map[string]any{"sha": "abc", "metricType": "readme"}},
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

func llmContextPath(projectID, metric string) string {
	return "/api/projects/" + projectID + "/analysis/metrics/" + metric + "/llm-context"
}

func TestGetProjectMetricLLMContext(t *testing.T) {
	t.Run("unauthenticated request succeeds", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		project := testutil.CreateProject(t, env, "gh-nobody-repo", "https://github.com/nobody/repo")
		ctx := context.Background()
		require.NoError(t, env.Queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID:         db.NewID(),
			ProjectID:  project.ID,
			MetricType: "readme",
			Level:      1,
			Score:      5,
			Data:       db.JSONMap{"has_readme": true},
			Sha:        "def456",
			UpdatedBy:  db.SystemUserID,
		}))
		resp := testutil.GetJSON(t, env, llmContextPath(project.ID.String(), "readme"))
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("returns JSON for owner with L1 metric", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		ctx := context.Background()
		require.NoError(t, env.Queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID:         db.NewID(),
			ProjectID:  project.ID,
			MetricType: "readme",
			Level:      1,
			Score:      6,
			Data:       db.JSONMap{"has_readme": true},
			Sha:        "abc123",
			UpdatedBy:  user.ID,
		}))

		resp := testutil.GetJSON(t, env, llmContextPath(project.ID.String(), "readme"))
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		resp.Body.Close()
		assert.Equal(t, "readme", body["metricType"])
		assert.NotEmpty(t, body["userPrompt"])
		assert.NotEmpty(t, body["systemPrompt"])
		assert.Equal(t, float64(6), body["l1Score"])
	})

	t.Run("rejects zero score metric with JSON body", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		ctx := context.Background()
		require.NoError(t, env.Queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID:         db.NewID(),
			ProjectID:  project.ID,
			MetricType: "readme",
			Level:      0,
			Score:      0,
			Data:       db.JSONMap{"has_readme": false},
			Sha:        "abc123",
			UpdatedBy:  user.ID,
		}))

		resp := testutil.GetJSON(t, env, llmContextPath(project.ID.String(), "readme"))
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		resp.Body.Close()
		assert.Equal(t, "readme", body["metricType"])
		assert.NotEmpty(t, body["userPrompt"])
		assert.NotEmpty(t, body["systemPrompt"])
		assert.Equal(t, float64(0), body["l1Score"])
	})

	t.Run("succeeds with missing metric row by falling back", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := testutil.GetJSON(t, env, llmContextPath(project.ID.String(), "readme"))
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		resp.Body.Close()
		assert.Equal(t, "readme", body["metricType"])
		assert.NotEmpty(t, body["userPrompt"])
		assert.NotEmpty(t, body["systemPrompt"])

		// Verify it contains the README-specific checkmarks, not all metrics.
		bodyStr := body["userPrompt"].(string)
		assert.Contains(t, bodyStr, "missing: Has README")
		assert.NotContains(t, bodyStr, "has_test_dir")
		assert.NotContains(t, bodyStr, "has_ci")
	})
}
