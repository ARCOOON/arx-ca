.PHONY: all build build-server build-cli build-agent clean test \
	build-server-linux build-server-windows \
	build-cli-linux build-cli-windows \
	build-agent-linux build-agent-windows \
	build-fips docker-build docker-up docker-down

SERVER_BINARY := arx-ca-server
CLI_BINARY := arx-ca-cli
AGENT_BINARY := arx-cert-service
BIN_DIR := bin
SERVER_PKG := ./cmd/server
CLI_PKG := ./cmd/cli
AGENT_PKG := ./cmd/agent
DOCKER_IMAGE := arx-ca-server:latest
COMPOSE := docker compose
LDFLAGS := -trimpath -ldflags="-s -w"
GOFLAGS := -buildvcs=false
export GOFLAGS

GO_LINUX := GOOS=linux GOARCH=amd64 CGO_ENABLED=0
GO_WINDOWS := GOOS=windows GOARCH=amd64 CGO_ENABLED=0

all: build

build: build-server build-cli build-agent

build-server:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(SERVER_BINARY) $(SERVER_PKG)

build-cli:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(CLI_BINARY) $(CLI_PKG)

build-agent:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(AGENT_BINARY) $(AGENT_PKG)

build-server-linux:
	@mkdir -p $(BIN_DIR)
	$(GO_LINUX) go build $(LDFLAGS) -o $(BIN_DIR)/$(SERVER_BINARY) $(SERVER_PKG)

build-server-windows:
	@mkdir -p $(BIN_DIR)
	$(GO_WINDOWS) go build $(LDFLAGS) -o $(BIN_DIR)/$(SERVER_BINARY).exe $(SERVER_PKG)

build-cli-linux:
	@mkdir -p $(BIN_DIR)
	$(GO_LINUX) go build $(LDFLAGS) -o $(BIN_DIR)/$(CLI_BINARY) $(CLI_PKG)

build-cli-windows:
	@mkdir -p $(BIN_DIR)
	$(GO_WINDOWS) go build $(LDFLAGS) -o $(BIN_DIR)/$(CLI_BINARY).exe $(CLI_PKG)

build-agent-linux:
	@mkdir -p $(BIN_DIR)
	$(GO_LINUX) go build $(LDFLAGS) -o $(BIN_DIR)/$(AGENT_BINARY) $(AGENT_PKG)

build-agent-windows:
	@mkdir -p $(BIN_DIR)
	$(GO_WINDOWS) go build $(LDFLAGS) -o $(BIN_DIR)/$(AGENT_BINARY).exe $(AGENT_PKG)

build-fips:
	@mkdir -p $(BIN_DIR)
	GOEXPERIMENT=boringcrypto go build $(LDFLAGS) -o $(BIN_DIR)/$(SERVER_BINARY) $(SERVER_PKG)

clean:
	rm -rf $(BIN_DIR)
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
