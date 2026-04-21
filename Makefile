
.PHONY: help dev down purge-data build run test seed logs log vendor swagger \
	k8s-up k8s-down k8s-logs k8s-port-forward k8s-status k8s-image

# ── Defaults ─────────────────────────────────────────────────────
APP_NAME  := go-elastic-search
API_IMAGE := novaku/$(APP_NAME)
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

KUBECTL        ?= kubectl
K8S_NAMESPACE  ?= go-elastic
K8S_MANIFESTS  ?= k8s
K8S_API_IMAGE  ?= go-elastic-docker-api:latest

help: ## Show this help
	@awk 'BEGIN{FS=":.*##"} /^[a-zA-Z0-9_-]+:.*##/{printf "  \033[36m%-18s\033[0m %s\n",$$1,$$2}' $(MAKEFILE_LIST)

# ── Local dev (Docker Compose) ────────────────────────────────────
dev: swagger ## Start full local stack (ES + API)
	@mkdir -p data/elasticsearch
	docker compose up --build -d
	@printf "\n⏳ Waiting for API to be ready"
	@until curl -fsS http://localhost:8080/ready >/dev/null 2>&1; do \
		printf "."; \
		sleep 1; \
	done
	@echo ""
	@echo "\n✅ Stack running:"
	@echo "   API     → http://localhost:8080"
	@echo "   QA UI   → http://localhost:8080/qa/"
	@echo "   API Docs→ http://localhost:8080/docs/index.html"
	@open http://localhost:8080/qa/ >/dev/null 2>&1 || true
	@open http://localhost:8080/docs/index.html >/dev/null 2>&1 || true

down: ## Stop and remove containers (keep volumes/data)
	docker compose down

purge-data: ## Stop stack and remove all containers + volumes (DANGER: deletes ES data)
	docker compose down -v
	rm -rf data/elasticsearch
	mkdir -p data/elasticsearch

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

swagger: ## Generate Swagger/OpenAPI docs
	go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g main.go -d cmd/api,internal/handler,internal/router,internal/service,config -o docs --parseInternal

# ── Seed ─────────────────────────────────────────────────────────
SEED_URL      ?= http://localhost:8080
SEED_USERNAME ?= admin
SEED_PASSWORD ?= admin123

seed: ## Bulk-index sample products (requires jq + curl)
	@echo "Seeding products…"
	$(eval SEED_TOKEN := $(shell curl -s -X POST $(SEED_URL)/auth/login \
		-H "Content-Type: application/json" \
		-d '{"username":"$(SEED_USERNAME)","password":"$(SEED_PASSWORD)"}' \
		| jq -r '.token'))
	@if [ -z "$(SEED_TOKEN)" ] || [ "$(SEED_TOKEN)" = "null" ]; then \
		echo "❌ Login failed — check SEED_USERNAME / SEED_PASSWORD"; exit 1; \
	fi
	@curl -s -X POST $(SEED_URL)/v1/products/bulk \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer $(SEED_TOKEN)" \
		-d @scripts/seed.json | jq .

# ── Production ────────────────────────────────────────────────────
prod-up: ## Start production stack
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

prod-down: ## Stop production stack
	docker compose -f docker-compose.yml -f docker-compose.prod.yml down

# ── Kubernetes (local cluster) ───────────────────────────────────
k8s-image: ## Build API image for local Kubernetes runtime
	docker build -t $(K8S_API_IMAGE) .

k8s-up: ## Deploy Elasticsearch + API to Kubernetes
	@docker image inspect $(K8S_API_IMAGE) >/dev/null 2>&1 || (echo "❌ Image $(K8S_API_IMAGE) not found. Build first with: make k8s-image K8S_API_IMAGE=$(K8S_API_IMAGE)" && exit 1)
	@$(KUBECTL) get namespace $(K8S_NAMESPACE) >/dev/null 2>&1 || $(KUBECTL) create namespace $(K8S_NAMESPACE)
	$(KUBECTL) apply -n $(K8S_NAMESPACE) -f $(K8S_MANIFESTS)/elasticsearch.yaml
	$(KUBECTL) apply -n $(K8S_NAMESPACE) -f $(K8S_MANIFESTS)/api.yaml
	$(KUBECTL) set image deployment/api api=$(K8S_API_IMAGE) -n $(K8S_NAMESPACE)
	$(KUBECTL) rollout restart deployment/api -n $(K8S_NAMESPACE)
	$(KUBECTL) rollout status deployment/elasticsearch -n $(K8S_NAMESPACE) --timeout=180s
	$(KUBECTL) rollout status deployment/api -n $(K8S_NAMESPACE) --timeout=180s
	@echo ""
	@echo "✅ Kubernetes stack running in namespace: $(K8S_NAMESPACE)"
	@echo "   Run: make k8s-port-forward"

k8s-down: ## Remove Kubernetes resources
	-$(KUBECTL) delete -n $(K8S_NAMESPACE) -f $(K8S_MANIFESTS)/api.yaml --ignore-not-found
	-$(KUBECTL) delete -n $(K8S_NAMESPACE) -f $(K8S_MANIFESTS)/elasticsearch.yaml --ignore-not-found

k8s-status: ## Show Kubernetes pods/services status
	$(KUBECTL) get all -n $(K8S_NAMESPACE)

k8s-logs: ## Tail API logs from Kubernetes
	$(KUBECTL) logs -n $(K8S_NAMESPACE) deploy/api -f

k8s-port-forward: ## Expose API on localhost:8080
	$(KUBECTL) port-forward -n $(K8S_NAMESPACE) service/api 8080:8080
