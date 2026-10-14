VENV := .venv
MKDOCS := $(VENV)/bin/mkdocs
DB_URL := postgres://noosphere:noosphere@localhost:5432/noosphere?sslmode=disable
TEST_DB_URL := postgres://noosphere:noosphere@localhost:5432/noosphere_test?sslmode=disable
GOOSE := GOOSE_DRIVER=postgres GOOSE_MIGRATION_DIR=db/migrations go tool goose

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
	GOOSE_DBSTRING='$(DB_URL)' $(GOOSE) up

migrate-down:
	GOOSE_DBSTRING='$(DB_URL)'$(GOOSE) down

migrate-status:
	GOOSE_DBSTRING='$(DB_URL)'$(GOOSE) status

.PHONY: generate

generate:
	go tool sqlc generate

.PHONY: run

run:
	DATABASE_URL='$(DB_URL)' go run ./cmd/noosphere

.PHONY: test

test:
	GOOSE_DBSTRING='$(TEST_DB_URL)' $(GOOSE) up
	TEST_DATABASE_URL='$(TEST_DB_URL)' go test -count=1 ./...
