# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Codebase Structure

Go application with:

- `cmd/server/main.go` — Main entry point
- `cmd/assetgen`, `cmd/creategame`, `cmd/i18nsync` — CLI tools
- `internal/api/` — HTTP router and middleware
- `internal/handlers/` — HTTP request handlers
- `internal/services/` — Business logic
- `internal/components/` — Templ components
- `internal/db/` — Database interactions (SQLC)
- `internal/webhooks/` — GitHub webhook handling
- `internal/config/` — Configuration
- `internal/i18n/` — Internationalization
- `web/` — Static assets and web server
- `migrations/` — Database migrations (go-migrate)

## Key Commands

```bash
make dev         # Dev server with hot reloading (air)
make generate    # SQLC, Templ, and asset generation
make test        # Run tests (gotestsum with race detection)
make test-watch  # Watch mode for tests
make migrate-up  # Run database migrations up
make i18n        # Sync missing i18n keys
```

## Migrations

1. Start with a migration: `make migrate-new name=description`
2. Update SQL queries in `internal/db/`
3. Run `make sqlc` to regenerate Go types
4. **Never edit generated SQLC Go files** — they will be overwritten

## Frontend

Conventions are documented in [`docs/templ-conventions.md`](docs/templ-conventions.md) — read that before editing `.templ` files.

- **ABSOLUTELY NEVER use the `Edit` tool on `.templ` or `.go` files.** They use tabs and `Edit` mangles tabs during transport. Use `Write` (full-file rewrite) for all `.templ` and `.go` files.
- UI components are `.templ` files — run `make templ` to regenerate Go code
- Styling is Tailwind — run `make tailwind` to update assets

## Localization

- Add new strings to Go code first, then run `make i18n` to sync to `internal/i18n/locales`

## Local Development

- GitHub OAuth callback URL: `http://localhost:8000/auth/callback`
- Run `make compose-deps` to start PostgreSQL and Adminer via Docker Compose
- Set `DATABASE_URL` env var (defaults to `postgres://postgres:postgres@localhost:5432/july?sslmode=disable`)
