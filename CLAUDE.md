# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Julython is a platform for a month-long programming challenge from July 1st to July 31st, encouraging participants to write code and open source it during the month. The platform tracks contributions and provides leaderboards to encourage participation.

## Codebase Structure

The codebase is a Go application with the following key components:

- `cmd/server/main.go` - Main entry point for the application
- `internal/` - Core application logic:
  - `api/` - HTTP router and middleware
  - `config/` - Configuration management
  - `db/` - Database interactions (using SQLC)
  - `components/` - Templ components to render html
  - `handlers/` - HTTP request handlers
  - `services/` - Business logic services
  - `webhooks/` - GitHub webhook handling
- `web/` - Static asset handling and web server components
- `web/assets/` - Static assets including CSS, JS, and vendor libraries
- `web/vendor/mlc-web-llm/` - Vendored WebLLM library for browser-based LLM inference

## Development Setup

### Prerequisites
- Go 1.21+
- Docker
- Make

### Setup Commands
```bash
make setup     # Install dependencies and dev tools
make dev       # Run the development server with hot reloading
make test      # Run tests
make generate  # Generate code (SQLC, Templ, assets)
```

### Key Development Tasks
- Run `make dev` to start the development server
- Use `make test` to run tests
- Use `make test-watch` to run tests in watch mode
- Use `make migrate-up` to run database migrations
- Use `make i18n` to sync internationalization keys

## Key Features

- GitHub OAuth authentication
- Webhook integration for tracking commits
- Leaderboard system for tracking contributions
- Project analysis with LLM integration (via WebLLM)
- Internationalization support
- Database migrations with go-migrate

## Database

The application uses PostgreSQL with database migrations managed by go-migrate. The database schema is defined in `migrations/` directory.

## Authentication

Authentication is handled via GitHub OAuth. For local development, you'll need to set up a GitHub OAuth application with callback URL: `http://localhost:8000/auth/callback`.

## Testing

Tests are run using `gotestsum` with race detection enabled. The test suite includes unit tests for services and handlers.

## Templ

Templ conventions and patterns are documented in [`docs/templ-conventions.md`](docs/templ-conventions.md). Key points:
- Use `templ.KV()` for conditional CSS classes
- Use `templ.SafeURL()` for dynamic URLs
- Avatar fallback: show first char in a colored circle
- HTMX loading states: `htmx-idle` / `htmx-busy` span pattern
- See the conventions doc for known inconsistencies to fix.

## Assets

Static assets (CSS, JS, vendor libraries) are managed through:
- Tailwind CSS for styling
- HTMX for dynamic UI interactions
- Mermaid for diagram rendering
- WebLLM for browser-based LLM inference

The WebLLM library is vendored from npm and is used for project analysis features.
