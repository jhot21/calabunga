# Makefile
.PHONY: build test test-integration run

build:
	go build -o bin/calabunga ./cmd/server

test:
	go test ./...

test-integration:
	go test -tags integration ./integration/...

run:
	go run ./cmd/server
