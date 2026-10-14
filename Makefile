VENV := .venv
MKDOCS := $(VENV)/bin/mkdocs
DB_URL := postgres://noosphere:noosphere@localhost:5432/noosphere?sslmode=disable
GOOSE := GOOSE_DRIVER=postgres GOOSE_DBSTRING='$(DB_URL)' GOOSE_MIGRATION_DIR=db/migrations go tool goose

.DEFAULT_GOAL := docs
.PHONY: venv docs docs-build clean

venv: $(MKDOCS)

$(MKDOCS): requirements.txt
	python3 -m venv $(VENV)
	$(VENV)/bin/pip install -q --upgrade pip
	$(VENV)/bin/pip install -q -r requirements.txt
	@touch $@

docs: $(MKDOCS)
	$(MKDOCS) serve

docs-build: $(MKDOCS)
	$(MKDOCS) build --strict

clean:
	rm -rf site

.PHONY: db-up db-down db-psql

db-up:
	docker compose up -d --wait

db-down:
	docker compose down

db-psql:
	docker compose exec db psql -U noosphere

.PHONY: migrate-up migrate-down migrate-status

migrate-up:
	$(GOOSE) up

migrate-down:
	$(GOOSE) down

migrate-status:
	$(GOOSE) status

.PHONY: generate

generate:
	go tool sqlc generate
