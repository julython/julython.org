package projects_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"july/internal/db"
	"july/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chatContextPath(projectID string) string {
	return "/api/projects/" + projectID + "/analysis/chat-context"
}

func postChatContext(t *testing.T, env *testutil.TestEnv, projectID, message string) *http.Response {
	t.Helper()
	return testutil.PostJSON(t, env, chatContextPath(projectID), map[string]any{
		"message": message,
	})
}

func TestPostProjectChatContext(t *testing.T) {
	t.Run("unauthenticated request succeeds", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		project := testutil.CreateProject(t, env, "gh-nobody-repo", "https://github.com/nobody/repo")

		resp := postChatContext(t, env, project.ID.String(), "Can you analyze my tests?")
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.NotEmpty(t, body["systemPrompt"])
		assert.NotEmpty(t, body["userPrompt"])
	})

	t.Run("unknown project returns 404", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := testutil.PostJSON(t, env, chatContextPath("00000000-0000-0000-0000-000000000000"), map[string]any{"message": "hello"})
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("non-owner chat succeeds", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		alice := testutil.CreateUser(t, env, "alice", "Alice")
		project := testutil.CreateOwnedProject(t, env, alice, "repo", "https://github.com/alice/repo")
		bob := testutil.CreateUser(t, env, "bob", "Bob")
		testutil.CreateUserIdentifier(t, env, bob.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := postChatContext(t, env, project.ID.String(), "Can you analyze my README?")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("non-GitHub project is rejected", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateAdminUser(t, env, "gitlabuser", "GitLab User")
		project := testutil.CreateOwnedProject(t, env, user, "project", "https://gitlab.com/gitlab-user/project")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "gitlab@example.com", true, true)
		env.LoginAs(t, "gitlab@example.com")

		// Change the project service to "gitlab" using the new SQLC query.
		ctx := context.Background()
		_, err := env.Queries.UpdateProjectService(ctx, db.UpdateProjectServiceParams{
			ID:      project.ID,
			Service: "gitlab",
		})
		require.NoError(t, err)

		resp := postChatContext(t, env, project.ID.String(), "Can you analyze my README?")
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty message returns 400", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := testutil.PostJSON(t, env, chatContextPath(project.ID.String()), map[string]any{"message": "  "})
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("response contains prompts when no L1 metric exists", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "chatowner", "Chat Owner")
		project := testutil.CreateOwnedProject(t, env, user, "chat-repo", "https://github.com/chatowner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := postChatContext(t, env, project.ID.String(), "Can you analyze my README?")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "readme", body["matchedMetric"])
		assert.Equal(t, "readme", body["contextMetric"])
		assert.Equal(t, false, body["usedDefaultReadme"])
		assert.NotEmpty(t, body["systemPrompt"])
		assert.NotEmpty(t, body["userPrompt"])
	})

	t.Run("keyword-matched metric returns matchedMetric field", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "keyworduser", "Keyword User")
		project := testutil.CreateOwnedProject(t, env, user, "keyword-repo", "https://github.com/keyworduser/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		// Establish a L1 ci metric for this project
		ctx := context.Background()
		require.NoError(t, env.Queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID:         db.NewID(),
			ProjectID:  project.ID,
			MetricType: "ci",
			Level:      1,
			Score:      7,
			Data:       db.JSONMap{"has_ci": true, "has_lint_step": true, "has_test_step": true, "has_build_step": true},
			Sha:        "abc123",
			UpdatedBy:  user.ID,
		}))

		resp := postChatContext(t, env, project.ID.String(), "Can you analyze my CI status?")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		// The keyword "CI" should be matched
		assert.Equal(t, "ci", body["matchedMetric"])
		assert.Equal(t, "ci", body["contextMetric"])
	})

	t.Run("response is correct when keyword metric exists with data", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "datauser", "Data User")
		project := testutil.CreateOwnedProject(t, env, user, "data-repo", "https://github.com/datauser/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		// Establish a L1 ci metric
		ctx := context.Background()
		require.NoError(t, env.Queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID:         db.NewID(),
			ProjectID:  project.ID,
			MetricType: "ci",
			Level:      1,
			Score:      7,
			Data:       db.JSONMap{"has_ci": true, "has_lint_step": true, "has_test_step": true, "has_build_step": true},
			Sha:        "abc123",
			UpdatedBy:  user.ID,
		}))

		resp := postChatContext(t, env, project.ID.String(), "Can you analyze my CI status?")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "ci", body["matchedMetric"])
		assert.Equal(t, "ci", body["contextMetric"])
	})

	t.Run("response is correct when keyword metric does not exist", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "noveluser", "Novel User")
		project := testutil.CreateOwnedProject(t, env, user, "novel-repo", "https://github.com/noveluser/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		// No L1 ci metric — the keyword match fails, falls back to readme.
		resp := postChatContext(t, env, project.ID.String(), "Can you analyze my CI status?")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		// "ci" is recognized but no L1 row, so it falls back to readme.
		assert.Equal(t, "ci", body["matchedMetric"])
		assert.Equal(t, "readme", body["contextMetric"])
		//assert.Equal(t, true, body["usedDefaultReadme"])
	})
}

// ── LLM Context expansion ─────────────────────────────────────────

func TestGetProjectMetricLLMContext_NonOwner(t *testing.T) {
	t.Run("non-owner succeeds", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		alice := testutil.CreateUser(t, env, "alice", "Alice")
		project := testutil.CreateOwnedProject(t, env, alice, "repo", "https://github.com/alice/repo")
		bob := testutil.CreateUser(t, env, "bob", "Bob")
		testutil.CreateUserIdentifier(t, env, bob.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		ctx := context.Background()
		// Seed an L1 metric row so the metric check is hit before the metric check.
		require.NoError(t, env.Queries.UpsertAnalysisMetric(ctx, db.UpsertAnalysisMetricParams{
			ID:         db.NewID(),
			ProjectID:  project.ID,
			MetricType: "readme",
			Level:      1,
			Score:      6,
			Data:       db.JSONMap{"has_readme": true},
			Sha:        "abc123",
			UpdatedBy:  alice.ID,
		}))

		resp := testutil.GetJSON(t, env, llmContextPath(project.ID.String(), "readme"))
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("unauthenticated request succeeds", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		project := testutil.CreateProject(t, env, "gh-public-repo", "https://github.com/owner/repo")
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

		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.NotEmpty(t, body["systemPrompt"])
		assert.NotEmpty(t, body["userPrompt"])
	})

	t.Run("unauthenticated request with no L1 data returns metric-scoped checks", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		project := testutil.CreateProject(t, env, "gh-nodata-repo", "https://github.com/nobody/repo")

		resp := testutil.GetJSON(t, env, llmContextPath(project.ID.String(), "ci"))
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "ci", body["metricType"])
		assert.Equal(t, float64(0), body["l1Score"])
		bodyStr := body["userPrompt"].(string)
		// Should contain the CI-specific checkmarks, not all metrics.
		assert.Contains(t, bodyStr, "missing: Has CI configuration")
		assert.Contains(t, bodyStr, "missing: Has lint step")
		assert.NotContains(t, bodyStr, "has_readme")
		assert.NotContains(t, bodyStr, "has_test_dir")
	})

	t.Run("non-GitHub project works", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateAdminUser(t, env, "gitlabuser", "GitLab User")
		project := testutil.CreateOwnedProject(t, env, user, "project", "https://gitlab.com/gitlab-user/project")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		ctx := context.Background()
		_, err := env.Queries.UpdateProjectService(ctx, db.UpdateProjectServiceParams{
			ID:      project.ID,
			Service: "gitlab",
		})
		require.NoError(t, err)

		resp := testutil.GetJSON(t, env, llmContextPath(project.ID.String(), "readme"))
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "readme", body["metricType"])
		assert.Equal(t, float64(0), body["l1Score"])
		bodyStr := body["userPrompt"].(string)
		assert.Contains(t, bodyStr, "missing: Has README")
	})

	t.Run("unknown metric type is rejected", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		project := testutil.CreateOwnedProject(t, env, user, "repo", "https://github.com/owner/repo")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := testutil.GetJSON(t, env, llmContextPath(project.ID.String(), "notarealmetric"))
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid project ID returns 400", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "owner", "Owner")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		env.LoginAs(t, "test@example.com")

		resp := testutil.GetJSON(t, env, llmContextPath("not-a-uuid", "readme"))
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// chatContextPathJSON returns the JSON response for a chat context request.
func chatContextPathJSON(t *testing.T, env *testutil.TestEnv, projectID, message string) map[string]any {
	resp := postChatContext(t, env, projectID, message)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return body
}

// Must is a helper to fail tests on error.
func Must[T any](t *testing.T, v T, err error) T {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return v
}

// Ensure _unused import is removed by having a proper reference.
var _ = context.Background
var _ = strings.TrimSpace
var _ = fmt.Sprintf
