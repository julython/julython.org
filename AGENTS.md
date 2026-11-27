# AGENTS.md

This file describes the Julython application and how changes are made using OpenSpec.

## Codebase Structure

Go application (Julython website):

- `cmd/server/main.go` — Main entry point
- `cmd/assetgen`, `cmd/creategame`, `cmd/i18nsync` — CLI tools
- `internal/features/` — Feature-based layout (blog, game, help, players, profile, projects, proxy, assets)
- `internal/api/` — HTTP router and middleware
- `internal/auth/` — Authentication
- `internal/config/` — Configuration
- `internal/db/` — Database interactions (SQLC)
- `internal/i18n/` — Internationalization
- `internal/metrics/` — Metrics
- `internal/services/` — Shared business logic
- `internal/shared/` — Shared packages
- `internal/webhooks/` — GitHub webhook handling
- `web/` — Static assets and web server
- `migrations/` — Database migrations (go-migrate)
- `openspec/` — OpenSpec project configuration (`openspec/config.yaml`)

Features live under `internal/features/` and own their handlers, templ components, services, and tests. Cross-cutting shared code lives in `internal/services/` and `internal/shared/`.

## Making Changes with OpenSpec

Changes to this repository are planned and implemented using OpenSpec workflows. Before starting any change, run an OpenSpec workflow (e.g. `/openspec-propose`) to produce a spec, design, and tasks artifact in `openspec/changes/`. Implement tasks from that artifact, then verify against it.

Project configuration lives at `openspec/config.yaml`. It holds schema, context, and rules that shape how OpenSpec generates artifacts. Keep rules short and additive — they extend OpenSpec's built-in guidance.

## Key Commands

```bash
make dev         # Dev server with hot reloading (air)
make generate    # SQLC, Templ, and asset generation
make test        # Run tests (gotestsum with race detection)
make test-cover  # Run tests with coverage report
make migrate-up  # Run database migrations up
make i18n        # Sync missing i18n keys
```

## Testing

- Always use `make test` to run tests. Never use `go test` directly.
- Default is `./...` (all packages). Override with `make test tests=./internal/webhooks/`.
  Combine with test flags: `make test tests=./internal/webhooks/ args="-count=1 -run TestFoo/Bar"`.

## Migrations

1. Start with a migration: `make migrate-new name=description`
2. Update SQL queries in `internal/db/`
3. Run `make sqlc` to regenerate Go types
4. Never edit generated SQLC Go files — they will be overwritten

## Frontend

Conventions are documented in `docs/templ-conventions.md` — read that before editing `.templ` files.

- `.templ` and `.go` files use tabs for indentation. Never use spaces to indent these files. The transport layer mangles tabs, so be careful when editing — always preserve existing tab indentation.
- UI components are `.templ` files — run `make templ` to regenerate Go code
- Styling is Tailwind — run `make tailwind` to update assets

## Localization

Add new strings to Go code first, then run `make i18n` to sync to `internal/i18n/locales`

## Local Development

- GitHub OAuth callback URL: `http://localhost:8000/auth/callback`
- Run `make compose-deps` to start PostgreSQL and Adminer via Docker Compose
- Set `DATABASE_URL` env var (defaults to `postgres://postgres:postgres@localhost:5432/july?sslmode=disable`)
