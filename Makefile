.PHONY: build clean test docker-build docker-up docker-down

BINARY_NAME := arx-rootca
BIN_DIR := bin
MAIN_PKG := ./cmd/server
DOCKER_IMAGE := arx-rootca:latest
COMPOSE := docker compose

# Default build target for convenience.
all: build

build:
	@mkdir -p $(BIN_DIR)
	go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PKG)

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
