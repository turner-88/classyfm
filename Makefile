.PHONY: help run build tidy sqlc css css-watch pdf require-db migrate-up migrate-down migrate-create tools create-admin

# --- config ---
GO            ?= go
BIN           ?= bin/classyfm
TAILWIND      ?= ./bin/tailwindcss
MIGRATIONS    ?= internal/db/migrations
DATABASE_DSN  ?=

# golang-migrate wants a scheme-prefixed URL, while the server wants a go-sql-driver
# DSN. Derive the former from the latter: prepend mysql:// and drop the ?query string
# (parseTime/charset/loc matter to the app, not to DDL). Override by exporting
# DATABASE_URL yourself if the DSN's password contains a '?' or a space.
DATABASE_URL  ?= mysql://$(firstword $(subst ?, ,$(DATABASE_DSN)))

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

pdf: ## Render a Markdown doc to a branded PDF: make pdf FILE=docs/API.md [OUT=docs/API.pdf]
	@test -n "$(FILE)" || { echo "usage: make pdf FILE=docs/API.md [OUT=docs/API.pdf]"; exit 1; }
	$(GO) run ./cmd/mdpdf $(FILE) $(OUT)

# --- migrations (requires golang-migrate: https://github.com/golang-migrate/migrate) ---
require-db:
	@test -n "$(DATABASE_DSN)" || { echo "DATABASE_DSN is not set — copy .env.example to .env"; exit 1; }

migrate-up: require-db ## Apply all up migrations (uses DATABASE_DSN from .env)
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" up

migrate-down: require-db ## Roll back one migration
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" down 1

migrate-create: ## Create a new migration: make migrate-create name=add_x
	migrate create -ext sql -dir $(MIGRATIONS) -seq $(name)

tools: ## Install dev tools (sqlc)
	$(GO) install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

create-admin: ## Bootstrap an admin user: make create-admin email=you@example.com password=secret name="Full Name"
	$(GO) run ./cmd/createadmin -email "$(email)" -password "$(password)" -name "$(name)"

import-podcasts: ## Import legacy podcasts from backup-old-db.sql (dry-run by default; pass WRITE=1 to write)
	$(GO) run ./cmd/importpodcasts -dry-run=$(if $(WRITE),false,true)
