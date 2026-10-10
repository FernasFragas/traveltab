APP_NAME := traveltab
BIN_DIR  := bin
BINARY   := $(BIN_DIR)/$(APP_NAME)

# go-sqlite3 needs cgo
export CGO_ENABLED := 1

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-8s\033[0m %s\n", $$1, $$2}'

.PHONY: fmt
fmt: ## Format all Go files
	gofmt -w .

.PHONY: lint
lint: ## Check formatting and run golangci-lint
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "Files need gofmt (run 'make fmt'):"; echo "$$unformatted"; exit 1; fi
	@command -v golangci-lint >/dev/null || { echo "golangci-lint is not installed: https://golangci-lint.run/welcome/install/"; exit 1; }
	golangci-lint run ./...

.PHONY: assets
assets: ## Install browser libraries from package-lock.json into public/vendor
	npm ci --ignore-scripts
	npm run vendor

.PHONY: test
test: assets ## Run all tests with the race detector (loads .env)
	@# Tests run from each package folder, so the app can't find .env on its own.
	@if [ -f .env ]; then set -a; . ./.env; set +a; fi; \
	go test -race -count=1 ./...

.PHONY: build
build: ## Build the web app into bin/
	go build -o $(BINARY) ./cmd/web

.PHONY: run
run: assets ## Run the web app locally on :8080 (reads .env)
	go run ./cmd/web

.PHONY: clean
clean: ## Remove build output
	rm -rf $(BIN_DIR)
