# arx-ca

A Go-based Certificate Authority platform built on the [step-ca SDK](https://github.com/smallstep/certificates). It exposes a RESTful HTTP API for X.509 and SSH PKI operations, enrollment protocols (ACME, SCEP, NDES), and a three-tier client model: server, super-admin CLI, and read-only local agent.

## Three-tier architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         arx-ca-server (:8080)                           │
│  REST API · OCSP · ACME · SCEP · NDES · JWT/API-key RBAC · step-ca PKI  │
│  Config: server.yaml (beside the binary) + ARX_* / CA_API_* env overrides │
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

## Configuration Guide

Configuration is managed with [Viper](https://github.com/spf13/viper). On first start, `arx-ca-server` and `arx-ca-cli` auto-create their YAML files if missing. `arx-cert-service` uses CLI flags only (see [CLI Reference](#command-line-interface-cli-reference)).

### Server — `server.yaml`

**Location:** Beside the `arx-ca-server` executable (for example `bin/server.yaml` when the binary lives in `bin/`). Override with `--config /path/to/server.yaml`.

**Complete example** (all nested sections with documented defaults):

```yaml
server:
  host: 0.0.0.0
  port: 8080
  log_level: info
  read_timeout: 15s
  write_timeout: 15s

database:
  host: ""                    # Empty = embedded SQLite user store (.pki/arx-ca-users.db)
  port: 5432
  user: ""
  password: ""
  password_file: ""           # Preferred over inline password (see Secure secrets)
  dbname: ""
  sslmode: disable
  max_open_conns: 25
  max_idle_conns: 5

ca:
  stepca_url: ""              # Optional remote step-ca URL
  root_path: .pki/certs/root_ca.crt
  intermediate_path: .pki/certs/intermediate_ca.crt
  provisioner_name: ca-admin
  password: ""
  password_file: ""
  provisioner_password_file: ""

security:
  jwt_secret: ""              # Auto-generated on first run if empty; set in production
  token_expiration_hours: 24

bootstrap:
  admin_email: admin@arx.local
  admin_password_hash: "$2a$10$dSttx8r7tN32Mbo/C3zOteNowfq2vyhloZndZ2OGBgFEcMl1QYj0a"

telemetry:
  service_name: arx-ca
  exporter_endpoint: http://localhost:4318
  exporter_insecure: true
  sdk_disabled: false
```

#### Server options (`server`)

| Key | Default | Description |
| --- | ------- | ----------- |
| `host` | `0.0.0.0` | Bind address (`0.0.0.0` listens on all interfaces as `:port`) |
| `port` | `8080` | HTTP listen port |
| `log_level` | `info` | Application log level |
| `read_timeout` | `15s` | `http.Server` read timeout |
| `write_timeout` | `15s` | `http.Server` write timeout |

#### Database options (`database`)

When `host` is non-empty, PostgreSQL is used for the application user store and a DSN is exported to `CA_API_DB_DATA_SOURCE`. When `host` is empty, SQLite under `.pki/arx-ca-users.db` is used.

| Key | Default | Description |
| --- | ------- | ----------- |
| `host` | *(empty)* | PostgreSQL hostname; empty disables PostgreSQL |
| `port` | `5432` | PostgreSQL port |
| `user` | *(empty)* | Database user |
| `password` | *(empty)* | Inline password (avoid in production) |
| `password_file` | *(empty)* | Path to file containing the DB password |
| `dbname` | *(empty)* | Database name |
| `sslmode` | `disable` | PostgreSQL `sslmode` query parameter |
| `max_open_conns` | `25` | Connection pool size |
| `max_idle_conns` | `5` | Idle connections in pool |

#### CA options (`ca`)

| Key | Default | Description |
| --- | ------- | ----------- |
| `stepca_url` | *(empty)* | Optional external step-ca base URL |
| `root_path` | `.pki/certs/root_ca.crt` | Root CA certificate path; used to derive `ca.json` at `.pki/config/ca.json` |
| `intermediate_path` | `.pki/certs/intermediate_ca.crt` | Intermediate CA certificate path |
| `provisioner_name` | `ca-admin` | JWK provisioner name |
| `password` | *(empty)* | Inline CA/provisioner password |
| `password_file` | *(empty)* | File containing CA password |
| `provisioner_password_file` | *(empty)* | File containing provisioner password (used if `password` / `password_file` are empty) |

#### Security options (`security`)

| Key | Default | Description |
| --- | ------- | ----------- |
| `jwt_secret` | *(generated)* | HMAC secret for admin JWTs; also read from `ARX_SECURITY_JWT_SECRET` or `CA_API_JWT_SECRET` if unset |
| `token_expiration_hours` | `24` | Admin JWT lifetime in hours |

#### Bootstrap options (`bootstrap`)

| Key | Default | Description |
| --- | ------- | ----------- |
| `admin_email` | `admin@arx.local` | Email stored for the first super-admin user |
| `admin_password_hash` | *(bcrypt in defaults)* | Bcrypt hash seeded into the `users` table when it is empty — **never put plain-text passwords here** |

#### Telemetry options (`telemetry`)

| Key | Default | Description |
| --- | ------- | ----------- |
| `service_name` | `arx-ca` | OpenTelemetry service name |
| `exporter_endpoint` | `http://localhost:4318` | OTLP/HTTP collector endpoint |
| `exporter_insecure` | `true` | Use insecure OTLP when endpoint is `http://` |
| `sdk_disabled` | `false` | Disable telemetry exporters when `true` |

### Environment variable overrides (`ARX_*`)

Viper loads `server.yaml` with env prefix `ARX` and binds keys with `AutomaticEnv()`. Dots in YAML keys become underscores in environment variable names.

| YAML key | Environment variable |
| -------- | -------------------- |
| `server.host` | `ARX_SERVER_HOST` |
| `server.port` | `ARX_SERVER_PORT` |
| `database.host` | `ARX_DATABASE_HOST` |
| `database.password_file` | `ARX_DATABASE_PASSWORD_FILE` |
| `ca.provisioner_password_file` | `ARX_CA_PROVISIONER_PASSWORD_FILE` |
| `security.jwt_secret` | `ARX_SECURITY_JWT_SECRET` |
| `bootstrap.admin_password_hash` | `ARX_BOOTSTRAP_ADMIN_PASSWORD_HASH` |

**Precedence:** Values already set in the process environment (including `CA_API_*` and `OTEL_*`) are not overwritten when `ApplyServerRuntimeFromViper()` exports YAML into legacy variables. For bootstrap, `CA_API_BOOTSTRAP_ADMIN_*` and `ARX_BOOTSTRAP_*` are also honored during normalization.

After load, unset `CA_API_*` / `OTEL_*` variables are populated from YAML (for example `CA_API_LISTEN_ADDR`, `CA_API_JWT_SECRET`, `CA_API_DB_DATA_SOURCE`). Run the server from the repository root, or use absolute paths under `ca.root_path`, so `.pki/` resolves correctly.

### Secure secret management

Sensitive values should not live in plain text in `server.yaml` when avoidable.

**`password_file` (database and CA):** When `password_file` (or `provisioner_password_file`) is set, `ResolveSecret` reads the trimmed contents of that file and uses them instead of the inline `password` field. If the file cannot be read, the inline value is used as a fallback.

```bash
# Example: provisioner password from a root-only file
echo -n 'my-provisioner-secret' | sudo tee /etc/arx-ca/provisioner.pass
sudo chmod 600 /etc/arx-ca/provisioner.pass
```

```yaml
ca:
  provisioner_password_file: /etc/arx-ca/provisioner.pass
```

**`admin_password_hash` (bootstrap):** Only bcrypt hashes belong in `bootstrap.admin_password_hash`. Generate a hash with `arx-ca-cli util hash` (see [arx-ca-cli](#arx-ca-cli)) and paste it into `server.yaml` or set `ARX_BOOTSTRAP_ADMIN_PASSWORD_HASH`. On first start, when the `users` table is empty, the server seeds one super-admin row with this hash.

**JWT secret:** Leave `security.jwt_secret` empty only for development; a random secret is generated once. In production, set `security.jwt_secret` or `CA_API_JWT_SECRET` explicitly.

### CLI — `~/.arx/cli.yaml`

Created on first `arx-ca-cli` run with mode `0600` in a `0700` directory.

```yaml
server_url: http://localhost:8080
log_level: info
```

| Key | Default | Description |
| --- | ------- | ----------- |
| `server_url` | `http://localhost:8080` | Default base URL for `login` and `ui` |
| `log_level` | `info` | CLI log level |

### CLI session — `~/.arx/config.json`

Written by `arx-ca-cli login` (mode `0600`). Not managed by Viper.

| Field | Description |
| ----- | ----------- |
| `server_url` | API base URL used for the session |
| `token` | JWT from `POST /api/v1/auth/login` |
| `token_type` | Usually `Bearer` |
| `expires_at` | Token expiry (RFC3339) |
| `username` | Logged-in admin username |

## Command Line Interface (CLI) Reference

Three binaries ship with the project. All support `--help` on the root command and subcommands.

### `arx-ca-server`

HTTP API, OCSP, enrollment protocols, and PKI engine.

| Command / flag | Description |
| -------------- | ----------- |
| *(default)* | Start the HTTP server (loads `server.yaml` via Viper) |
| `--config <path>` | Path to `server.yaml` (default: `server.yaml` beside the executable) |
| `service install` | Install and enable the systemd unit (Linux, root) |
| `service uninstall` | Stop, disable, and remove the systemd unit (Linux, root) |

**Run the server:**

```bash
./bin/arx-ca-server
./bin/arx-ca-server --config /etc/arx-ca/server.yaml
```

**systemd integration (Linux only):** `service install` resolves paths dynamically from the running binary:

1. **Executable path** — `os.Executable()` with symlinks resolved (the binary you invoke).
2. **Config path** — `--config` if passed to `service install`, otherwise `server.yaml` beside the executable.
3. **Working directory** — Directory containing the executable.

It writes `/etc/systemd/system/arx-ca-server.service` with `ExecStart=<exe> --config <config>`, creates the `arx-ca` system user if needed, runs `systemctl daemon-reload`, and `systemctl enable --now arx-ca-server`. Ensure the `arx-ca` user can read the binary, `server.yaml`, PKI paths, and any `*_file` secret paths.

```bash
sudo ./bin/arx-ca-server service install
sudo ./bin/arx-ca-server --config /etc/arx-ca/server.yaml service install
sudo ./bin/arx-ca-server service uninstall
```

On Windows and macOS, `service install` / `uninstall` return an error indicating Linux-only support.

### `arx-ca-cli`

Super Admin CLI and terminal UI. Loads `~/.arx/cli.yaml` on every command.

| Command | Description |
| ------- | ----------- |
| `login` | Prompt for username/password, call `/api/v1/auth/login`, save JWT to `~/.arx/config.json` |
| `login --url <base>` | Override `server_url` from `cli.yaml` |
| `ui` | Interactive TUI (requires prior `login`) |
| `ui --url <base>` | Override server URL for the TUI session |
| `util hash <password>` | Print a bcrypt hash suitable for `bootstrap.admin_password_hash` |

**Bootstrapping flow:**

1. Start `arx-ca-server` and confirm health (`GET /api/v1/health`).
2. Optionally customize bootstrap in `server.yaml` (hash only — no plain-text password).
3. Run login against the default bootstrap account:

```bash
./bin/arx-ca-cli login
# Server URL [http://localhost:8080]:
# Username: admin
# Password: (bootstrap password — see Bootstrap admin below)
```

4. Launch the TUI: `./bin/arx-ca-cli ui`

**Generate a bcrypt hash for `server.yaml`:**

```bash
./bin/arx-ca-cli util hash 'MySecureAdminPassword!'
# Output: $2a$10$...
```

Copy the hash into `bootstrap.admin_password_hash` before the first start, or set `ARX_BOOTSTRAP_ADMIN_PASSWORD_HASH`, so the seeded database user matches your chosen password policy.

### `arx-cert-service`

Read-only local agent: inspects certificate stores, installs trust anchors, downloads public certificates. **Never handles private keys from the server.**

| Command | Description |
| ------- | ----------- |
| `local list` | List certs in system, user, and browser stores |
| `local list --store system` | Filter by store: `system`, `user`, `browser` |
| `local view <id>` | Show details by thumbprint or serial |
| `trust install-root --url <base>` | Fetch root CA and install into local trust stores |
| `trust install-intermediate --url <base>` | Install intermediate CA |
| `trust uninstall-root` | Remove installed root CA |
| `trust uninstall-intermediate` | Remove installed intermediate CA |
| `server list --url <base>` | List public certificates from the API |
| `server download --url <base>` | Download a public PEM |
| `server download --kind root -o root.pem` | Kind: `leaf`, `intermediate`, or `root` |
| `server download --serial <hex> -o leaf.pem` | Serial required for `kind=leaf` |

State for trust operations is stored under `~/.arx-cert-service/`. On Windows, `local list` skips stores that require elevation (for example Local Machine `ROOT`) and continues with accessible stores.

```bash
./bin/arx-cert-service local list
./bin/arx-cert-service server list --url http://localhost:8080
./bin/arx-cert-service server download --url http://localhost:8080 --kind root -o root.pem
./bin/arx-cert-service trust install-root --url http://localhost:8080
```

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

Super Admin CLI and agent: see [CLI Reference](#command-line-interface-cli-reference).

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

Copy `.env.example` to `.env` for Compose. For bare-metal, prefer `server.yaml`; use `ARX_*` or `CA_API_*` for overrides or secrets injection (see [Configuration Guide](#configuration-guide)).

| Variable | Default | Description |
| -------- | ------- | ----------- |
| `ARX_SERVER_PORT` | *(from YAML)* | Override `server.port` via Viper |
| `ARX_DATABASE_HOST` | *(from YAML)* | Override `database.host` via Viper |
| `ARX_BOOTSTRAP_ADMIN_PASSWORD_HASH` | *(from YAML)* | Override bootstrap bcrypt hash |
| `CA_API_LISTEN_ADDR` | `:8080` (from YAML) | HTTP listen address (exported from YAML if unset) |
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

| Item | Value |
| ---- | ----- |
| Username | `admin` |
| Default password | `ArxRootCA-Bootstrap-Admin-2026!` |

API login validates this bootstrap password via the embedded verifier in `internal/auth/admin.go`. Separately, `bootstrap.admin_password_hash` in `server.yaml` seeds the application `users` table on first start when it is empty.

**Production:** Generate a new hash with `arx-ca-cli util hash`, set `bootstrap.admin_password_hash` (or `ARX_BOOTSTRAP_ADMIN_PASSWORD_HASH`) before first boot, align the login verifier with your deployment process, and rotate credentials after initial access.

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
