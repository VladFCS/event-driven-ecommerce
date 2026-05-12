.DEFAULT_GOAL := help

GO ?= go
PROTOC ?= protoc
GOFMT ?= gofmt
GOVULNCHECK ?= govulncheck
GOVULNCHECK_PKG ?= golang.org/x/vuln/cmd/govulncheck@latest
GOFLAGS ?= -buildvcs=false
GOCACHE ?= $(CURDIR)/.cache/go-build
BIN_DIR ?= $(CURDIR)/bin
PROTO_DIR := api/proto
GEN_DIR := gen
MODULE := github.com/vladfc/event-driven-ecommerce-app

CATALOG_CMD := ./cmd/catalog-service
INVENTORY_CMD := ./cmd/inventory-service
PAYMENT_CMD := ./cmd/payment-service
ORDER_CMD := ./cmd/order-service
GATEWAY_CMD := ./cmd/gateway-service

.PHONY: help doctor fmt vet test govulncheck govulncheck-install check tidy proto proto-check build clean \
	build-catalog build-inventory build-payment build-order build-gateway \
	run-catalog run-inventory run-payment run-order run-gateway run-services

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

doctor: ## Check required local tooling
	@command -v $(GO) >/dev/null || { echo "$(GO) is not installed or not in PATH"; exit 1; }
	@command -v $(GOFMT) >/dev/null || { echo "$(GOFMT) is not installed or not in PATH"; exit 1; }
	@command -v $(GOVULNCHECK) >/dev/null || { echo "$(GOVULNCHECK) is not installed or not in PATH"; exit 1; }
	@command -v $(PROTOC) >/dev/null || { echo "$(PROTOC) is not installed or not in PATH"; exit 1; }
	@command -v protoc-gen-go >/dev/null || { echo "protoc-gen-go is not installed or not in PATH"; exit 1; }
	@command -v protoc-gen-go-grpc >/dev/null || { echo "protoc-gen-go-grpc is not installed or not in PATH"; exit 1; }
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
	GOCACHE=$(GOCACHE) $(GO) run $(GOFLAGS) $(CATALOG_CMD)

run-inventory: ## Run inventory-service
	GOCACHE=$(GOCACHE) $(GO) run $(GOFLAGS) $(INVENTORY_CMD)

run-payment: ## Run payment-service
	GOCACHE=$(GOCACHE) $(GO) run $(GOFLAGS) $(PAYMENT_CMD)

run-order: ## Run order-service
	GOCACHE=$(GOCACHE) $(GO) run $(GOFLAGS) $(ORDER_CMD)

run-gateway: ## Run gateway-service
	GOCACHE=$(GOCACHE) $(GO) run $(GOFLAGS) $(GATEWAY_CMD)

run-services: ## Print the recommended local startup order
	@echo "Start services in separate terminals in this order:"
	@echo "  make run-catalog"
	@echo "  make run-inventory"
	@echo "  make run-payment"
	@echo "  make run-order"
	@echo "  make run-gateway"

clean: ## Remove local build artifacts and cache
	rm -rf "$(BIN_DIR)" "$(CURDIR)/.cache"
