# arx-ca unified build and development targets.
.DEFAULT_GOAL := help

.PHONY: help all build build-agent build-all build-arx-cross build-agent-cross \
	webui clean test docker-build docker-up docker-down build-fips \
	build-arx-linux-amd64 build-arx-linux-arm64 \
	build-arx-darwin-amd64 build-arx-darwin-arm64 \
	build-arx-windows-amd64 \
	build-arx-agent-linux-amd64 build-arx-agent-linux-arm64 \
	build-arx-agent-darwin-amd64 build-arx-agent-darwin-arm64 \
	build-arx-agent-windows-amd64 \
	build-linux build-linux-agent build-windows build-windows-agent

VERSION ?= v0.0.0-dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BIN_DIR := bin
BUILD_DIR := build
WEBUI_DIST := $(BUILD_DIR)/webui-dist.tar.gz
LDFLAGS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -s -w

BINARY := arx
AGENT_BINARY := arx-agent
PKG := ./cmd/arx
AGENT_PKG := ./cmd/arx-agent
DOCKER_IMAGE := arx-ca:latest
COMPOSE := docker compose
GOFLAGS := -buildvcs=false
export GOFLAGS

CROSS_ARX_TARGETS := \
	build-arx-linux-amd64 \
	build-arx-linux-arm64 \
	build-arx-darwin-amd64 \
	build-arx-darwin-arm64 \
	build-arx-windows-amd64

CROSS_AGENT_TARGETS := \
	build-arx-agent-linux-amd64 \
	build-arx-agent-linux-arm64 \
	build-arx-agent-darwin-amd64 \
	build-arx-agent-darwin-arm64 \
	build-arx-agent-windows-amd64

##@ General

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

all: build-all ## Alias for build-all

##@ Local development (native host)

build: ## Build arx and arx-agent for the current OS/arch into bin/
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(PKG)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(AGENT_BINARY) $(AGENT_PKG)

build-agent: ## Build arx-agent only for the current OS/arch into bin/
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(AGENT_BINARY) $(AGENT_PKG)

build-fips: ## Build arx with GOEXPERIMENT=boringcrypto (local bin/)
	@mkdir -p $(BIN_DIR)
	GOEXPERIMENT=boringcrypto go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(PKG)

test: ## Run all Go tests
	go test ./...

##@ Cross-compilation (release binaries in build/)

build-arx-cross: $(CROSS_ARX_TARGETS) ## Build arx for all supported OS/arch combinations

build-agent-cross: $(CROSS_AGENT_TARGETS) ## Build arx-agent for all supported OS/arch combinations

build-arx-linux-amd64: ## Cross-compile arx for Linux amd64
	@$(call cross_go_build,linux,amd64,$(BINARY),$(PKG))

build-arx-linux-arm64: ## Cross-compile arx for Linux arm64
	@$(call cross_go_build,linux,arm64,$(BINARY),$(PKG))

build-arx-darwin-amd64: ## Cross-compile arx for Darwin amd64
	@$(call cross_go_build,darwin,amd64,$(BINARY),$(PKG))

build-arx-darwin-arm64: ## Cross-compile arx for Darwin arm64
	@$(call cross_go_build,darwin,arm64,$(BINARY),$(PKG))

build-arx-windows-amd64: ## Cross-compile arx for Windows amd64
	@$(call cross_go_build,windows,amd64,$(BINARY),$(PKG))

build-arx-agent-linux-amd64: ## Cross-compile arx-agent for Linux amd64
	@$(call cross_go_build,linux,amd64,$(AGENT_BINARY),$(AGENT_PKG))

build-arx-agent-linux-arm64: ## Cross-compile arx-agent for Linux arm64
	@$(call cross_go_build,linux,arm64,$(AGENT_BINARY),$(AGENT_PKG))

build-arx-agent-darwin-amd64: ## Cross-compile arx-agent for Darwin amd64
	@$(call cross_go_build,darwin,amd64,$(AGENT_BINARY),$(AGENT_PKG))

build-arx-agent-darwin-arm64: ## Cross-compile arx-agent for Darwin arm64
	@$(call cross_go_build,darwin,arm64,$(AGENT_BINARY),$(AGENT_PKG))

build-arx-agent-windows-amd64: ## Cross-compile arx-agent for Windows amd64
	@$(call cross_go_build,windows,amd64,$(AGENT_BINARY),$(AGENT_PKG))

# Backward-compatible shorthand targets (Linux/Windows amd64 only).
build-linux: build-arx-linux-amd64 ## Alias: arx Linux amd64

build-linux-agent: build-arx-agent-linux-amd64 ## Alias: arx-agent Linux amd64

build-windows: build-arx-windows-amd64 ## Alias: arx Windows amd64

build-windows-agent: build-arx-agent-windows-amd64 ## Alias: arx-agent Windows amd64

define cross_go_build
	@mkdir -p $(BUILD_DIR)
	@echo "Building $(3) for $(1)-$(2)"
	CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) go build -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(3)-$(1)-$(2)$(if $(filter windows,$(1)),.exe,) $(4)
endef

##@ WebUI and full release suite

webui: ## Install deps, build WebUI, and package webui-dist.tar.gz
	@cd webui && npm install
	@cd webui && npm run build
	@tar -czf $(WEBUI_DIST) -C webui/dist .
	@echo "WebUI built and packaged to $(WEBUI_DIST)"

build-all: build-arx-cross build-agent-cross webui ## Full release suite: all cross binaries + WebUI tarball

##@ Cleanup and Docker

clean: ## Remove build/, bin/, webui-dist.tar.gz, and test artifacts
	rm -rf $(BUILD_DIR) $(BIN_DIR)
	rm -f $(WEBUI_DIST) coverage.out coverage.txt
	@find . -name '*.test' -not -path './vendor/*' -delete 2>/dev/null || true

docker-build: ## Build the arx-ca Docker image
	docker build -t $(DOCKER_IMAGE) .

docker-up: ## Start docker compose stack in the background
	$(COMPOSE) up -d

docker-down: ## Stop docker compose stack
	$(COMPOSE) down
