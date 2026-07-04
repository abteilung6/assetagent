BINARY := bin/assetagent
CMD := ./cmd/assetagent

ifneq (,$(wildcard .env))
include .env
export
endif

GOPATH_BIN := $(shell go env GOPATH)/bin
SQLC := $(GOPATH_BIN)/sqlc

.PHONY: build test clean dev-up dev-down dev-ps dev-logs migrate-up migrate-down migrate-status goose-install sqlc-install sqlc-generate

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
