SHELL := /bin/bash

GO ?= go
BIN_DIR ?= bin
BIN ?= $(BIN_DIR)/mayday-server
PKG ?= ./...
DB_URL ?= postgres://mayday:mayday@localhost:5432/mayday?sslmode=disable

.PHONY: run dev build test fmt vet lint migrate-up migrate-down docker-up docker-down clean tidy

run:
	$(GO) run ./cmd/server

dev:
	$(GO) run ./cmd/server

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN) ./cmd/server

test:
	$(GO) test -count=1 -race $(PKG)

fmt:
	$(GO) fmt $(PKG)

vet:
	$(GO) vet $(PKG)

# golangci-lint is optional. If it isn't installed, the target prints
# guidance instead of failing.
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. See https://golangci-lint.run/ for install instructions."; \
	fi

migrate-up:
	@command -v goose >/dev/null 2>&1 || { echo "goose not installed. Install with: go install github.com/pressly/goose/v3/cmd/goose@latest"; exit 1; }
	goose -dir migrations postgres "$(DB_URL)" up

migrate-down:
	@command -v goose >/dev/null 2>&1 || { echo "goose not installed. Install with: go install github.com/pressly/goose/v3/cmd/goose@latest"; exit 1; }
	goose -dir migrations postgres "$(DB_URL)" down

docker-up:
	docker compose up --build

docker-down:
	docker compose down

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BIN_DIR)
