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
	GOOSE_DBSTRING='$(DB_URL)' $(GOOSE) down

migrate-status:
	GOOSE_DBSTRING='$(DB_URL)' $(GOOSE) status

.PHONY: generate

generate:
	go tool sqlc generate

.PHONY: run

run:
	DATABASE_URL='$(DB_URL)' go run ./cmd/noosphere

.PHONY: test test-v test-migrate

test-migrate:
	GOOSE_DBSTRING='$(TEST_DB_URL)' $(GOOSE) up

test: test-migrate
	TEST_DATABASE_URL='$(TEST_DB_URL)' go test -count=1 ./...

test-v: test-migrate
	TEST_DATABASE_URL='$(TEST_DB_URL)' go tool ginkgo -r -v --randomize-all $(ARGS)

.PHONY: fmt fmt-check vet check

fmt:
	go fmt ./...

fmt-check:
	@out=$$(gofmt -l $$(go list -f '{{.Dir}}' ./...)); \
	if [ -n "$$out" ]; then echo "not formatted (run make fmt):"; echo "$$out"; exit 1; fi

vet:
	go vet ./...

check: fmt-check vet test
