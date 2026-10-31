SHELL := pwsh.exe
.SHELLFLAGS := -NoProfile -Command

GO ?= go
NPM ?= npm
VERSION ?= dev
CONFIG ?=
ARGS ?=

FRONTEND_DIR := cmd/yawa-ui/frontend
BIN_DIR := bin
CONFIG_FLAG := $(if $(CONFIG),--config "$(CONFIG)",)

.DEFAULT_GOAL := help

.PHONY: help test vet check format tidy frontend-install frontend-build cli validate agent ui build build-cli build-ui clean

help: ## Show available development commands
	@Write-Output "Yawa development commands:"; Get-Content '$(MAKEFILE_LIST)' | Select-String '^[a-zA-Z_-]+:.*## ' | ForEach-Object { $$parts = $$_.Line -split ':.*## ', 2; '  {0,-20} {1}' -f $$parts[0], $$parts[1] }

test: ## Run all Go tests
	$(GO) test ./...

vet: ## Run Go static analysis
	$(GO) vet ./...

check: frontend-build test vet ## Build the frontend and run all automated checks

format: ## Format all Go source files
	$(GO) fmt ./...

tidy: ## Normalize Go module dependencies
	$(GO) mod tidy

frontend-install: ## Install pinned frontend dependencies
	Set-Location '$(FRONTEND_DIR)'; $(NPM) ci

frontend-build: ## Build the React frontend embedded by Wails
	Set-Location '$(FRONTEND_DIR)'; $(NPM) run build

cli: ## Run the CLI directly; example: make cli ARGS="list"
	$(GO) run ./cmd/yawa $(CONFIG_FLAG) $(ARGS)

validate: ## Validate the active config; optionally pass CONFIG=C:\path\config.toml
	$(GO) run ./cmd/yawa $(CONFIG_FLAG) validate

agent: ## Run the global hotkey agent directly from source
	$(GO) run ./cmd/yawa $(CONFIG_FLAG) agent

ui: frontend-build ## Run the Wails customization UI directly from source
	$(GO) run -tags production ./cmd/yawa-ui

build: build-cli build-ui ## Build development executables into bin/

build-cli: ## Build the CLI and agent into bin/yawa.exe
	New-Item -ItemType Directory -Force '$(BIN_DIR)' | Out-Null; $(GO) build -trimpath -ldflags '-X main.version=$(VERSION)' -o '$(BIN_DIR)/yawa.exe' ./cmd/yawa

build-ui: frontend-build ## Build the Wails UI into bin/yawa-ui.exe
	New-Item -ItemType Directory -Force '$(BIN_DIR)' | Out-Null; $(GO) build -tags production -trimpath -o '$(BIN_DIR)/yawa-ui.exe' ./cmd/yawa-ui

clean: ## Remove generated development executables and frontend output
	if (Test-Path -LiteralPath '$(BIN_DIR)') { Remove-Item -Recurse -Force -LiteralPath '$(BIN_DIR)' }; if (Test-Path -LiteralPath '$(FRONTEND_DIR)/dist') { Remove-Item -Recurse -Force -LiteralPath '$(FRONTEND_DIR)/dist' }
