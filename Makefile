.PHONY: up down migrate-up migrate-down migrate-create sqlc run build ssh-test clean help

# Load .env if it exists
-include .env
export

BINARY_NAME := readit-server
DATABASE_URL ?= postgres://readit:readit_dev@localhost:5432/readit?sslmode=disable

## up: Start PostgreSQL via Docker Compose
up:
	docker compose up -d
	@echo "Waiting for PostgreSQL to become healthy..."
	@until docker compose exec postgres pg_isready -U readit -d readit > /dev/null 2>&1; do sleep 1; done
	@echo "PostgreSQL is ready."

## down: Stop Docker Compose services
down:
	docker compose down

## migrate-up: Run all pending Goose migrations
migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

## migrate-down: Roll back the last migration
migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

## migrate-create: Create a new migration file (usage: make migrate-create NAME=add_foo)
migrate-create:
	goose -dir migrations create $(NAME) sql

## sqlc: Regenerate Go code from SQL queries
sqlc:
	sqlc generate

## build: Compile the server binary
build:
	go build -o bin/$(BINARY_NAME) ./cmd/server

## run: Build and run the server
run: build
	./bin/$(BINARY_NAME)

## ssh-test: Connect to the local server via SSH
ssh-test:
	ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p 2222 localhost

## tui-screen: Capture tmux TUI session frame into PNG screenshot
tui-screen:
	@./scripts/tui-screen.sh $(or $(TARGET),dev) $(or $(OUT),/tmp/tui-screen.png)

## clean: Remove build artifacts
clean:
	rm -rf bin/

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
