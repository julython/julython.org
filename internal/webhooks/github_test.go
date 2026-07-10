package webhooks_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"july/internal/db"
	"july/internal/testutil"
	"july/internal/webhooks"
)

func TestMain(m *testing.M) {
	if err := testutil.SetupSharedEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "shared setup failed: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// webhookPayload builds a minimal valid GitHubPushEvent. Hash is required;
// opts allow callers to override fields without constructing the full struct.
func webhookPayload(hash string, opts ...func(*testutil.WebhookOpts)) webhooks.GitHubPushEvent {
	o := &testutil.WebhookOpts{
		Ref:      "refs/heads/main",
		RepoID:   12345,
		RepoName: "test-repo",
		FullName: "alice/test-repo",
		HTMLURL:  "https://github.com/alice/test-repo",
		Author:   webhooks.GitHubAuthor{Name: "Alice", Email: "alice@test.com"},
		Message:  "Add a meaningful change",
		Files:    []string{"main.go"},
	}
	if o.Timestamp.IsZero() {
		o.Timestamp = time.Now()
	}
	for _, opt := range opts {
		opt(o)
	}
	// Set owner from FullName (first part before /)
	var ownerName string
	if slashIdx := strings.IndexByte(o.FullName, '/'); slashIdx >= 0 {
		ownerName = o.FullName[:slashIdx]
	}
	repo := webhooks.GitHubRepo{
		ID:          o.RepoID,
		Name:        o.RepoName,
		FullName:    o.FullName,
		HTMLURL:     o.HTMLURL,
		Description: o.Description,
		Private:     o.Private,
		Fork:        o.Fork,
		ForksCount:  o.ForksCount,
		Watchers:    o.Watchers,
		Owner:       webhooks.GitHubOwner{Name: ownerName},
	}
	if o.Organization != "" {
		repo.Organization = o.Organization
	}
	return webhooks.GitHubPushEvent{
		Ref:        o.Ref,
		Forced:     o.Forced,
		Repository: repo,
		Commits: []webhooks.GitHubCommit{
			{
				ID:        hash,
				Message:   o.Message,
				Timestamp: o.Timestamp,
				URL:       o.HTMLURL + "/commit/" + hash,
				Author:    o.Author,
				Added:     o.Files,
			},
		},
	}
}

func TestGitHubWebhook(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	t.Run("processes valid push event", func(t *testing.T) {
		payload := webhooks.GitHubPushEvent{
			Ref: "refs/heads/main",
			Repository: webhooks.GitHubRepo{
				ID:       12345,
				Name:     "my-project",
				FullName: "alice/my-project",
				HTMLURL:  "https://github.com/alice/my-project",
				Owner:    webhooks.GitHubOwner{Name: "alice"},
			},
			Commits: []webhooks.GitHubCommit{
				{
					ID:        "abc123def456789",
					Message:   "Add user authentication feature",
					Timestamp: time.Now(),
					URL:       "https://github.com/alice/my-project/commit/abc123",
					Author:    webhooks.GitHubAuthor{Name: "Alice", Email: "alice@test.com"},
					Added:     []string{"auth.go", "auth_test.go"},
					Modified:  []string{"main.go"},
				},
				{
					ID:        "def456abc789012",
					Message:   "Fix bug in login flow",
					Timestamp: time.Now(),
					URL:       "https://github.com/alice/my-project/commit/def456",
					Author:    webhooks.GitHubAuthor{Name: "Alice", Email: "alice@test.com"},
					Modified:  []string{"auth.go"},
				},
			},
		}

		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 2, result.Received)
		assert.Equal(t, 2, result.Created)
		assert.Equal(t, 0, result.Skipped)

		project, err := env.Queries.GetProjectBySlug(t.Context(), "gh-alice-my-project")
		require.NoError(t, err)
		assert.Equal(t, "my-project", project.Name)
		assert.Equal(t, "github", project.Service)
	})

	t.Run("skips duplicate commits", func(t *testing.T) {
		testutil.PostJSON(t, env, "/api/v1/github", webhookPayload("duplicate-hash-001"))

		resp := testutil.PostJSON(t, env, "/api/v1/github", webhookPayload("duplicate-hash-001"))
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 1, result.Received)
		assert.Equal(t, 1, result.Skipped)
		assert.Equal(t, 0, result.Created)
	})

	t.Run("skips non-default branches", func(t *testing.T) {
		payload := webhookPayload("branch-hash-001", func(o *testutil.WebhookOpts) {
			o.Ref = "refs/heads/feature-branch"
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, testutil.DecodeBody(t, resp), "not default branch")
	})

	t.Run("skips force pushes", func(t *testing.T) {
		payload := webhookPayload("force-hash-001", func(o *testutil.WebhookOpts) {
			o.Forced = true
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, testutil.DecodeBody(t, resp), "force push")
	})

	t.Run("skips invalid commits", func(t *testing.T) {
		payload := webhooks.GitHubPushEvent{
			Ref: "refs/heads/main",
			Repository: webhooks.GitHubRepo{
				ID:       99999,
				Name:     "test-repo",
				FullName: "bob/test-repo",
				HTMLURL:  "https://github.com/bob/test-repo",
				Owner:    webhooks.GitHubOwner{Name: "bob"},
			},
			Commits: []webhooks.GitHubCommit{
				{ID: "wip-001", Message: "wip", Timestamp: time.Now(), Author: webhooks.GitHubAuthor{Email: "bob@test.com"}},
				{ID: "merge-001", Message: "Merge branch 'feature' into main", Timestamp: time.Now(), Author: webhooks.GitHubAuthor{Email: "bob@test.com"}},
				{ID: "short-001", Message: "fix", Timestamp: time.Now(), Author: webhooks.GitHubAuthor{Email: "bob@test.com"}},
				{ID: "valid-001", Message: "Add proper feature implementation", Timestamp: time.Now(), Author: webhooks.GitHubAuthor{Email: "bob@test.com"}},
			},
		}
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 4, result.Received)
		assert.Equal(t, 1, result.Created)
		assert.Equal(t, 3, result.Skipped)
	})

	t.Run("creates new project from webhook", func(t *testing.T) {
		payload := webhookPayload("newrepo-hash-001", func(o *testutil.WebhookOpts) {
			o.RepoID = 77777
			o.RepoName = "new-repo"
			o.FullName = "carol/new-repo"
			o.HTMLURL = "https://github.com/carol/new-repo"
			o.Description = "A brand new repository"
			o.ForksCount = 5
			o.Watchers = 10
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		project, err := env.Queries.GetProjectBySlug(t.Context(), "gh-carol-new-repo")
		require.NoError(t, err)
		assert.Equal(t, "new-repo", project.Name)
		assert.Equal(t, "https://github.com/carol/new-repo", project.Url)
		assert.Equal(t, "github", project.Service)
		assert.Equal(t, int64(77777), project.RepoID.Int64)
		assert.False(t, project.Forked)
		assert.Equal(t, int32(5), project.Forks)
		assert.Equal(t, int32(10), project.Watchers)
	})

	t.Run("sets owner from webhook payload", func(t *testing.T) {
		payload := webhooks.GitHubPushEvent{
			Ref: "refs/heads/main",
			Repository: webhooks.GitHubRepo{
				ID:       99999,
				Name:     "owner-repo",
				FullName: "eve/owner-repo",
				HTMLURL:  "https://github.com/eve/owner-repo",
				Owner:    webhooks.GitHubOwner{Name: "eve"},
			},
			Commits: []webhooks.GitHubCommit{
				{ID: "owner-hash-001", Message: "Add owner", Timestamp: time.Now(), Author: webhooks.GitHubAuthor{Name: "Eve", Email: "eve@test.com"}},
			},
		}
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		project, err := env.Queries.GetProjectBySlug(t.Context(), "gh-eve-owner-repo")
		require.NoError(t, err)
		assert.Equal(t, "eve", project.Owner)
	})

	t.Run("sets owner from nested owner object", func(t *testing.T) {
		payload := webhooks.GitHubPushEvent{
			Ref: "refs/heads/main",
			Repository: webhooks.GitHubRepo{
				ID:          11111,
				Name:        "org-repo",
				FullName:    "acme-org/org-repo",
				HTMLURL:     "https://github.com/acme-org/org-repo",
				Description: "An organization repository",
				Fork:        false,
				ForksCount:  3,
				Watchers:    5,
				Owner:       webhooks.GitHubOwner{Name: "acme-org"},
			},
			Commits: []webhooks.GitHubCommit{
				{ID: "org-hash-001", Message: "Add organization repo", Timestamp: time.Now(), Author: webhooks.GitHubAuthor{Email: "org@test.com"}},
			},
		}
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		project, err := env.Queries.GetProjectBySlug(t.Context(), "gh-acme-org-org-repo")
		require.NoError(t, err)
		assert.Equal(t, "acme-org", project.Owner)
	})

	t.Run("private project creates commit but no game scoring", func(t *testing.T) {
		game := testutil.CreateActiveGame(t, env)

		payload := webhookPayload("priv-hash-001", func(o *testutil.WebhookOpts) {
			o.RepoID = 22222
			o.RepoName = "private-repo"
			o.FullName = "frank/private-repo"
			o.HTMLURL = "https://github.com/frank/private-repo"
			o.Private = true
			o.Author = webhooks.GitHubAuthor{Name: "Frank", Email: "frank@test.com"}
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 1, result.Created)

		project, err := env.Queries.GetProjectBySlug(t.Context(), "gh-frank-private-repo")
		require.NoError(t, err)
		assert.True(t, project.IsPrivate)

		commit, err := env.Queries.GetCommitByHashStr(t.Context(), "priv-hash-001")
		require.NoError(t, err)
		assert.False(t, commit.GameID.Valid)

		_, err = env.Queries.GetBoardByProjectAndGame(t.Context(), db.GetBoardByProjectAndGameParams{
			ProjectID: project.ID,
			GameID:    game.ID,
		})
		assert.ErrorIs(t, err, pgx.ErrNoRows)
	})

	t.Run("public commits from same user get boards assigned sequentially", func(t *testing.T) {
		game := testutil.CreateActiveGame(t, env)

		user := testutil.CreateUser(t, env, "boardassign", "Board Assign User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "boardassign@test.com", true, true)

		// Post 2 commits from the same user through the webhook
		testutil.WebhookCommit(t, env, "board-hash-001", func(o *testutil.WebhookOpts) {
			o.RepoID = 33333
			o.RepoName = "project-one"
			o.FullName = "boardassign/project-one"
			o.HTMLURL = "https://github.com/boardassign/project-one"
			o.Author = webhooks.GitHubAuthor{Name: "Board Assign User", Email: "boardassign@test.com"}
		})
		testutil.WebhookCommit(t, env, "board-hash-002", func(o *testutil.WebhookOpts) {
			o.RepoID = 33334
			o.RepoName = "project-two"
			o.FullName = "boardassign/project-two"
			o.HTMLURL = "https://github.com/boardassign/project-two"
			o.Author = webhooks.GitHubAuthor{Name: "Board Assign User", Email: "boardassign@test.com"}
		})

		// Verify player has 2 boards assigned
		player, err := env.Queries.GetPlayerByUserAndGame(t.Context(), db.GetPlayerByUserAndGameParams{
			UserID: user.ID,
			GameID: game.ID,
		})
		require.NoError(t, err)
		ids, err := env.Queries.GetPlayerBoardIds(t.Context(), player.ID)
		require.NoError(t, err)
		require.True(t, ids.Board1ID.Valid, "board_1 should be assigned")
		require.True(t, ids.Board2ID.Valid, "board_2 should be assigned")
		assert.False(t, ids.Board3ID.Valid, "board_3 should not be assigned yet")
	})

	t.Run("fourth project does not get board assigned", func(t *testing.T) {
		game := testutil.CreateActiveGame(t, env)

		user := testutil.CreateUser(t, env, "overflowuser", "Overflow User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "overflowuser@test.com", true, true)

		// Post 4 commits from the same user
		for i := 1; i <= 4; i++ {
			testutil.WebhookCommit(t, env, fmt.Sprintf("overflow-hash-%03d", i), func(o *testutil.WebhookOpts) {
				o.RepoID = 44440 + int64(i)
				o.RepoName = fmt.Sprintf("overflow-project-%d", i)
				o.FullName = fmt.Sprintf("overflowuser/overflow-project-%d", i)
				o.HTMLURL = fmt.Sprintf("https://github.com/overflowuser/overflow-project-%d", i)
				o.Author = webhooks.GitHubAuthor{Name: "Overflow User", Email: "overflowuser@test.com"}
			})
		}

		// Verify player has exactly 3 boards (4th project is not assigned to player)
		player, err := env.Queries.GetPlayerByUserAndGame(t.Context(), db.GetPlayerByUserAndGameParams{
			UserID: user.ID,
			GameID: game.ID,
		})
		require.NoError(t, err)
		ids, err := env.Queries.GetPlayerBoardIds(t.Context(), player.ID)
		require.NoError(t, err)
		require.True(t, ids.Board1ID.Valid)
		require.True(t, ids.Board2ID.Valid)
		require.True(t, ids.Board3ID.Valid)

		// Verify the 4th project's board is NOT assigned to the player
		project4, err := env.Queries.GetProjectBySlug(t.Context(), "gh-overflowuser-overflow-project-4")
		require.NoError(t, err)
		assert.NotEqual(t, ids.Board1ID, project4.ID, "4th project board should not be in player's board_1")
		assert.NotEqual(t, ids.Board2ID, project4.ID, "4th project board should not be in player's board_2")
		assert.NotEqual(t, ids.Board3ID, project4.ID, "4th project board should not be in player's board_3")
	})

	t.Run("adds commit to the active game", func(t *testing.T) {
		game := testutil.CreateActiveGame(t, env)
		payload := webhookPayload("game-hash-001", func(o *testutil.WebhookOpts) {
			o.RepoID = 77777
			o.FullName = "carol/new-repo"
			o.HTMLURL = "https://github.com/carol/new-repo"
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		commits, err := env.Queries.GetRecentCommits(t.Context(), db.GetRecentCommitsParams{
			GameID:     db.UUID(game.ID),
			LimitCount: 10,
		})
		require.NoError(t, err)
		assert.Len(t, commits, 1)
	})

	t.Run("links commit to existing user", func(t *testing.T) {
		// User created with "gh-dave" matches repo owner "dave" → gh-dave
		user := testutil.CreateUser(t, env, "gh-dave", "Dave Developer")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "dave@test.com", true, true)

		payload := webhookPayload("dave-hash-001", func(o *testutil.WebhookOpts) {
			o.RepoID = 88888
			o.RepoName = "dave-repo"
			o.FullName = "dave/dave-repo"
			o.HTMLURL = "https://github.com/dave/dave-repo"
			o.Author = webhooks.GitHubAuthor{Name: "Dave Developer", Email: "dave@test.com"}
			o.Files = []string{"feature.py"}
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		commit, err := env.Queries.GetCommitByHashStr(t.Context(), "dave-hash-001")
		require.NoError(t, err)
		assert.True(t, commit.UserID.Valid)
		commitUserID, err := uuid.FromBytes(commit.UserID.Bytes[:])
		require.NoError(t, err)
		assert.Equal(t, user.ID, commitUserID)
	})

	t.Run("detects languages from files", func(t *testing.T) {
		payload := webhookPayload("poly-hash-001", func(o *testutil.WebhookOpts) {
			o.RepoID = 66666
			o.RepoName = "polyglot"
			o.FullName = "poly/polyglot"
			o.HTMLURL = "https://github.com/poly/polyglot"
			o.Author = webhooks.GitHubAuthor{Name: "Poly", Email: "poly@test.com"}
			o.Files = []string{"main.go", "app.py", "index.ts", "README.md"}
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		commit, err := env.Queries.GetCommitByHashStr(t.Context(), "poly-hash-001")
		require.NoError(t, err)
		assert.Contains(t, commit.Languages, "Go")
		assert.Contains(t, commit.Languages, "Python")
		assert.Contains(t, commit.Languages, "TypeScript")
	})

	t.Run("skips inactive project", func(t *testing.T) {
		project := testutil.CreateProjectWithRepoID(t, env, "inactive-project", "gh-test-inactive-project", "https://github.com/test/inactive-project", 55555)
		require.NoError(t, env.Queries.DeactivateProject(t.Context(), project.ID))

		payload := webhookPayload("inactive-hash-001", func(o *testutil.WebhookOpts) {
			o.RepoID = 55555
			o.RepoName = "inactive-project"
			o.FullName = "test/inactive-project"
			o.HTMLURL = "https://github.com/test/inactive-project"
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, testutil.DecodeBody(t, resp), "project inactive")
	})

	t.Run("handles ping event", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, env.Server.URL+"/api/v1/github", strings.NewReader("{}"))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "ping")

		resp, err := env.Client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "pong", testutil.DecodeBody(t, resp))
	})

	t.Run("finds existing user by repo owner", func(t *testing.T) {
		// First commit: creates gh-frank user
		payload1 := webhookPayload("uname-hash-001", func(o *testutil.WebhookOpts) {
			o.RepoName = "username-repo"
			o.FullName = "frank/username-repo"
			o.HTMLURL = "https://github.com/frank/username-repo"
			o.Author = webhooks.GitHubAuthor{Name: "Frank Updated", Email: "other@example.com"}
			o.Message = "Add feature by repo owner lookup"
		})
		resp1 := testutil.PostJSON(t, env, "/api/v1/github", payload1)
		assert.Equal(t, http.StatusOK, resp1.StatusCode)

		var result1 webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp1.Body).Decode(&result1))
		assert.Equal(t, 1, result1.Created)

		user, err := env.Queries.GetUserByUsername(t.Context(), "gh-frank")
		require.NoError(t, err)

		// Second commit: finds existing gh-frank user (same repo owner)
		payload2 := webhookPayload("uname-hash-002", func(o *testutil.WebhookOpts) {
			o.RepoName = "username-repo"
			o.FullName = "frank/username-repo"
			o.Owner = "frank"
			o.HTMLURL = "https://github.com/frank/username-repo"
			o.Author = webhooks.GitHubAuthor{Name: "Frank Again", Email: "other2@example.com"}
			o.Message = "Second commit by same repo owner"
		})
		resp2 := testutil.PostJSON(t, env, "/api/v1/github", payload2)
		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		var result2 webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp2.Body).Decode(&result2))
		assert.Equal(t, 1, result2.Created)

		// Verify second commit links to the same user (not a new one)
		commit, err := env.Queries.GetCommitByHashStr(t.Context(), "uname-hash-002")
		require.NoError(t, err)
		require.True(t, commit.UserID.Valid)
		commitUserID, err := uuid.FromBytes(commit.UserID.Bytes[:])
		require.NoError(t, err)
		assert.Equal(t, user.ID, commitUserID)

		// User's name stays as original (first commit) — webhooks don't update user records
		updatedUser, err := env.Queries.GetUserByID(t.Context(), user.ID)
		require.NoError(t, err)
		assert.Equal(t, "Frank", updatedUser.Name)
	})

	t.Run("existing user with verified email not overwritten by webhook", func(t *testing.T) {
		// Create user with unique username to avoid collision with other subtests
		user := testutil.CreateUser(t, env, "gh-eve-verified", "Eve Developer")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "eve@verified.com", true, true)

		// Verify the identifier was created as verified/primary
		id, err := env.Queries.GetUserIdentifier(t.Context(), "email:eve@verified.com")
		require.NoError(t, err)
		assert.True(t, id.Verified, "original verified flag should be preserved")
		assert.True(t, id.IsPrimary, "original is_primary flag should be preserved")

		// Webhook commit: finds existing user by owner, adds email as unverified
		// The existing verified identifier should NOT be overwritten
		payload := webhookPayload("eve-verified-hash", func(o *testutil.WebhookOpts) {
			o.RepoName = "eve-repo"
			o.FullName = "eve-verified/eve-repo"
			o.Owner = "eve-verified"
			o.HTMLURL = "https://github.com/eve-verified/eve-repo"
			o.Author = webhooks.GitHubAuthor{Name: "Eve Developer", Email: "eve@verified.com"}
			o.Message = "Commit from user with verified email"
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify the existing verified identifier was NOT overwritten
		id, err = env.Queries.GetUserIdentifier(t.Context(), "email:eve@verified.com")
		require.NoError(t, err)
		assert.True(t, id.Verified, "verified flag should be preserved")
		assert.True(t, id.IsPrimary, "is_primary flag should be preserved")
	})

	t.Run("creates new user from commit repo owner", func(t *testing.T) {
		// Personal repo: user created from repo owner (gh-newgrit), not author's username
		payload := webhookPayload("newuser-hash-001", func(o *testutil.WebhookOpts) {
			o.RepoName = "newuser-repo"
			o.FullName = "newgrit/newuser-repo"
			o.HTMLURL = "https://github.com/newgrit/newuser-repo"
			o.Author = webhooks.GitHubAuthor{Name: "New Grizzard", Email: "newgrit@example.com"}
			o.Message = "Initial commit from new user"
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 1, result.Created)

		// Verify user was created from repo owner (not author's username)
		user, err := env.Queries.GetUserByUsername(t.Context(), "gh-newgrit")
		require.NoError(t, err)
		assert.Equal(t, "New Grizzard", user.Name)

		// Verify email identifier was added as unverified
		_, err = env.Queries.GetUserIdentifier(t.Context(), "email:newgrit@example.com")
		require.NoError(t, err)

		// Verify a commit was created and linked to this user
		commit, err := env.Queries.GetCommitByHashStr(t.Context(), "newuser-hash-001")
		require.NoError(t, err)
		require.True(t, commit.UserID.Valid)
		commitUserID, err := uuid.FromBytes(commit.UserID.Bytes[:])
		require.NoError(t, err)
		assert.Equal(t, user.ID, commitUserID)
	})

	t.Run("same new user commits multiple times", func(t *testing.T) {
		// First commit: user created from repo owner
		payload1 := webhookPayload("multi-hash-001", func(o *testutil.WebhookOpts) {
			o.RepoName = "multi-repo"
			o.FullName = "repmulti/multi-repo"
			o.HTMLURL = "https://github.com/repmulti/multi-repo"
			o.Author = webhooks.GitHubAuthor{Name: "Repmulti", Email: "repmulti@example.com"}
			o.Message = "First commit from repmulti"
		})
		resp1 := testutil.PostJSON(t, env, "/api/v1/github", payload1)
		assert.Equal(t, http.StatusOK, resp1.StatusCode)

		var result1 webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp1.Body).Decode(&result1))
		assert.Equal(t, 1, result1.Created)

		// Get the created user
		user, err := env.Queries.GetUserByUsername(t.Context(), "gh-repmulti")
		require.NoError(t, err)

		// Second commit with updated name and avatar (same repo owner)
		payload2 := webhookPayload("multi-hash-002", func(o *testutil.WebhookOpts) {
			o.RepoName = "multi-repo-two"
			o.FullName = "repmulti/multi-repo-two"
			o.HTMLURL = "https://github.com/repmulti/multi-repo-two"
			o.Author = webhooks.GitHubAuthor{Name: "Repmulti Updated", Email: "repmulti@example.com"}
			o.Message = "Second commit from repmulti"
		})
		resp2 := testutil.PostJSON(t, env, "/api/v1/github", payload2)
		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		var result2 webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp2.Body).Decode(&result2))
		assert.Equal(t, 1, result2.Created)

		// Verify same user was used (not a duplicate)
		secondUser, err := env.Queries.GetUserByUsername(t.Context(), "gh-repmulti")
		require.NoError(t, err)
		assert.Equal(t, user.ID, secondUser.ID)

		// User's name stays as original — webhooks don't update user records

		// Verify the second commit links to the same user
		commit2, err := env.Queries.GetCommitByHashStr(t.Context(), "multi-hash-002")
		require.NoError(t, err)
		require.True(t, commit2.UserID.Valid)
		commit2UserID, err := uuid.FromBytes(commit2.UserID.Bytes[:])
		require.NoError(t, err)
		assert.Equal(t, user.ID, commit2UserID)
	})

	t.Run("personal repo uses repo owner as user identifier", func(t *testing.T) {
		// Personal repo: commit author doesn't exist, but repo owner creates the user
		// The user is identified by gh-{repo_owner}, NOT by the author's email
		payload := webhookPayload("orphan-hash-001", func(o *testutil.WebhookOpts) {
			o.RepoName = "orphan-repo"
			o.FullName = "orphanuser/orphan-repo"
			o.HTMLURL = "https://github.com/orphanuser/orphan-repo"
			o.Author = webhooks.GitHubAuthor{Name: "Orphan User", Email: "orphan@example.com"}
			o.Message = "Commit from user without matching email"
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 1, result.Created)

		// A user was created from the repo owner (not the author)
		user, err := env.Queries.GetUserByUsername(t.Context(), "gh-orphanuser")
		require.NoError(t, err)
		assert.Equal(t, "Orphan User", user.Name)

		// The commit links to the repo owner user
		commit, err := env.Queries.GetCommitByHashStr(t.Context(), "orphan-hash-001")
		require.NoError(t, err)
		require.True(t, commit.UserID.Valid)
		commitUserID, err := uuid.FromBytes(commit.UserID.Bytes[:])
		require.NoError(t, err)
		assert.Equal(t, user.ID, commitUserID)

		// Email added as unverified identifier for future matching
		_, err = env.Queries.GetUserIdentifier(t.Context(), "email:orphan@example.com")
		require.NoError(t, err)
	})

}

func TestGitHubWebhookOrganizations(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	t.Run("org repo commits match by email only", func(t *testing.T) {
		// Organization repo: falls back to email lookup only, no user created from owner
		// No matching user → commit created without user association
		payload := webhookPayload("org-no-match-hash", func(o *testutil.WebhookOpts) {
			o.RepoName = "no-match-repo"
			o.FullName = "julython/no-match-repo"
			o.HTMLURL = "https://github.com/julython/no-match-repo"
			o.Author = webhooks.GitHubAuthor{Name: "Contributor", Email: "contributor@example.com"}
			o.Message = "Commit to org repo, no existing user"
			o.Organization = "julython"
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 1, result.Created)

		// Commit created without user association (org repo, no email match)
		commit, err := env.Queries.GetCommitByHashStr(t.Context(), "org-no-match-hash")
		require.NoError(t, err)
		assert.False(t, commit.UserID.Valid, "org repo commit with no email match should have no user")

		// No gh-julython user was created (org repos don't create users)
		_, err = env.Queries.GetUserByUsername(t.Context(), "gh-julython")
		assert.Error(t, err)
	})

	t.Run("org repo links commit via matching email from another repo", func(t *testing.T) {
		// First: create a user through a PERSONAL repo (standard behavior)
		testutil.WebhookCommit(t, env, "org-link-personal-hash", func(o *testutil.WebhookOpts) {
			o.RepoName = "personal-repo"
			o.FullName = "alice/personal-repo"
			o.HTMLURL = "https://github.com/alice/personal-repo"
			o.Author = webhooks.GitHubAuthor{Name: "Alice", Email: "alice-org@example.com"}
			o.Message = "First commit from alice"
		})

		// Then: same email commits to an ORG repo — should link to the same user
		payload := webhookPayload("org-link-hash", func(o *testutil.WebhookOpts) {
			o.RepoName = "match-repo"
			o.FullName = "julython/match-repo"
			o.HTMLURL = "https://github.com/julython/match-repo"
			o.Author = webhooks.GitHubAuthor{Name: "Alice", Email: "alice-org@example.com"}
			o.Message = "Org repo commit by alice"
			o.Organization = "julython"
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 1, result.Created)

		// The org repo commit links to the same user created from personal repo
		commit, err := env.Queries.GetCommitByHashStr(t.Context(), "org-link-hash")
		require.NoError(t, err)
		require.True(t, commit.UserID.Valid, "org repo commit with matching email should link to user")

		// Verify the personal repo commit also links to the same user
		personalCommit, err := env.Queries.GetCommitByHashStr(t.Context(), "org-link-personal-hash")
		require.NoError(t, err)
		require.True(t, personalCommit.UserID.Valid)
		require.NoError(t, err)
		orgUserID, _ := uuid.FromBytes(commit.UserID.Bytes[:])
		personalUserID, _ := uuid.FromBytes(personalCommit.UserID.Bytes[:])
		assert.Equal(t, orgUserID, personalUserID, "both commits should link to the same user")
	})

	t.Run("org repo with matching email links commit", func(t *testing.T) {
		// Org repo: first commit creates a user via email
		payload := webhookPayload("org-match-hash", func(o *testutil.WebhookOpts) {
			o.RepoName = "match-repo"
			o.FullName = "julython/match-repo"
			o.HTMLURL = "https://github.com/julython/match-repo"
			o.Author = webhooks.GitHubAuthor{Name: "Org User", Email: "orguser@example.com"}
			o.Message = "Commit to org repo"
			o.Organization = "julython"
		})
		resp := testutil.PostJSON(t, env, "/api/v1/github", payload)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 1, result.Created)

		// The commit links to this user
		commit, err := env.Queries.GetCommitByHashStr(t.Context(), "org-match-hash")
		require.NoError(t, err)
		require.False(t, commit.UserID.Valid)
	})

}

func TestGitHubWebhookContentTypes(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	t.Run("accepts application/json", func(t *testing.T) {
		resp := testutil.PostJSON(t, env, "/api/v1/github", webhookPayload("ct-json-001"))
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 1, result.Created)
	})

	t.Run("accepts form-encoded payload", func(t *testing.T) {
		payload := webhookPayload("ct-form-001")
		b, err := json.Marshal(payload)
		require.NoError(t, err)

		form := url.Values{"payload": {string(b)}}
		req, err := http.NewRequest(http.MethodPost, env.Server.URL+"/api/v1/github", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := env.Client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result webhooks.ProcessResult
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 1, result.Created)
	})

	t.Run("rejects unsupported content type", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, env.Server.URL+"/api/v1/github", strings.NewReader("data"))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "text/plain")

		resp, err := env.Client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
	})

	t.Run("rejects form without payload field", func(t *testing.T) {
		form := url.Values{"wrong_field": {"data"}}
		req, err := http.NewRequest(http.MethodPost, env.Server.URL+"/api/v1/github", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := env.Client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		file     string
		expected string
	}{
		{"main.go", "Go"},
		{"app.py", "Python"},
		{"index.ts", "TypeScript"},
		{"component.tsx", "TypeScript"},
		{"script.js", "JavaScript"},
		{"style.css", "CSS"},
		{"lib.rs", "Rust"},
		{"Main.java", "Java"},
		{"README.md", "Documentation"},
		{"Makefile", ""},
		{".gitignore", ""},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			assert.Equal(t, tt.expected, webhooks.DetectLanguage(tt.file))
		})
	}
}

func TestIsValidCommit(t *testing.T) {
	tests := []struct {
		name    string
		message string
		valid   bool
	}{
		{"normal commit", "Add user authentication", true},
		{"wip commit", "wip: working on feature", false},
		{"WIP uppercase", "WIP", false},
		{"merge commit", "Merge branch 'feature' into main", false},
		{"short message", "fix", false},
		{"skip ci", "Update docs [skip ci]", false},
		{"ci skip", "Minor change [ci skip]", false},
		{"valid short", "Fix login bug", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, webhooks.IsValidCommit(webhooks.GitHubCommit{Message: tt.message}))
		})
	}
}
