.DEFAULT_GOAL := help

APP_NAME    ?= yolo-api
IMAGE_REPO  ?= melvinjosec/yolo-api
IMAGE_TAG   ?= latest
PORT        ?= 8080

.PHONY: help
help: ## Show this help message
	@echo "YOLO Cloud-Native DevOps Automation Tooling"
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Compile Go backend binary
	cd app/backend && CGO_ENABLED=0 go build -ldflags="-w -s" -o ../../bin/$(APP_NAME) .

.PHONY: test
test: ## Run unit and integration tests with race detector and coverage
	cd app/backend && go test -v -race -coverprofile=coverage.out ./...

.PHONY: lint
lint: ## Run golangci-lint
	cd app/backend && golangci-lint run --timeout=5m

.PHONY: docker-build
docker-build: ## Build Docker container image locally
	docker build -t $(IMAGE_REPO):$(IMAGE_TAG) -f app/backend/Dockerfile app/backend

.PHONY: docker-up
docker-up: ## Start entire stack via Docker Compose
	docker compose up -d --build

.PHONY: docker-down
docker-down: ## Stop and remove Docker Compose services and volumes
	docker compose down -v

.PHONY: helm-lint
helm-lint: ## Lint Helm chart syntax and parameters
	helm lint deploy/helm/yolo-app

.PHONY: helm-template
helm-template: ## Dry-run render Helm templates for inspection
	helm template yolo-app deploy/helm/yolo-app

.PHONY: k8s-apply
k8s-apply: ## Apply raw Kubernetes manifests to cluster
	kubectl apply -f deploy/k8s/namespace.yaml
	kubectl apply -f deploy/k8s/

.PHONY: security-scan
security-scan: ## Scan local Docker image with Trivy for vulnerabilities
	trivy image --severity HIGH,CRITICAL $(IMAGE_REPO):$(IMAGE_TAG)

.PHONY: clean
clean: ## Clean built binaries and temporary artifacts
	rm -rf bin/ app/backend/coverage.out
