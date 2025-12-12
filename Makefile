REQUIREMENTS		:= pyproject.toml
SHELL				:= /bin/bash
VIRTUAL_ENV			:= .venv
PYTHON				:= $(shell command -v python3 || command -v python)
app_name 			:= july

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
	docker compose up api

test: deps ## Run pytests
	$(VIRTUAL_ENV)/bin/pytest

test-failed: deps ## Run pytests
	$(VIRTUAL_ENV)/bin/pytest --lf

db_create: deps ## Create the initial database
	$(VIRTUAL_ENV)/bin/python -m $(app_name) db create

migrate: deps  ## Upgrade the database by applying migrations
	$(VIRTUAL_ENV)/bin/python -m $(app_name) db upgrade

upgrade: migrate  ## Alias: Upgrade the database by applying migrations

downgrade: deps ## Downgrade the database to the previous revision
	$(VIRTUAL_ENV)/bin/python -m $(app_name) db downgrade

revision: deps ## Generate a new database migration script  (`make revision message="Message"`)
	$(VIRTUAL_ENV)/bin/python -m $(app_name) db revision -m "$(message)"


help:
	@@grep -h '^[a-zA-Z]' $(MAKEFILE_LIST) | awk -F ':.*?## ' 'NF==2 {printf "   %-20s%s\n", $$1, $$2}' | sort


.PHONY: help up dev test
