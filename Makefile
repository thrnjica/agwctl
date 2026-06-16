VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X main.Version=$(VERSION)

BINARY := agwctl
BUILD_DIR := .
CMD_DIR := ./cmd/agwctl
FLAGS := GOEXPERIMENT=jsonv2
PACKAGES := ./...

.PHONY: all build test clean fmt vet tidy help

all: build ## Build the binary

build: ## Build the binary with version injection
	@$(FLAGS) go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(CMD_DIR)
	@echo "Built $(BINARY) version $(VERSION)"

test: ## Run tests
	@$(FLAGS) go test -v $(PACKAGES)

test-coverage: ## Run tests with coverage
	@$(FLAGS) go test -v -coverprofile=coverage.out $(PACKAGES)
	@$(FLAGS) go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts
	rm -f $(BUILD_DIR)/$(BINARY)
	rm -f coverage.out coverage.html
	rm -rf data

lint: ## Lint code
	@$(FLAGS) golangci-lint run $(PACKAGES) --timeout=2m

fmt: ## Format code
	@$(FLAGS) gofumpt -w -extra .

format: fmt ## Alias for fmt

vet: ## Run go vet
	@$(FLAGS) go vet $(PACKAGES)

tidy: ## Tidy go modules
	@$(FLAGS) go mod tidy

install: build ## Install binary to GOPATH/bin
	cp $(BUILD_DIR)/$(BINARY) $(GOPATH)/bin/

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_-]+:.*?##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

# Made with Bob
