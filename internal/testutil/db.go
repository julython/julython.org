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
	"july/internal/services"
)

var (
	sharedPool      *pgxpool.Pool
	sharedContainer testcontainers.Container
	setupOnce       sync.Once
	setupErr        error
	sharedCfg       *config.Config
)

type TestEnv struct {
	Server      *httptest.Server
	Client      *http.Client
	Pool        *pgxpool.Pool
	Queries     *db.Queries
	GameService *services.GameService
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

	ctx := context.Background()
	if err := truncateTables(ctx, sharedPool); err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}
	if err := ensureSystemUser(ctx, sharedPool); err != nil {
		t.Fatalf("failed to ensure system user: %v", err)
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: zerolog.NewTestWriter(t)}).
		With().Timestamp().Logger()

	router := api.NewRouter(sharedPool, sharedCfg, logger)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	queries := db.New(sharedPool)

	return &TestEnv{
		Pool:        sharedPool,
		Queries:     queries,
		Server:      server,
		Client:      server.Client(),
		GameService: services.NewGameService(queries),
	}
}

func ensureSystemUser(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, username, role)
		VALUES ($1, 'Julython System', 'julython-system', 'admin')
		ON CONFLICT (id) DO NOTHING
	`, db.SystemUserID)
	return err
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
