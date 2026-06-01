# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOTOOLCHAIN=auto

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -trimpath -ldflags="-s -w" -o /out/arx ./cmd/arx

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 1000 app \
    && adduser -D -u 1000 -G app app

WORKDIR /app

COPY --from=builder /out/arx /app/arx

RUN mkdir -p /app/data \
    && chown -R app:app /app

USER app

EXPOSE 8080

ENV CA_API_LISTEN_ADDR=:8080 \
    CA_API_CA_CONFIG=/app/data/config/ca.json

ENTRYPOINT ["/app/arx", "server", "start"]
