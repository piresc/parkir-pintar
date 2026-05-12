# Assessment Boilerplate Makefile

APP_NAME := parkir-pintar
BINARY := ./bin/gateway
DOCKER_IMAGE := $(APP_NAME)
VERSION ?= dev
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildTime=$(BUILD_TIME)

.PHONY: help test test-coverage test-race test-race-e2e generate-mocks lint gosec gitleaks build run docker-build docker-run clean proto-gen

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

test: ## Run all tests
	go test ./...

test-coverage: ## Run tests with coverage report
	go test ./... -coverprofile=coverage.txt -covermode=atomic
	go tool cover -func=coverage.txt

test-race: ## Run tests with race detector
	go test ./... -race

generate-mocks: ## Generate mocks via go generate
	go generate ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

gosec: ## Run security scanner (gosec)
	gosec -exclude=G401,G304,G501,G505 -fmt=sonarqube -out=sonar-gosec.json ./...

gitleaks: ## Run secret scanning (gitleaks)
	gitleaks detect --source=. --config=.gitleaks.toml --verbose

build: ## Build the binary
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/gateway

run: build ## Build and run the binary
	$(BINARY)

docker-build: ## Build Docker image
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_IMAGE):$(VERSION) .

docker-run: ## Run via Docker Compose
	docker compose up -d

proto-gen: ## Generate Go code from proto files
	protoc \
		-I proto/ \
		-I /usr/local/include \
		--go_out=. --go_opt=module=parkir-pintar \
		--go-grpc_out=. --go-grpc_opt=module=parkir-pintar \
		proto/search/v1/search.proto \
		proto/reservation/v1/reservation.proto \
		proto/billing/v1/billing.proto \
		proto/payment/v1/payment.proto \
		proto/presence/v1/presence.proto \
		proto/notification/v1/notification.proto

clean: ## Remove build artifacts
	rm -rf ./bin coverage.txt sonar-gosec.json

test-e2e: ## Run Layer 1 E2E tests (testcontainers-go)
	go test -v -timeout 300s ./tests/e2e/...

test-e2e-docker: ## Run Layer 2 E2E tests (Docker Compose)
	go test -v -timeout 600s ./tests/e2e_docker/...

test-e2e-all: test-e2e test-e2e-docker ## Run both E2E test layers

test-race-e2e: ## Run race condition E2E tests with -race detector
	go test -race -v -timeout 300s -count=1 -run TestRace ./tests/e2e/...

test-load: ## Run load/stress tests
	go test -v -timeout 600s -count=1 -run TestLoad ./tests/e2e/...
