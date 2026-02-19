#!/usr/bin/make -f

BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
COMMIT := $(shell git log -1 --format='%H' 2>/dev/null)
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "v0.0.0-dev")

BINARY_NAME := qorechaind
BUILD_DIR := ./build

GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin

# CGO is required for PQC Rust FFI
export CGO_ENABLED := 1

# Detect OS/ARCH for library path
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
LIB_DIR := ./lib/$(GOOS)_$(GOARCH)

LDFLAGS := -X github.com/cosmos/cosmos-sdk/version.Name=qorechain \
	-X github.com/cosmos/cosmos-sdk/version.AppName=$(BINARY_NAME) \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(LDFLAGS)'

###############################################################################
###                                Build                                    ###
###############################################################################

all: build

.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/qorechaind

.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(BUILD_FLAGS) ./cmd/qorechaind

.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)

###############################################################################
###                                Testing                                  ###
###############################################################################

.PHONY: test
test:
	@echo "Running unit tests..."
	go test ./... -count=1

.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	go test ./tests/integration/... -count=1 -v

.PHONY: test-race
test-race:
	@echo "Running tests with race detector..."
	go test ./... -race -count=1

###############################################################################
###                                Linting                                  ###
###############################################################################

.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run --timeout 5m

.PHONY: format
format:
	@echo "Formatting code..."
	gofumpt -l -w .

###############################################################################
###                                Protobuf                                 ###
###############################################################################

.PHONY: proto-gen
proto-gen:
	@echo "Generating protobuf files..."
	@./scripts/protocgen.sh

###############################################################################
###                                Docker                                   ###
###############################################################################

.PHONY: docker-build
docker-build:
	docker build -t qorechain/qorechain-node:latest .

.PHONY: docker-up
docker-up:
	docker-compose up -d

.PHONY: docker-down
docker-down:
	docker-compose down

###############################################################################
###                             Init & Run                                  ###
###############################################################################

.PHONY: init
init: build
	@echo "Initializing QoreChain testnet..."
	$(BUILD_DIR)/$(BINARY_NAME) init testnode --chain-id qorechain-diana

.PHONY: start
start:
	$(BUILD_DIR)/$(BINARY_NAME) start

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build            - Build the qorechaind binary"
	@echo "  install          - Install qorechaind to GOBIN"
	@echo "  clean            - Remove build artifacts"
	@echo "  test             - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  lint             - Run linter"
	@echo "  format           - Format code"
	@echo "  proto-gen        - Generate protobuf code"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-up        - Start Docker Compose stack"
	@echo "  docker-down      - Stop Docker Compose stack"
	@echo "  init             - Initialize a testnet node"
	@echo "  start            - Start the node"
