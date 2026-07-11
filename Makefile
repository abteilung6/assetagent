BINARY := bin/assetagent
CMD := ./cmd/assetagent

ifneq (,$(wildcard .env))
include .env
export
endif

GOPATH_BIN := $(shell go env GOPATH)/bin
SQLC := $(GOPATH_BIN)/sqlc
OAPICODEGEN := $(GOPATH_BIN)/oapi-codegen

OLLAMA_MODEL ?= llama3.2
OLLAMA_BASE_URL ?= http://localhost:11434

.PHONY: build test clean dev-up dev-down dev-ps dev-logs migrate-up migrate-down migrate-status goose-install sqlc-install sqlc-generate api-install api-generate api-client-generate import serve ollama-pull ollama-logs console-install console-dev console-build console-test console-e2e-deps console-e2e-run console-e2e langfuse-up langfuse-down langfuse-ps langfuse-logs langfuse-health langfuse-reset

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./... -count=1

clean:
	rm -rf bin/

dev-up:
	docker compose up -d

dev-down:
	docker compose down

dev-ps:
	docker compose ps

dev-logs:
	docker compose logs -f postgres

ollama-logs:
	docker compose logs -f ollama

ollama-pull:
	curl -fsS "$(OLLAMA_BASE_URL)/api/pull" -d '{"name":"$(OLLAMA_MODEL)"}'

migrate-up: build
	./$(BINARY) migrate up

migrate-down: build
	./$(BINARY) migrate down

migrate-status: build
	./$(BINARY) migrate status

goose-install:
	go install github.com/pressly/goose/v3/cmd/goose@latest

sqlc-install:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

sqlc-generate:
	@test -x "$(SQLC)" || (echo "sqlc not found. Run: make sqlc-install" && exit 1)
	$(SQLC) generate

api-install:
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

api-generate:
	@test -x "$(OAPICODEGEN)" || (echo "oapi-codegen not found. Run: make api-install" && exit 1)
	$(OAPICODEGEN) -config oapi-codegen.yaml api/openapi.yaml

api-client-generate:
	cd console && npm run api:generate

import: build
	./$(BINARY) import $(FILE)

serve: build
	./$(BINARY) serve

console-install:
	cd console && npm install

console-dev:
	cd console && npm run dev

console-build: api-client-generate
	cd console && npm run build

console-test:
	cd console && npm test

PLAYWRIGHT_IMAGE ?= mcr.microsoft.com/playwright:v1.61.1-noble

console-e2e-deps:
	@echo "One-time host setup for local Playwright runs (requires sudo):"
	@echo "  cd console && sudo npx playwright install-deps chromium"
	cd console && npx playwright install chromium

console-e2e-run:
	docker run --rm --network host \
		-v "$(CURDIR):/repo" \
		-w /repo/console \
		-e CI=1 \
		$(PLAYWRIGHT_IMAGE) \
		npm run e2e

console-e2e: build dev-up console-install
	$(MAKE) console-e2e-run

LANGFUSE_COMPOSE := env -u DATABASE_URL -u DIRECT_URL docker compose -f docker-compose.langfuse.yml --env-file .env.langfuse

langfuse-up:
	@test -f .env.langfuse || (cp .env.langfuse.example .env.langfuse && echo "Created .env.langfuse from example")
	$(LANGFUSE_COMPOSE) up -d

langfuse-down:
	$(LANGFUSE_COMPOSE) down

langfuse-ps:
	$(LANGFUSE_COMPOSE) ps

langfuse-logs:
	$(LANGFUSE_COMPOSE) logs -f langfuse-web

langfuse-health:
	@curl -fsS http://localhost:3000/api/public/health && echo

langfuse-reset:
	$(LANGFUSE_COMPOSE) down -v
	@echo "Langfuse volumes removed. Run: make langfuse-up"
