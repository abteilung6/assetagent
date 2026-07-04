BINARY := bin/assetagent
CMD := ./cmd/assetagent

.PHONY: build test clean dev-up dev-down dev-ps dev-logs

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
