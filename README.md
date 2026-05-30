# arx-ca

A Go-based Certificate Authority platform built on the [step-ca SDK](https://github.com/smallstep/certificates). It exposes a RESTful HTTP API for X.509 and SSH PKI operations, enrollment protocols (ACME, SCEP, NDES), and a three-tier client model: server, super-admin CLI, and read-only local agent.

## Three-tier architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         arx-ca-server (:8080)                           │
│  REST API · OCSP · ACME · SCEP · NDES · JWT/API-key RBAC · step-ca PKI  │
│  Config: server.yaml (beside the binary) + optional CA_API_* env vars     │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │ HTTPS / HTTP
              ┌─────────────────┴─────────────────┐
              │                                   │
              ▼                                   ▼
┌─────────────────────────────┐   ┌─────────────────────────────────────┐
│       arx-ca-cli            │   │         arx-cert-service            │
│  Super Admin CLI + TUI      │   │  Read-only local agent              │
│  login · ui                 │   │  local · trust · server             │
│  Config: ~/.arx/cli.yaml    │   │  State: ~/.arx-cert-service/        │
│  Auth: ~/.arx/config.json   │   │  No private keys from the server    │
└─────────────────────────────┘   └─────────────────────────────────────┘
```

| Tier | Binary | Role | Configuration |
| ---- | ------ | ---- | ------------- |
| **Server** | `arx-ca-server` | CA API, signing engine, enrollment protocols | `server.yaml` next to the executable; env vars override YAML |
| **Super Admin CLI** | `arx-ca-cli` | Admin login and terminal UI for CA operations | `~/.arx/cli.yaml` (defaults); `~/.arx/config.json` (saved JWT) |
| **Read-only agent** | `arx-cert-service` | Local cert inventory, trust-store install, public cert download | CLI flags (`--url`); state under `~/.arx-cert-service/` |

## Design principle: local-first, cloud-optional

**arx-ca runs entirely on your infrastructure without cloud dependency.**

| Layer | Default (local) | Optional (plugins) |
| ----- | --------------- | ------------------ |
| Cryptography | SoftCAS — keys on disk under `.pki/` | AWS KMS, GCP Cloud KMS, HashiCorp Vault |
| Persistence | Embedded BadgerDB (`badgerv2`) | PostgreSQL via `ca.json` / `CA_API_DB_*` |
| Identity | JWT admin login and API key service accounts | OIDC provisioners (`CA_API_OIDC_*`) |
| Observability | Structured request logging | OpenTelemetry OTLP export |

The server automatically heals a corrupted BadgerDB value log on startup (truncate and retry). No separate database container is required for the default stack.

## Build

Requires **Go 1.22+** (see `go.mod` for the toolchain pin).

```bash
# Build all three binaries into bin/
make build

# Individual targets
make build-server    # bin/arx-ca-server
make build-cli       # bin/arx-ca-cli
make build-agent     # bin/arx-cert-service

# Cross-compile (Linux / Windows)
make build-server-linux build-server-windows \
     build-cli-linux build-cli-windows \
     build-agent-linux build-agent-windows
```

Windows artifacts are written as `bin/*.exe`. Linux artifacts have no extension.

Without `make` (e.g. on Windows without GNU Make):

```powershell
mkdir bin -Force
go build -trimpath -ldflags="-s -w" -o bin/arx-ca-server.exe ./cmd/server
go build -trimpath -ldflags="-s -w" -o bin/arx-ca-cli.exe ./cmd/cli
go build -trimpath -ldflags="-s -w" -o bin/arx-cert-service.exe ./cmd/agent
```

Other Makefile targets:

| Target | Description |
| ------ | ----------- |
| `make all` | Alias for `make build` |
| `make clean` | Remove `bin/` and coverage artifacts |
| `make test` | Run Go unit tests |
| `make build-fips` | Build server with `GOEXPERIMENT=boringcrypto` |
| `make docker-build` | Build `arx-ca-server:latest` image |
| `make docker-up` | Start Compose stack |
| `make docker-down` | Stop Compose stack |

## Configuration (Viper)

On first start, each binary auto-creates its YAML config if missing.

### Server — `server.yaml` (beside the executable)

Created next to `arx-ca-server` (e.g. `bin/server.yaml` when the binary lives in `bin/`). Example:

```yaml
host: ""
port: 8080
ca_config_path: .pki/config/ca.json
log_level: info
db_type: badgerv2
db_data_source: ""
otel_service_name: arx-ca
otel_exporter_endpoint: http://localhost:4318
otel_exporter_insecure: true
otel_sdk_disabled: false
```

`InitServerConfig` loads this file and exports unset values into `CA_API_*` and `OTEL_*` environment variables. **Explicit environment variables always override YAML** for listen address and CA config path.

Run the server from the repository root (or set `ca_config_path` to an absolute path) so `.pki/` resolves correctly.

### CLI — `~/.arx/cli.yaml`

```yaml
server_url: http://localhost:8080
log_level: info
```

After `arx-ca-cli login`, credentials are stored separately in `~/.arx/config.json`.

### Agent — flags

`arx-cert-service` does not use Viper. Pass `--url` to subcommands that talk to the server (e.g. `trust install-root --url http://localhost:8080`).

On Windows, `local list` skips certificate stores that require elevated privileges (for example Local Machine `ROOT`) and continues with accessible user and browser stores.

## Deployment

### Local (bare metal)

```bash
make build

# Recommended for production and stable JWT sessions across restarts
export CA_API_JWT_SECRET="$(openssl rand -base64 32)"

# Disable OTLP export when no collector is running
export OTEL_SDK_DISABLED=true

./bin/arx-ca-server
```

On first run the server bootstraps PKI material under `.pki/` (Root CA, Intermediate CA, JWK provisioner, SSH CAs, BadgerDB). Default listen address: `:8080`.

Verify health:

```bash
curl http://localhost:8080/api/v1/health
# Expect HTTP 200 with ca_backend.status "healthy"

curl http://localhost:8080/api/v1/ca/root
# Expect HTTP 200 with a PEM certificate
```

Authenticate and exercise protected endpoints:

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"ArxRootCA-Bootstrap-Admin-2026!"}' \
  | jq -r '.data.token')

curl -s -X POST http://localhost:8080/api/v1/certificates/auto \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"common_name":"example.local","dns_sans":["example.local"],"ttl":"24h"}'
```

Super Admin CLI:

```bash
./bin/arx-ca-cli login
./bin/arx-ca-cli ui          # interactive TUI — do not run in unattended CI
./bin/arx-ca-cli --help
```

Read-only agent:

```bash
./bin/arx-cert-service local list
./bin/arx-cert-service server list --url http://localhost:8080
./bin/arx-cert-service server download --url http://localhost:8080 --kind root -o root.pem
./bin/arx-cert-service trust install-root --url http://localhost:8080
```

### Docker Compose

```bash
cp .env.example .env
# Edit .env and set CA_API_JWT_SECRET

make docker-build
make docker-up
```

The container listens on port **8080**. PKI data persists in `./data/arx-ca` (mounted to `/app/data`). The image sets `CA_API_CA_CONFIG=/app/data/config/ca.json` so certificates and BadgerDB survive restarts.

## API overview

| Method | Path | Auth | Description |
| ------ | ---- | ---- | ----------- |
| `GET` | `/api/v1/health` | — | Health and runtime metrics |
| `GET` | `/api/v1/ca/root` | — | Root CA PEM |
| `GET` | `/api/v1/ca/crl` | — | CRL (DER; `?pem` for PEM) |
| `GET` | `/api/v1/public/certificates` | — | Public certificate inventory |
| `POST` | `/ocsp` | — | OCSP responder |
| `POST` | `/api/v1/auth/login` | — | Admin JWT |
| `POST` | `/api/v1/auth/service-accounts` | Admin | Create API key |
| `POST` | `/api/v1/certificates/issue` | Service / Admin | Sign CSR |
| `POST` | `/api/v1/certificates/auto` | Service / Admin | Generate key + sign |
| `POST` | `/api/v1/certificates/revoke` | Service / Admin | Revoke by serial |
| `GET` | `/api/v1/certificates` | Service / Admin | List certificates |
| `POST` | `/api/v1/ssh/sign-user` | Token / API / Admin | SSH user certificate |
| `GET` | `/api/v1/ssh/roots` | — | SSH CA public keys |
| `*` | `/acme/...` | ACME | ACME directory (when enabled) |
| `*` | `/scep/...` | SCEP | SCEP enrollment (when enabled) |

JSON envelope: `{ "data": …, "error": null | { "message": "…" } }`.

## Environment variables

Copy `.env.example` to `.env` for Compose. For bare-metal, prefer `server.yaml`; use env vars for overrides or secrets injection.

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `CA_API_LISTEN_ADDR` | `:8080` (from YAML) | HTTP listen address |
| `CA_API_CA_CONFIG` | `.pki/config/ca.json` | Path to step-ca `ca.json` |
| `CA_API_JWT_SECRET` | *(ephemeral if unset)* | HMAC secret for admin JWTs — **set in production** |
| `CA_API_JWT_ISSUER` | `arx-ca` | JWT issuer claim |
| `CA_API_JWT_EXPIRY` | `24h` | Admin token lifetime |
| `CA_API_DB_TYPE` | `badgerv2` | Database driver hint |
| `CA_API_DB_DATA_SOURCE` | — | PostgreSQL DSN when using external DB |
| `CA_API_OIDC_*` | — | Optional OIDC provisioner |
| `CA_API_ACME_*` | — | ACME ports, DNS, EAB, device attestation |
| `CA_API_SCEP_*` | — | SCEP challenge and disable flag |
| `CA_API_NDES_*` | — | NDES connector settings |
| `OTEL_SERVICE_NAME` | `arx-ca` | OpenTelemetry service name |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` | OTLP/HTTP collector |
| `OTEL_EXPORTER_OTLP_INSECURE` | `true` for `http://` | Disable TLS for OTLP |
| `OTEL_SDK_DISABLED` | `false` | Disable telemetry exporters |

For **PostgreSQL**, configure the `db` block in `ca.json` per [step-ca database documentation](https://smallstep.com/docs/step-ca/configuration#databases) after bootstrap, then restart the server.

## RBAC

| Role | Credential | Use |
| ---- | ---------- | --- |
| **Admin** | JWT from `/api/v1/auth/login` | Service accounts, templates, break-glass |
| **Service account** | `X-API-Key` or Bearer API key | CI/CD issuance, renewals |
| **Provisioner token** | JWK or OIDC token in request body | End-entity or SSH user flows |

## Bootstrap admin

On first login use username `admin` with password `ArxRootCA-Bootstrap-Admin-2026!`. **Change the bcrypt hash in `internal/auth/admin.go` before any production deployment.**

## Project layout

| Path | Purpose |
| ---- | ------- |
| `cmd/server` | `arx-ca-server` entrypoint |
| `cmd/cli` | `arx-ca-cli` entrypoint |
| `cmd/agent` | `arx-cert-service` entrypoint |
| `internal/api` | HTTP handlers and middleware |
| `internal/ca` | PKI engine (step-ca wrapper), Badger self-healing |
| `internal/config` | Viper YAML init and optional cloud integrations |
| `internal/cli` | CLI API client, login, TUI |
| `internal/agent` | Local cert stores, trust install, public download |

## License

See repository license file.
