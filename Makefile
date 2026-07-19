.PHONY: help run build tidy sqlc css css-watch migrate-up migrate-down migrate-create tools create-admin

# --- config ---
GO            ?= go
BIN           ?= bin/classyfm
TAILWIND      ?= ./bin/tailwindcss
MIGRATIONS    ?= internal/db/migrations
DATABASE_DSN  ?=

# Load .env if present so run/create-admin/migrate-* pick up local config
ifneq (,$(wildcard .env))
include .env
export
endif

help: ## List targets
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n",$$1,$$2}'

run: ## Run the server
	$(GO) run ./cmd/server

build: css ## Build the binary (embeds templates + static)
	$(GO) build -o $(BIN) ./cmd/server

tidy: ## Tidy go modules
	$(GO) mod tidy

sqlc: ## Generate typed DB code from SQL
	sqlc generate

css: ## Build Tailwind CSS once (minified)
	$(TAILWIND) -i web/static/css/tailwind.css -o web/static/css/app.css --minify

css-watch: ## Rebuild Tailwind CSS on change
	$(TAILWIND) -i web/static/css/tailwind.css -o web/static/css/app.css --watch

# --- migrations (requires golang-migrate: https://github.com/golang-migrate/migrate) ---
migrate-up: ## Apply all up migrations (needs DATABASE_URL, e.g. mysql://user:pass@tcp(host:3306)/classyfm)
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" up

migrate-down: ## Roll back one migration
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" down 1

migrate-create: ## Create a new migration: make migrate-create name=add_x
	migrate create -ext sql -dir $(MIGRATIONS) -seq $(name)

tools: ## Install dev tools (sqlc)
	$(GO) install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

create-admin: ## Bootstrap an admin user: make create-admin email=you@example.com password=secret name="Full Name"
	$(GO) run ./cmd/createadmin -email "$(email)" -password "$(password)" -name "$(name)"
