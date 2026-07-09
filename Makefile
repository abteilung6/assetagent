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

.PHONY: build test clean dev dev-up dev-down dev-ps dev-logs migrate-up migrate-down migrate-status goose-install sqlc-install sqlc-generate api-install api-generate api-client-generate import serve ollama-pull ollama-logs console-install console-dev console-build console-test

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

dev: dev-up build
	@echo "Starting API (:8080) and console (:5173). Ctrl+C stops both."
	@trap 'kill 0' INT TERM; \
	./$(BINARY) serve & \
	cd console && npm run dev; \
	wait

console-install:
	cd console && npm install

console-dev:
	@curl -sf http://localhost:8080/api/health >/dev/null 2>&1 || \
		echo "Warning: API not reachable at :8080 — run 'make dev-up serve' or 'make dev' first"
	cd console && npm run dev

console-build: api-client-generate
	cd console && npm run build

console-test:
	cd console && npm test
