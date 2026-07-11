.DEFAULT_GOAL := build

# Where sqlc/swag write generated code — kept as vars so a path change
# only needs updating in one place.
SQLC_OUT   := internal/db/generated
DOCS_OUT   := docs
BINARY_OUT := bin/server
MAIN_PKG   := ./cmd/server

.PHONY: tools tidy sqlc swag generate goimports vet staticcheck lint test build run migrate-up migrate-down clean ci

# --- one-time setup ---

# Installs every code-generation/lint tool this Makefile calls, in
# case a fresh clone (or a fresh machine) doesn't have them yet. Not a
# dependency of any other target — run it manually once.
tools:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install golang.org/x/tools/cmd/goimports@latest

tidy:
	go mod tidy

# --- code generation ---
# main.go imports both generated packages (db/generated, docs) — build
# will fail with BrokenImport errors if either hasn't been generated
# yet, so `generate` runs before anything that compiles the module.

sqlc:
	sqlc generate

swag:
	swag init -g cmd/server/main.go -o $(DOCS_OUT)

generate: sqlc swag

# --- format / vet / lint ---

goimports:
	goimports -w .

vet: generate goimports
	go vet ./...

# staticcheck.conf enables the full check set, including ST1000/1020/
# 1021/1022 — the closest automated check for "every exported symbol
# has a doc comment starting with its name" from go-docs-instructions.md.
# It won't catch a comment that exists but doesn't explain *why* (that
# still needs a human read), but it will catch one that's missing
# entirely or misnamed.
#
# internal/db/generated is excluded: it's sqlc output, never hand-
# edited, and `sqlc generate` wipes any doc comment added to satisfy
# the linter on the next run anyway — flagging it produces noise with
# no fix that survives regeneration.
staticcheck: vet
	staticcheck $$(go list ./... | grep -v /internal/db/generated)

lint: staticcheck

# --- test / build ---

test: generate
	go test ./... -v

build: staticcheck test
	go build -o $(BINARY_OUT) $(MAIN_PKG)

run: build
	./$(BINARY_OUT)

# --- database ---
# Both assume DATABASE_URL is already in your shell env (`set -a;
# source .env; set +a` first) — see DECISIONS.md for why `export
# $(grep ...)` doesn't work with quoted/spaced values like
# RESEND_FROM_ADDRESS.

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

# --- housekeeping ---

clean:
	rm -rf $(BINARY_OUT) $(DOCS_OUT) $(SQLC_OUT)

# Full clean-room pipeline — what CI (or you, before a defense/demo)
# should run: nothing here trusts anything cached or previously
# generated.
ci: clean tidy generate goimports vet staticcheck test build