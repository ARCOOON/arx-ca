.PHONY: all build build-agent build-all build-linux build-linux-agent build-windows build-windows-agent build-fips clean test docker-build docker-up docker-down

VERSION ?= v0.0.0-dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DIR = bin
LDFLAGS = -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -s -w

BINARY := arx
AGENT_BINARY := arx-agent
PKG := ./cmd/arx
AGENT_PKG := ./cmd/arx-agent
DOCKER_IMAGE := arx-ca:latest
COMPOSE := docker compose
GOFLAGS := -buildvcs=false
export GOFLAGS

GO_LINUX := GOOS=linux GOARCH=amd64 CGO_ENABLED=0
GO_WINDOWS := GOOS=windows GOARCH=amd64 CGO_ENABLED=0

all: build-all

build-all: build build-linux-agent build-windows-agent build-linux build-windows

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(PKG)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(AGENT_BINARY) $(AGENT_PKG)

build-agent:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(AGENT_BINARY) $(AGENT_PKG)

build-linux:
	@mkdir -p $(BUILD_DIR)
	$(GO_LINUX) go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(PKG)

build-linux-agent:
	@mkdir -p $(BUILD_DIR)
	$(GO_LINUX) go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(AGENT_BINARY) $(AGENT_PKG)

build-windows:
	@mkdir -p $(BUILD_DIR)
	$(GO_WINDOWS) go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY).exe $(PKG)

build-windows-agent:
	@mkdir -p $(BUILD_DIR)
	$(GO_WINDOWS) go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(AGENT_BINARY).exe $(AGENT_PKG)

build-fips:
	@mkdir -p $(BUILD_DIR)
	GOEXPERIMENT=boringcrypto go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(PKG)

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.txt
	@find . -name '*.test' -not -path './vendor/*' -delete 2>/dev/null || true

test:
	go test ./...

docker-build:
	docker build -t $(DOCKER_IMAGE) .

docker-up:
	$(COMPOSE) up -d

docker-down:
	$(COMPOSE) down
