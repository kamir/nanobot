SHELL := /bin/bash

.PHONY: help build run rerun test test-classification release release-major release-minor release-patch docker-build docker-up docker-down docker-logs

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2} END {printf "\n"}' $(MAKEFILE_LIST)

test: ## Run all tests
	go test ./...

test-classification: ## Run internal/external message classification E2E test (verbose)
	go test -v -run "TestInternalExternalClassificationE2E|TestMessageTypeAccessorDefaults|TestPolicyTierGatingByMessageType" ./internal/agent/

build: ## Build the gomikrobot binary
	go build ./cmd/gomikrobot

run: build ## Build and run the gateway
	./gomikrobot gateway

rerun: ## Restart the gateway (kill ports 18790/18791)
	@set -euo pipefail; \
	pids=""; \
	for port in 18790 18791; do \
	  pid=$$(lsof -ti tcp:$$port -sTCP:LISTEN || true); \
	  if [[ -n "$$pid" ]]; then \
	    pids="$$pids $$pid"; \
	  fi; \
	done; \
	if [[ -n "$$pids" ]]; then \
	  echo "Killing existing gateway PIDs:$$pids"; \
	  kill $$pids; \
	  sleep 0.5; \
	fi; \
	$(MAKE) build; \
	./gomikrobot gateway

release: release-major ## Bump major version and build

release-major: ## Bump major version and build
	@bash ./scripts/release.sh major
	$(MAKE) build

release-minor: ## Bump minor version and build
	@bash ./scripts/release.sh minor
	$(MAKE) build

release-patch: ## Bump patch version and build
	@bash ./scripts/release.sh patch
	$(MAKE) build

docker-build: build ## Build local Docker image (gomikrobot:local)
	docker build -t gomikrobot:local -f Dockerfile .

docker-up: docker-build ## Start docker-compose using local image only
	SYSTEM_REPO_PATH=$${SYSTEM_REPO_PATH:-$$(pwd)} WORK_REPO_PATH=$${WORK_REPO_PATH:-$${HOME}/GoMikroBot-Workspace} docker compose -f docker-compose.yml up -d --no-build

docker-down: ## Stop docker-compose
	docker compose -f docker-compose.yml down

docker-logs: ## Tail docker-compose logs
	docker compose -f docker-compose.yml logs -f
