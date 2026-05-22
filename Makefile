.DEFAULT_GOAL := help

GO ?= go
PROTOC ?= protoc
GOFMT ?= gofmt
GOVULNCHECK ?= govulncheck
GOVULNCHECK_PKG ?= golang.org/x/vuln/cmd/govulncheck@latest
SQLC ?= sqlc
SQLC_PKG ?= github.com/sqlc-dev/sqlc/cmd/sqlc@latest
MIGRATE ?= migrate
MIGRATE_PKG ?= github.com/golang-migrate/migrate/v4/cmd/migrate@latest
GOFLAGS ?= -buildvcs=false
GOCACHE ?= $(CURDIR)/.cache/go-build
BIN_DIR ?= $(CURDIR)/bin
PROTO_DIR := api/proto
GEN_DIR := gen
MODULE := github.com/vladfc/event-driven-ecommerce-app
DB_MIGRATIONS_DIR := db/migrations

CATALOG_CMD := ./cmd/catalog-service
INVENTORY_CMD := ./cmd/inventory-service
PAYMENT_CMD := ./cmd/payment-service
ORDER_CMD := ./cmd/order-service
GATEWAY_CMD := ./cmd/gateway-service

COMPOSE ?= docker compose
KAFKA_BROKERS ?= localhost:19092
KAFKA_ORDER_CREATED_TOPIC ?= orders.created
REDIS_ADDR ?= localhost:6379
POSTGRES_HOST ?= localhost
POSTGRES_PORT ?= 5432
POSTGRES_DB ?= ecommerce
POSTGRES_USER ?= app
POSTGRES_PASSWORD ?= app
DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable
PAYMENT_DATABASE_URL ?= $(DATABASE_URL)
ORDER_DATABASE_URL ?= $(DATABASE_URL)

.PHONY: help doctor fmt vet test govulncheck govulncheck-install check tidy proto proto-check build clean \
	build-catalog build-inventory build-payment build-order build-gateway \
	run-catalog run-inventory run-inventory-kafka run-payment run-payment-kafka run-order run-order-kafka run-gateway run-services \
	kafka-up kafka-down kafka-logs db-up db-down db-logs db-prepare db-migrate-up db-migrate-down db-migrate-version \
	migrate-install sqlc-install sqlc-generate sqlc-verify

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

doctor: ## Check required local tooling
	@command -v $(GO) >/dev/null || { echo "$(GO) is not installed or not in PATH"; exit 1; }
	@command -v $(GOFMT) >/dev/null || { echo "$(GOFMT) is not installed or not in PATH"; exit 1; }
	@command -v $(GOVULNCHECK) >/dev/null || { echo "$(GOVULNCHECK) is not installed or not in PATH"; exit 1; }
	@command -v $(PROTOC) >/dev/null || { echo "$(PROTOC) is not installed or not in PATH"; exit 1; }
	@command -v protoc-gen-go >/dev/null || { echo "protoc-gen-go is not installed or not in PATH"; exit 1; }
	@command -v protoc-gen-go-grpc >/dev/null || { echo "protoc-gen-go-grpc is not installed or not in PATH"; exit 1; }
	@command -v $(SQLC) >/dev/null || { echo "$(SQLC) is not installed or not in PATH"; exit 1; }
	@command -v $(MIGRATE) >/dev/null || { echo "$(MIGRATE) is not installed or not in PATH"; echo "install it with: make migrate-install"; exit 1; }
	@echo "tooling looks good"

fmt: ## Format Go code
	$(GOFMT) -w $$(find . -type f -name '*.go' -not -path './vendor/*')

vet: ## Run go vet
	GOCACHE=$(GOCACHE) $(GO) vet $(GOFLAGS) ./...

test: ## Run all tests
	GOCACHE=$(GOCACHE) $(GO) test $(GOFLAGS) ./...

govulncheck: ## Run Go vulnerability scan
	@command -v $(GOVULNCHECK) >/dev/null || { \
		echo "$(GOVULNCHECK) is not installed or not in PATH"; \
		echo "install it with: make govulncheck-install"; \
		exit 1; \
	}
	@GOFLAGS="$(GOFLAGS)" GOCACHE=$(GOCACHE) $(GOVULNCHECK) ./... || { \
		echo ""; \
		echo "govulncheck failed"; \
		echo "if your Go version was upgraded recently, rebuild the tool with: make govulncheck-install"; \
		exit 1; \
	}

govulncheck-install: ## Install or rebuild govulncheck with the current Go toolchain
	GOCACHE=$(GOCACHE) $(GO) install $(GOVULNCHECK_PKG)

migrate-install: ## Install or rebuild golang-migrate with the current Go toolchain
	GOCACHE=$(GOCACHE) $(GO) install -tags 'postgres' $(MIGRATE_PKG)

sqlc-install: ## Install or rebuild sqlc with the current Go toolchain
	GOCACHE=$(GOCACHE) $(GO) install $(SQLC_PKG)

sqlc-generate: ## Generate Go code from SQL using sqlc
	$(SQLC) generate

sqlc-verify: ## Verify sqlc configuration and generated code are up to date
	$(SQLC) vet

check: vet test govulncheck ## Run the default verification suite

tidy: ## Tidy go.mod and go.sum
	$(GO) mod tidy

proto: ## Generate protobuf and gRPC code
	PATH="$(PATH):$(HOME)/go/bin" find "$(PROTO_DIR)" -type f -name '*.proto' -print0 | xargs -0 $(PROTOC) \
		-I "$(PROTO_DIR)" \
		--go_out=. \
		--go_opt=module=$(MODULE) \
		--go-grpc_out=. \
		--go-grpc_opt=module=$(MODULE)

proto-check: ## Regenerate protobufs and fail if generated files changed
	@$(MAKE) proto
	@git diff --quiet -- $(GEN_DIR) || { \
		echo "generated protobuf code is out of date"; \
		echo "run 'make proto' and commit the updated files"; \
		git diff -- $(GEN_DIR); \
		exit 1; \
	}

build: build-catalog build-inventory build-payment build-order build-gateway ## Build all service binaries into ./bin

build-catalog: ## Build catalog-service binary
	@mkdir -p "$(BIN_DIR)"
	GOCACHE=$(GOCACHE) $(GO) build $(GOFLAGS) -o "$(BIN_DIR)/catalog-service" $(CATALOG_CMD)

build-inventory: ## Build inventory-service binary
	@mkdir -p "$(BIN_DIR)"
	GOCACHE=$(GOCACHE) $(GO) build $(GOFLAGS) -o "$(BIN_DIR)/inventory-service" $(INVENTORY_CMD)

build-payment: ## Build payment-service binary
	@mkdir -p "$(BIN_DIR)"
	GOCACHE=$(GOCACHE) $(GO) build $(GOFLAGS) -o "$(BIN_DIR)/payment-service" $(PAYMENT_CMD)

build-order: ## Build order-service binary
	@mkdir -p "$(BIN_DIR)"
	GOCACHE=$(GOCACHE) $(GO) build $(GOFLAGS) -o "$(BIN_DIR)/order-service" $(ORDER_CMD)

build-gateway: ## Build gateway-service binary
	@mkdir -p "$(BIN_DIR)"
	GOCACHE=$(GOCACHE) $(GO) build $(GOFLAGS) -o "$(BIN_DIR)/gateway-service" $(GATEWAY_CMD)

run-catalog: ## Run catalog-service
	CATALOG_DATABASE_URL=$(CATALOG_DATABASE_URL) GOCACHE=$(GOCACHE) $(GO) run $(GOFLAGS) $(CATALOG_CMD)

run-inventory: ## Run inventory-service
	INVENTORY_DATABASE_URL=$(INVENTORY_DATABASE_URL) GOCACHE=$(GOCACHE) $(GO) run $(GOFLAGS) $(INVENTORY_CMD)

run-inventory-kafka: ## Run inventory-service against the local Redpanda broker and Redis
	INVENTORY_DATABASE_URL=$(INVENTORY_DATABASE_URL) KAFKA_BROKERS=$(KAFKA_BROKERS) REDIS_ADDR=$(REDIS_ADDR) GOCACHE=$(GOCACHE) $(GO) run $(GOFLAGS) $(INVENTORY_CMD)

run-payment: ## Run payment-service
	PAYMENT_DATABASE_URL=$(PAYMENT_DATABASE_URL) GOCACHE=$(GOCACHE) $(GO) run $(GOFLAGS) $(PAYMENT_CMD)

run-payment-kafka: ## Run payment-service against the local Redpanda broker and Redis
	PAYMENT_DATABASE_URL=$(PAYMENT_DATABASE_URL) KAFKA_BROKERS=$(KAFKA_BROKERS) REDIS_ADDR=$(REDIS_ADDR) GOCACHE=$(GOCACHE) $(GO) run $(GOFLAGS) $(PAYMENT_CMD)

run-order: ## Run order-service (requires KAFKA_BROKERS)
	ORDER_DATABASE_URL=$(ORDER_DATABASE_URL) GOCACHE=$(GOCACHE) $(GO) run $(GOFLAGS) $(ORDER_CMD)

run-order-kafka: ## Run order-service against the local Redpanda broker
	ORDER_DATABASE_URL=$(ORDER_DATABASE_URL) KAFKA_BROKERS=$(KAFKA_BROKERS) KAFKA_ORDER_CREATED_TOPIC=$(KAFKA_ORDER_CREATED_TOPIC) GOCACHE=$(GOCACHE) $(GO) run $(GOFLAGS) $(ORDER_CMD)

run-gateway: ## Run gateway-service
	GOCACHE=$(GOCACHE) $(GO) run $(GOFLAGS) $(GATEWAY_CMD)

kafka-up: ## Start local Redpanda + Console + Redis
	$(COMPOSE) up -d redpanda redpanda-console redis

kafka-down: ## Stop local Redpanda + Console
	$(COMPOSE) down

kafka-logs: ## Tail local Redpanda + Console + Redis logs
	$(COMPOSE) logs -f redpanda redpanda-console redis

db-up: ## Start local PostgreSQL
	$(COMPOSE) up -d postgres

db-down: ## Stop local PostgreSQL
	$(COMPOSE) stop postgres

db-logs: ## Tail local PostgreSQL logs
	$(COMPOSE) logs -f postgres

db-prepare: db-up db-migrate-up ## Start shared PostgreSQL and apply shared service migrations

db-migrate-up: ## Apply shared PostgreSQL migrations for local services
	$(MIGRATE) -path $(DB_MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

db-migrate-down: ## Roll back the most recent shared PostgreSQL migration
	$(MIGRATE) -path $(DB_MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

db-migrate-version: ## Show the current shared PostgreSQL migration version
	$(MIGRATE) -path $(DB_MIGRATIONS_DIR) -database "$(DATABASE_URL)" version

run-services: ## Print the recommended local startup order
	@echo "Start services in separate terminals in this order:"
	@echo "  make kafka-up"
	@echo "  make db-prepare"
	@echo "  make run-catalog"
	@echo "  make run-inventory-kafka"
	@echo "  make run-payment-kafka"
	@echo "  make run-order-kafka"
	@echo "  make run-gateway"

clean: ## Remove local build artifacts and cache
	rm -rf "$(BIN_DIR)" "$(CURDIR)/.cache"
