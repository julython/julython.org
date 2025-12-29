REQUIREMENTS		:= pyproject.toml
SHELL				:= /bin/bash
VIRTUAL_ENV			:= .venv
PYTHON				:= $(shell command -v python3 || command -v python)
app_name 			:= july
git_sha				:= $(shell git rev-parse --short=8 HEAD)
image				:= gcr.io/julython/julython:$(git_sha)

# Setup the env
export PYTHONPATH = ./

# PHONY just means this target does not make any files
.PHONY: setup clean test help

default: help

# Make sure the virtualenv exists, create it if not.
$(VIRTUAL_ENV):
	uv sync

# Check for the existence/timestamp of .reqs-installed if the
# file is missing or older than the pyproject.toml this will run pip
$(VIRTUAL_ENV)/.reqs-installed: $(REQUIREMENTS)
	uv sync
	touch $(VIRTUAL_ENV)/.reqs-installed

.env:
	cp dotenv .env

setup: $(VIRTUAL_ENV) $(VIRTUAL_ENV)/.reqs-installed ## Setup local environment

deps: ## Make sure the docker deps are running
	docker compose up -d postgres adminer

up: dev
dev: ## Run the service in docker
	docker compose up api --build

dev-ui: ## Run the ui in dev mode
	pushd ui && npm i && npm run dev

test: deps ## Run pytests
	$(VIRTUAL_ENV)/bin/pytest

test-failed: deps ## Run pytests
	$(VIRTUAL_ENV)/bin/pytest --lf

build:  ## Build the docker image
	docker build . -t $(image)

push:  ## Push the docker image
	docker push $(image)

deploy: ## deploy to gcloud
	gcloud run deploy julython \
		--image $(image) \
		--platform managed \
		--region us-central1 \
		--allow-unauthenticated

db_create: deps ## Create the initial database
	$(VIRTUAL_ENV)/bin/python -m $(app_name) db create

migrate: deps  ## Upgrade the database by applying migrations
	$(VIRTUAL_ENV)/bin/python -m $(app_name) db upgrade

upgrade: migrate  ## Alias: Upgrade the database by applying migrations

downgrade: deps ## Downgrade the database to the previous revision
	$(VIRTUAL_ENV)/bin/python -m $(app_name) db downgrade

prod-create:  ## Create DB in prod
	docker compose run prod db create

prod-upgrade:  ## Upgrade DB in prod
	docker compose run prod db upgrade

prod-downgrade: ## Downgrade DB in prod
	docker compose run prod db downgrade


revision: deps ## Generate a new database migration script  (`make revision message="Message"`)
	$(VIRTUAL_ENV)/bin/python -m $(app_name) db revision -m "$(message)"


help:
	@@grep -h '^[a-zA-Z]' $(MAKEFILE_LIST) | awk -F ':.*?## ' 'NF==2 {printf "   %-20s%s\n", $$1, $$2}' | sort


.PHONY: help up dev test
