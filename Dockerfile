# syntax=docker/dockerfile:1

# Stage 1: compile arx release binaries and the Vue WebUI distribution.
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache ca-certificates git make nodejs npm

WORKDIR /src

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOTOOLCHAIN=auto

# Cache Go module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Cache WebUI npm dependencies.
COPY webui/package.json webui/package-lock.json ./webui/
RUN cd webui && npm ci

COPY Makefile ./
COPY cmd ./cmd
COPY internal ./internal
COPY webui ./webui

# Remaining source (templates, embeds, etc.).
COPY . .

ARG TARGETARCH
RUN make build-all

# Stage 2: minimal runtime image with the platform binary and static WebUI assets.
FROM alpine:3.20 AS runner

RUN apk add --no-cache ca-certificates tzdata wget jq \
    && addgroup -g 1000 arx \
    && adduser -D -u 1000 -G arx arx

WORKDIR /app

ARG TARGETARCH

COPY --from=builder /src/build/arx-linux-amd64 /app/arx-amd64
COPY --from=builder /src/build/arx-linux-arm64 /app/arx-arm64
COPY --from=builder /src/build/webui-dist.tar.gz /tmp/webui-dist.tar.gz
COPY deploy/docker-healthcheck.sh /app/docker-healthcheck.sh

RUN ARCH=$(case "${TARGETARCH}" in arm64) echo arm64 ;; *) echo amd64 ;; esac) \
    && cp "/app/arx-${ARCH}" /app/arx \
    && chmod 755 /app/arx /app/docker-healthcheck.sh \
    && rm -f /app/arx-amd64 /app/arx-arm64 \
    && mkdir -p /app/ui \
    && tar -xzf /tmp/webui-dist.tar.gz -C /app/ui \
    && rm -f /tmp/webui-dist.tar.gz \
    && mkdir -p /data \
    && chown -R arx:arx /app /data

USER arx

VOLUME ["/data"]

EXPOSE 8080 8443

ENV ARX_WEBUI_ENABLED=true \
    ARX_WEBUI_UI_DIR=/app/ui \
    ARX_WEBUI_LISTEN_ADDRESS=:8443

HEALTHCHECK --interval=60s --timeout=10s --start-period=120s --retries=3 \
    CMD ["/app/docker-healthcheck.sh"]

ENTRYPOINT ["/app/arx", "server", "start", "--config", "/data/server.yaml"]
