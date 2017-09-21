SHELL := /bin/bash
COMPOSE_FLAGS := --project-name julython -f ./docker-compose.yml
PROD_FLAGS := --project-name julython -f ./docker-compose-prod.yml
COMPOSE_BUILD_SERVICES := python
COMPOSE_SERVICES := httpd-server django

build:
	@docker-compose $(COMPOSE_FLAGS) build --force-rm --pull $(COMPOSE_BUILD_SERVICES)

up:
	@docker-compose $(COMPOSE_FLAGS) up --abort-on-container-exit $(COMPOSE_SERVICES)

prod:
	@docker-compose $(PROD_FLAGS) up -d $(COMPOSE_SERVICES)

stop:
	@docker-compose $(COMPOSE_FLAGS) stop --timeout 0

clean:
	@docker-compose $(COMPOSE_FLAGS) down --volumes

clean-all:
	@docker-compose $(COMPOSE_FLAGS) down --volumes --rmi all
