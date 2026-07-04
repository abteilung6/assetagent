BINARY := bin/assetagent
CMD := ./cmd/assetagent

.PHONY: build test clean

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./... -count=1

clean:
	rm -rf bin/
