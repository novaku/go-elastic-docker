.PHONY: help dev down build run test seed logs log vendor

# ── Defaults ─────────────────────────────────────────────────────
APP_NAME  := go-elastic-search
API_IMAGE := yourorg/$(APP_NAME)
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

help: ## Show this help
	@awk 'BEGIN{FS=":.*##"} /^[a-zA-Z_-]+:.*##/{printf "  \033[36m%-18s\033[0m %s\n",$$1,$$2}' $(MAKEFILE_LIST)

# ── Local dev (Docker Compose) ────────────────────────────────────
dev: ## Start full local stack (ES + API)
	docker compose up --build -d
	@echo "\n✅ Stack running:"
	@echo "   API     → http://localhost:8080"

down: ## Stop and remove all containers + volumes
	docker compose down -v

logs log: ## Tail API logs
	docker compose logs -f api

restart-api: ## Rebuild & restart only the API container
	docker compose up --build -d --no-deps api

# ── Build ─────────────────────────────────────────────────────────
build: ## Build the Go binary locally
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/$(APP_NAME) ./cmd/api

image: ## Build Docker image and tag with version
	docker build -t $(API_IMAGE):$(VERSION) -t $(API_IMAGE):latest .

push: image ## Push image to registry
	docker push $(API_IMAGE):$(VERSION)
	docker push $(API_IMAGE):latest

# ── Run locally (no Docker) ───────────────────────────────────────
run: ## Run the API server locally (needs ES running separately)
	@cp -n .env.example .env 2>/dev/null || true
	go run ./cmd/api

# ── Tests ─────────────────────────────────────────────────────────
test: ## Run all tests
	go test -v -race ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

vendor: ## Download and vendor all dependencies
	go mod tidy && go mod vendor

# ── Seed ─────────────────────────────────────────────────────────
seed: ## Bulk-index sample products (requires jq + curl)
	@echo "Seeding products…"
	@curl -s -X POST http://localhost:8080/v1/products/bulk \
		-H "Content-Type: application/json" \
		-d @scripts/seed.json | jq .

# ── Production ────────────────────────────────────────────────────
prod-up: ## Start production stack
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

prod-down: ## Stop production stack
	docker compose -f docker-compose.yml -f docker-compose.prod.yml down
