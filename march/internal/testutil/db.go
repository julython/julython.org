package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"july/internal/api"
	"july/internal/config"
	"july/internal/db"
	"july/internal/i18n"
)

var (
	sharedPool      *pgxpool.Pool
	sharedContainer testcontainers.Container
	setupOnce       sync.Once
	setupErr        error
	sharedCfg       *config.Config
)

type TestEnv struct {
	Server  *httptest.Server
	Client  *http.Client
	Pool    *pgxpool.Pool
	Queries *db.Queries
}

// SetupSharedEnv initializes the shared container, pool, and config once.
// Call this from TestMain — it has no *testing.T dependency.
func SetupSharedEnv() error {
	ctx := context.Background()

	setupOnce.Do(func() {
		if err := i18n.Init(); err != nil {
			setupErr = fmt.Errorf("failed to init i18n: %w", err)
			return
		}

		container, err := postgres.Run(ctx,
			"postgres:16-alpine",
			postgres.WithDatabase("july_test"),
			postgres.WithUsername("test"),
			postgres.WithPassword("test"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(30*time.Second),
			),
		)
		if err != nil {
			setupErr = fmt.Errorf("failed to start postgres: %w", err)
			return
		}
		sharedContainer = container

		connStr, err := container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			setupErr = fmt.Errorf("failed to get connection string: %w", err)
			return
		}

		if err := runMigrations(connStr); err != nil {
			setupErr = fmt.Errorf("migrations failed: %w", err)
			return
		}

		pool, err := pgxpool.New(ctx, connStr)
		if err != nil {
			setupErr = fmt.Errorf("failed to connect: %w", err)
			return
		}
		sharedPool = pool

		cfg, err := config.Load()
		if err != nil {
			setupErr = fmt.Errorf("failed to load config: %w", err)
			return
		}
		sharedCfg = cfg
	})

	return setupErr
}

// SetupTestEnv truncates tables and creates a fresh server for each test.
// The logger routes through t so output is scoped to the test.
func SetupTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	if setupErr != nil {
		t.Fatalf("shared setup failed: %v", setupErr)
	}
	if sharedPool == nil {
		// Support calling SetupTestEnv directly without a prior TestMain
		if err := SetupSharedEnv(); err != nil {
			t.Fatalf("test setup failed: %v", err)
		}
	}

	if err := truncateTables(context.Background(), sharedPool); err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: zerolog.NewTestWriter(t)}).
		With().Timestamp().Logger()

	router := api.NewTestRouter(sharedPool, sharedCfg, logger)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return &TestEnv{
		Pool:    sharedPool,
		Queries: db.New(sharedPool),
		Server:  server,
		Client:  server.Client(),
	}
}

func truncateTables(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		TRUNCATE TABLE
			audit_logs,
			reports,
			team_boards,
			team_members,
			teams,
			language_boards,
			languages,
			boards,
			players,
			commits,
			projects,
			games,
			user_identifiers,
			users
		RESTART IDENTITY CASCADE
	`)
	return err
}

func runMigrations(connStr string) error {
	m, err := migrate.New("file://../../migrations", connStr)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

// Fixtures

func CreateUser(t *testing.T, env *TestEnv, username, name string) db.User {
	t.Helper()

	user, err := env.Queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:        db.NewID(),
		Name:      name,
		Username:  username,
		AvatarUrl: db.Text(""),
		Role:      "user",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func CreateUserIdentifier(t *testing.T, env *TestEnv, userID uuid.UUID, idType, value string, verified, primary bool) db.UserIdentifier {
	t.Helper()

	key := fmt.Sprintf("%s:%s", idType, value)
	identifier, err := env.Queries.UpsertUserIdentifier(context.Background(), db.UpsertUserIdentifierParams{
		Value:     key,
		Type:      idType,
		UserID:    userID,
		Verified:  verified,
		IsPrimary: primary,
		Data:      []byte("{}"),
	})
	if err != nil {
		t.Fatalf("failed to create user identifier: %v", err)
	}
	return identifier
}

func CreateProject(t *testing.T, env *TestEnv, slug, repoURL string) db.Project {
	t.Helper()

	// slug is already the canonical form: e.g. "gh-alice-my-repo"
	project, err := env.Queries.CreateProject(context.Background(), db.CreateProjectParams{
		ID:          db.NewID(),
		Url:         repoURL,
		Name:        slug,
		Slug:        slug,
		Description: db.NullText(),
		RepoID:      pgtype.Int8{},
		Service:     "github",
		Forked:      false,
		Forks:       0,
		Watchers:    0,
		ParentUrl:   db.NullText(),
	})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	return project
}

// CreateOwnedProject builds a slug from the user's username so that
// canEditProject will allow that user to edit the project.
//
//	project := testutil.CreateOwnedProject(t, env, user, "my-repo", repoURL)
func CreateOwnedProject(t *testing.T, env *TestEnv, owner db.User, repo, repoURL string) db.Project {
	t.Helper()
	slug := fmt.Sprintf("gh-%s-%s", owner.Username, repo)
	return CreateProject(t, env, slug, repoURL)
}

func CreateProjectWithRepoID(t *testing.T, env *TestEnv, name, slug, repoURL string, repoID int64) db.Project {
	t.Helper()

	project, err := env.Queries.CreateProject(context.Background(), db.CreateProjectParams{
		ID:          db.NewID(),
		Url:         repoURL,
		Name:        name,
		Slug:        slug,
		Description: db.NullText(),
		RepoID:      db.BigInt(repoID),
		Service:     "github",
		Forked:      false,
		Forks:       0,
		Watchers:    0,
		ParentUrl:   db.NullText(),
	})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	return project
}

func CreateCommit(t *testing.T, env *TestEnv, projectID uuid.UUID, hash, message string) db.Commit {
	t.Helper()

	commit, err := env.Queries.CreateCommit(context.Background(), db.CreateCommitParams{
		ID:        db.NewID(),
		Hash:      db.Text(hash),
		ProjectID: projectID,
		UserID:    db.NullUUID(),
		GameID:    db.NullUUID(),
		Author:    db.Text("Test Author"),
		Email:     db.Text("test@example.com"),
		Message:   message,
		Url:       fmt.Sprintf("https://github.com/owner/repo/commit/%s", hash),
		Timestamp: time.Now(),
		Languages: []string{},
		Files:     []byte("[]"),
	})
	if err != nil {
		t.Fatalf("failed to create commit: %v", err)
	}
	return commit
}

func CreateGame(t *testing.T, env *TestEnv, name string, startAt, endAt time.Time, isActive bool) db.Game {
	t.Helper()

	game, err := env.Queries.CreateGame(context.Background(), db.CreateGameParams{
		ID:            db.NewID(),
		Name:          name,
		StartsAt:      startAt,
		EndsAt:        endAt,
		CommitPoints:  1,
		ProjectPoints: 10,
		IsActive:      isActive,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}
	return game
}

func CreateActiveGame(t *testing.T, env *TestEnv) db.Game {
	t.Helper()

	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour)
	end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC).Add(24 * time.Hour)

	return CreateGame(t, env,
		fmt.Sprintf("Test Game %s", db.NewID().String()[:8]),
		start,
		end,
		true,
	)
}

// CreateTestScenario sets up a user with a project and commit for common test cases
func CreateTestScenario(t *testing.T, env *TestEnv) (db.User, db.Project, db.Commit) {
	t.Helper()

	user := CreateUser(t, env, "testuser", "Test User")
	CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
	CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)

	project := CreateProject(t, env, "test-project", "https://github.com/testuser/test-project")
	commit := CreateCommit(t, env, project.ID, "abc123def456", "Initial commit")

	return user, project, commit
}

// CreateGameScenario sets up a full game scenario with user, project, and active game
func CreateGameScenario(t *testing.T, env *TestEnv) (db.User, db.Project, db.Game, db.Commit) {
	t.Helper()

	user, project, _ := CreateTestScenario(t, env)
	game := CreateActiveGame(t, env)
	hash := fmt.Sprintf("game-%s", db.NewID().String()[:8])
	// Create a commit within the game period
	commit, err := env.Queries.CreateCommit(context.Background(), db.CreateCommitParams{
		ID:        db.NewID(),
		Hash:      db.Text(hash),
		ProjectID: project.ID,
		UserID:    db.UUID(user.ID),
		GameID:    db.UUID(game.ID),
		Author:    db.Text(user.Name),
		Email:     db.Text("test@example.com"),
		Message:   "Game commit",
		Url:       "https://github.com/testuser/test-project/commit/abc123",
		Timestamp: time.Now(),
		Languages: []string{"Go", "Python"},
		Files:     []byte(`[{"file":"main.go","type":"added","language":"Go"}]`),
	})
	if err != nil {
		t.Fatalf("failed to create game commit: %v", err)
	}

	return user, project, game, commit
}

// Request Helpers

func PostJSON(t *testing.T, env *TestEnv, path string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := env.Client.Post(env.Server.URL+path, "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	return resp
}

func GetJSON(t *testing.T, env *TestEnv, path string) *http.Response {
	t.Helper()
	resp, err := env.Client.Get(env.Server.URL + path)
	require.NoError(t, err)
	return resp
}

func DecodeBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return strings.TrimSpace(string(b))
}

// LoggerForTest returns a console logger routing through t.Log.
// Exported so test packages can use it when constructing services directly.
func LoggerForTest(t *testing.T) zerolog.Logger {
	t.Helper()
	return zerolog.New(zerolog.ConsoleWriter{Out: zerolog.NewTestWriter(t)}).
		With().Timestamp().Logger()
}
