.PHONY: all build build-linux build-windows build-fips clean test docker-build docker-up docker-down

BINARY := arx
BIN_DIR := bin
PKG := ./cmd/arx
DOCKER_IMAGE := arx-ca:latest
COMPOSE := docker compose
LDFLAGS := -trimpath -ldflags="-s -w"
GOFLAGS := -buildvcs=false
export GOFLAGS

GO_LINUX := GOOS=linux GOARCH=amd64 CGO_ENABLED=0
GO_WINDOWS := GOOS=windows GOARCH=amd64 CGO_ENABLED=0

all: build

build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) $(PKG)

build-linux:
	@mkdir -p $(BIN_DIR)
	$(GO_LINUX) go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) $(PKG)

build-windows:
	@mkdir -p $(BIN_DIR)
	$(GO_WINDOWS) go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY).exe $(PKG)

build-fips:
	@mkdir -p $(BIN_DIR)
	GOEXPERIMENT=boringcrypto go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) $(PKG)

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
