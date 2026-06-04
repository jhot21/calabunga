# Makefile
.PHONY: build web test test-integration run dev

web:
	cd web && npm install && npm run build

build: web
	go build -o bin/calabunga ./cmd/server

test:
	go test ./...

test-integration:
	go test -tags integration ./integration/...

run: build
	./bin/calabunga

dev:
	go run ./cmd/server &
	cd web && npm run dev
