# ParkirPintar Makefile
# Smart Parking Reservation System - Developer Experience

# Go binary path
GO := /usr/local/go/bin/go
GOFLAGS := -v
MODULE := github.com/parkir-pintar/parkir-pintar

# Tool binaries
GOLANGCI_LINT := $(shell which golangci-lint 2>/dev/null || echo "golangci-lint")
GOSEC := $(shell which gosec 2>/dev/null || echo "gosec")
GOVULNCHECK := $(shell which govulncheck 2>/dev/null || echo "govulncheck")
PROTOC := $(shell which protoc 2>/dev/null || echo "protoc")
K6 := $(shell which k6 2>/dev/null || echo "k6")

# Docker
DOCKER_COMPOSE := docker compose
COMPOSE_FILE := docker-compose.yml

# Migration
MIGRATE := $(shell which migrate 2>/dev/null || echo "migrate")
MIGRATION_DIR := db/migrations

# Coverage
COVERAGE_DIR := .coverage
COVERAGE_FILE := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML := $(COVERAGE_DIR)/coverage.html

# Services
SERVICES := reservation billing payment presence search gateway analytics

# Build info
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

.PHONY: help build test test-coverage bench lint security proto \
        docker-up docker-down migrate-up migrate-down load-test ci \
        clean tools fmt vet

## help: Show this help message
help:
	@echo "ParkirPintar - Smart Parking Reservation System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /' | sort
	@echo ""
	@echo "Services: $(SERVICES)"

## build: Build all service binaries
build:
	@echo "==> Building all services..."
	@for svc in $(SERVICES); do \
		echo "  Building $$svc..."; \
		$(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$$svc ./cmd/$$svc; \
	done
	@echo "==> Build complete. Binaries in ./bin/"

## build-service: Build a single service (usage: make build-service SVC=reservation)
build-service:
	@if [ -z "$(SVC)" ]; then echo "Error: SVC not set. Usage: make build-service SVC=reservation"; exit 1; fi
	@echo "==> Building $(SVC)..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(SVC) ./cmd/$(SVC)

## test: Run all tests with race detector (short mode)
test:
	@echo "==> Running tests..."
	$(GO) test -race -short -count=1 ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "==> Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	$(GO) tool cover -func=$(COVERAGE_FILE) | tail -1
	@echo "==> Coverage report: $(COVERAGE_HTML)"

## test-integration: Run integration tests (requires docker services)
test-integration:
	@echo "==> Running integration tests..."
	$(GO) test -race -count=1 -tags=integration ./...

## bench: Run benchmarks
bench:
	@echo "==> Running benchmarks..."
	$(GO) test -bench=. -benchmem -run=^$$ ./...

## bench-service: Run benchmarks for a single service (usage: make bench-service SVC=reservation)
bench-service:
	@if [ -z "$(SVC)" ]; then echo "Error: SVC not set. Usage: make bench-service SVC=reservation"; exit 1; fi
	$(GO) test -bench=. -benchmem -run=^$$ ./internal/$(SVC)/...

## lint: Run golangci-lint
lint:
	@echo "==> Running linter..."
	$(GOLANGCI_LINT) run ./...

## fmt: Format code
fmt:
	@echo "==> Formatting code..."
	$(GO) fmt ./...
	@echo "==> Checking for unformatted files..."
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:"; gofmt -l .; exit 1)

## vet: Run go vet
vet:
	@echo "==> Running go vet..."
	$(GO) vet ./...

## security: Run security scanners (gosec + govulncheck)
security:
	@echo "==> Running gosec..."
	$(GOSEC) -quiet ./...
	@echo ""
	@echo "==> Running govulncheck..."
	$(GOVULNCHECK) ./...
	@echo ""
	@echo "==> Security scan complete."

## proto: Generate protobuf/gRPC code
proto:
	@echo "==> Generating protobuf code..."
	@buf generate
	@echo "==> Proto generation complete."

## docker-up: Start all services with docker compose
docker-up:
	@echo "==> Starting docker compose services..."
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) up -d
	@echo "==> Waiting for services to be healthy..."
	@sleep 5
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) ps
	@echo ""
	@echo "Services:"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  Redis:      localhost:6379"
	@echo "  NATS:       localhost:4222 (monitoring: localhost:8222)"

## docker-down: Stop all docker compose services
docker-down:
	@echo "==> Stopping docker compose services..."
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) down -v

## docker-logs: Tail docker compose logs
docker-logs:
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) logs -f

## migrate-up: Run database migrations (all services)
migrate-up:
	@echo "==> Running migrations..."
	@for svc in $(SERVICES); do \
		if [ -d "$(MIGRATION_DIR)/$$svc" ]; then \
			echo "  Migrating $$svc..."; \
			$(MIGRATE) -path $(MIGRATION_DIR)/$$svc -database "$${DATABASE_URL_$$(echo $$svc | tr a-z A-Z)}" up; \
		fi; \
	done
	@echo "==> Migrations complete."

## migrate-down: Rollback last migration (all services)
migrate-down:
	@echo "==> Rolling back migrations..."
	@for svc in $(SERVICES); do \
		if [ -d "$(MIGRATION_DIR)/$$svc" ]; then \
			echo "  Rolling back $$svc..."; \
			$(MIGRATE) -path $(MIGRATION_DIR)/$$svc -database "$${DATABASE_URL_$$(echo $$svc | tr a-z A-Z)}" down 1; \
		fi; \
	done
	@echo "==> Rollback complete."

## migrate-create: Create a new migration (usage: make migrate-create SVC=reservation NAME=add_index)
migrate-create:
	@if [ -z "$(SVC)" ] || [ -z "$(NAME)" ]; then \
		echo "Error: SVC and NAME required. Usage: make migrate-create SVC=reservation NAME=add_index"; exit 1; \
	fi
	@mkdir -p $(MIGRATION_DIR)/$(SVC)
	$(MIGRATE) create -ext sql -dir $(MIGRATION_DIR)/$(SVC) -seq $(NAME)

## load-test: Run k6 smoke test
load-test:
	@echo "==> Running k6 smoke test..."
	$(K6) run --vus 10 --duration 30s tests/load/smoke.js
	@echo "==> Smoke test complete."

## load-test-stress: Run k6 stress test
load-test-stress:
	@echo "==> Running k6 stress test..."
	$(K6) run tests/load/stress.js

## load-test-spike: Run k6 spike test
load-test-spike:
	@echo "==> Running k6 spike test..."
	$(K6) run tests/load/spike.js

## ci: Run all checks locally (mirrors CI pipeline)
ci: fmt vet lint security test-coverage build
	@echo ""
	@echo "============================================"
	@echo "  All CI checks passed!"
	@echo "============================================"

## clean: Remove build artifacts and caches
clean:
	@echo "==> Cleaning..."
	@rm -rf bin/
	@rm -rf $(COVERAGE_DIR)/
	@$(GO) clean -cache -testcache
	@echo "==> Clean complete."

## tools: Install development tools
tools:
	@echo "==> Installing development tools..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install github.com/securego/gosec/v2/cmd/gosec@latest
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	$(GO) install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "==> Tools installed."

## mod: Tidy and verify go modules
mod:
	@echo "==> Tidying modules..."
	$(GO) mod tidy
	$(GO) mod verify

## generate: Run go generate
generate:
	@echo "==> Running go generate..."
	$(GO) generate ./...
