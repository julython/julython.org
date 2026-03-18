git_sha  := $(shell git rev-parse --short=8 HEAD)
image    := gcr.io/julython/julython-go:$(git_sha)

default: help

ifneq (,$(wildcard .env))
    include .env
    export
endif

export TZ=UTC

# ============================================
# Platform detection (for Tailwind binary)
# ============================================

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_S),Darwin)
    ifeq ($(UNAME_M),arm64)
        TAILWIND_PLATFORM := macos-arm64
    else
        TAILWIND_PLATFORM := macos-x64
    endif
else
    ifeq ($(UNAME_M),aarch64)
        TAILWIND_PLATFORM := linux-arm64
    else
        TAILWIND_PLATFORM := linux-x64
    endif
endif

TAILWIND_VERSION := 4.1.18
HTMX_VERSION     := 2.0.8
MERMAID_VERSION  := 11.4.1
TAILWIND_BIN     := bin/tailwindcss
TAILWIND_URL     := https://github.com/tailwindlabs/tailwindcss/releases/download/v$(TAILWIND_VERSION)/tailwindcss-$(TAILWIND_PLATFORM)
HTMX_URL         := https://unpkg.com/htmx.org@$(HTMX_VERSION)/dist/htmx.min.js
HTMX_DEST        := web/assets/htmx-$(HTMX_VERSION).min.js
MERMAID_URL      := https://cdn.jsdelivr.net/npm/mermaid@$(MERMAID_VERSION)/dist/mermaid.min.js
MERMAID_DEST     := web/assets/mermaid-$(MERMAID_VERSION).min.js
LOCALES_DIR      := internal/i18n/locales
DATABASE_URL     ?= postgres://postgres:postgres@localhost:5432/july?sslmode=disable

# ============================================
# Setup
# ============================================

setup: $(TAILWIND_BIN) $(HTMX_DEST) $(MERMAID_DEST) ## Setup project, install dev tools, and vendor assets
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/a-h/templ/cmd/templ@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/air-verse/air@latest
	go install gotest.tools/gotestsum@latest
	go mod tidy

$(TAILWIND_BIN):
	@mkdir -p bin
	@echo "→ Downloading Tailwind CSS v$(TAILWIND_VERSION) ($(TAILWIND_PLATFORM))..."
	@curl -fsSL $(TAILWIND_URL) -o $@
	@chmod +x $@
	@echo "✓ $(TAILWIND_BIN)"

$(HTMX_DEST):
	@mkdir -p web/assets
	@echo "→ Downloading HTMX v$(HTMX_VERSION)..."
	@curl -fsSL $(HTMX_URL) -o $@
	@echo "✓ $(HTMX_DEST)"

$(MERMAID_DEST):
	@mkdir -p web/assets
	@echo "→ Downloading Mermaid v$(MERMAID_VERSION)..."
	@curl -fsSL $(MERMAID_URL) -o $@
	@echo "✓ $(MERMAID_DEST)"

compose-up:  ## Run the shared services in docker
	docker compose up -d postgres adminer smee

compose-down:  ## Stop the docker services
	docker compose down

# ============================================
# Development
# ============================================

dev: ## Run the site locally in dev mode
	air -c .air.toml

run: generate  ## Run the site like in production
	go run cmd/server/main.go

compile: generate  ## Compile the source
	go build -o bin/july ./cmd/server

build: generate ## Build docker image
	docker build . -t $(image)

push:  ## Push docker image
	docker push $(image)

deploy:  ## Deploy to gcloud
	gcloud run deploy julython-go \
		--image $(image) \
		--platform managed \
		--region us-central1 \
		--allow-unauthenticated

clean:  ## Clean up build artifacts (keeps vendored assets)
	rm -rf tmp bin coverage.out
	rm -f web/assets/tailwind*.css
	rm -f internal/components/assets_gen.go

# ============================================
# Code Generation
# ============================================

generate: sqlc templ tailwind  ## Generate sql, templ, and asset files

sqlc:  ## Generate sqlc files
	sqlc generate

templ:  ## Generate templ files
	templ generate

tailwind:  ## Build Tailwind CSS and generate asset manifest
	go run ./cmd/assetgen

# ============================================
# Testing
# ============================================

test-clean:
	go clean -testcache

test:  ## Run the tests
	gotestsum --format short -- -race ./...

test-v: test-clean  ## Run the tests with verbose output
	gotestsum --format short-verbose -- -race ./...

test-dots:  ## Run the tests with dot output
	gotestsum --format dots -- -race ./...

test-cover:  ## Run tests and generate coverage report
	gotestsum --format testname -- -coverprofile=coverage.out -coverpkg=./internal/... ./...
	go tool cover -func=coverage.out | grep -v 100.0
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-watch:  ## Run the tests in watch mode
	gotestsum --watch --format testname

# ============================================
# Database Migrations
# ============================================

migrate-up:  ## Migrate the database up
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:  ## Migrate the database down
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-new:  ## New migration (make migrate-new name=description_here)
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-new name=description"; exit 1; fi
	migrate create -ext sql -dir migrations -format "20060102" $(name)

migrate-status:  ## Show the migration status
	migrate -path migrations -database "$(DATABASE_URL)" version

migrate-prod-up:  ## Migrate production up
	migrate -path migrations -database "$(PROD_DATABASE_URL)" up

migrate-prod-down:  ## Migrate production down
	migrate -path migrations -database "$(PROD_DATABASE_URL)" down 1

migrate-prod-status:  ## Show production status
	migrate -path migrations -database "$(PROD_DATABASE_URL)" version

# ============================================
# i18n
# ============================================

i18n:  ## Sync missing i18n keys to locale files
	go run ./cmd/i18nsync -locales=$(LOCALES_DIR)

# ============================================
# Help
# ============================================

help:  ## Help is on the way
	@echo "Julython golang application"
	@echo ""
	@echo "Available Commands:"
	@grep -h '^[a-zA-Z]' $(MAKEFILE_LIST) | awk -F ':.*?## ' 'NF==2 {printf "   %-20s%s\n", $$1, $$2}' | sort

.PHONY: help setup dev run build push deploy test clean generate sqlc templ tailwind \
        compose-up compose-down migrate-up migrate-down migrate-new migrate-status \
        migrate-prod-up migrate-prod-down migrate-prod-status i18n test-v test-dots \
        test-cover test-watch compile