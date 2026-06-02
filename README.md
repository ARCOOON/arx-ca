# arx — Arx Certificate Authority

**arx** is a single-binary Certificate Authority platform built on the [step-ca SDK](https://github.com/smallstep/certificates). It ships one executable that runs the HTTP API, admin CLI, and local certificate agent. The default stack needs no external database server: application state (admin users, ACME accounts, orders, and challenges) lives in **pure-Go SQLite** (`modernc.org/sqlite`). PKI material and issued-certificate metadata use embedded storage under `.pki/`.

## Highlights

| Capability | Description |
| ---------- | ----------- |
| **Single binary** | `arx` replaces separate server, CLI, and agent binaries |
| **Zero-dependency datastore** | SQLite user/ACME store beside `server.yaml` (no Postgres required) |
| **REST API** | X.509 issuance, revocation, OCSP, templates, SSH CA, enrollment status |
| **ACMEv2 (RFC 8555)** | Directory at `/acme/directory` with HTTP-01, DNS-01, and TLS-ALPN-01 |
| **Enrollment protocols** | SCEP and NDES when configured in `ca.json` |
| **Stateful CLI** | `arx login --url` persists base URL and JWT under `~/.arx/` |

Works with standard ACME clients and reverse-proxy integrations (**Traefik**, **Caddy**, **Certbot**, and any RFC 8555 client). See [docs/acme.md](docs/acme.md).

## Architecture at a glance

```
┌──────────────────────────────────────────────────────────────────────────┐
│  arx server start                                                        │
│  REST API · OCSP · ACME (/acme/*) · SCEP · NDES · JWT/API-key RBAC       │
│  server.yaml (beside binary) · SQLite arx.db · step-ca PKI under .pki/   │
└───────────────────────────────┬──────────────────────────────────────────┘
                                │ HTTPS / HTTP
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
  arx login / ui          arx cert list            arx agent local list
  arx cert revoke         (admin API)              arx agent trust …
  ~/.arx/cli.yaml         JWT from login           ~/.arx-cert-service/
  ~/.arx/config.json
```

Deeper design notes: [docs/architecture.md](docs/architecture.md).

## Quick Start (Linux production)

Requires **Go 1.22+** (see `go.mod`) or a prebuilt `bin/arx`. On Linux with systemd, use the interactive installer (recommended):

```bash
make build
sudo ./bin/arx server setup
```

The wizard copies the binary to `/opt/arx`, bootstraps `server.yaml`, registers `arx-server`, and starts the service. Edit `/opt/arx/server.yaml` before production use (JWT secret, bootstrap password hash).

Non-interactive or IaC installs can set the optional `service` block in `server.yaml` (see [Configuration](#configuration)) and run `sudo ./bin/arx server service install`, or pass `--run-as-user` and `--install-dir` to override the file.

## Quick Start (development)

```bash
# Build
make build

# Generate server.yaml next to the binary (default: bin/server.yaml)
./bin/arx server config init

# Edit bin/server.yaml if needed (JWT secret, bootstrap hash, listen port)

# Start the CA API (bootstraps PKI under .pki/ on first run)
./bin/arx server start
```

Optional production hardening before `server start`:

```bash
export CA_API_JWT_SECRET="$(openssl rand -base64 32)"
export OTEL_SDK_DISABLED=true
```

Verify the server:

```bash
curl -s http://localhost:8080/api/v1/health | jq .
curl -s http://localhost:8080/api/v1/ca/root
```

Log in and open the admin TUI:

```bash
./bin/arx login --url http://localhost:8080
# Username: admin
# Password: ArxRootCA-Bootstrap-Admin-2026!

./bin/arx ui
```

Generate a bcrypt hash for `bootstrap.admin_password_hash` in `server.yaml`:

```bash
./bin/arx hash 'MySecureAdminPassword!'
# or: ./bin/arx util hash 'MySecureAdminPassword!'
```

Full CLI reference: [docs/cli_reference.md](docs/cli_reference.md).

## Build

```bash
make build          # bin/arx
make build-linux    # Linux amd64, CGO_ENABLED=0
make build-windows  # bin/arx.exe
make build-fips     # GOEXPERIMENT=boringcrypto
make test
make clean
```

Without Make:

```bash
mkdir -p bin
go build -trimpath -ldflags="-s -w" -o bin/arx ./cmd/arx
```

## Configuration

| File | Purpose |
| ---- | ------- |
| `server.yaml` | Server bind, database, CA paths, security, bootstrap, telemetry, optional `service` install settings (beside the `arx` binary, or `--config`) |
| `~/.arx/cli.yaml` | CLI defaults: `server_url`, `log_level` (mode `0600`, dir `0700`) |
| `~/.arx/config.json` | Saved JWT session after `arx login` |
| `.pki/` | Root/intermediate certs, `ca.json`, step-ca Badger certificate DB |
| `arx.db` | Application SQLite DB (default path relative to `server.yaml` directory) |

Initialize server config explicitly (recommended):

```bash
./bin/arx server config init
./bin/arx server config init --force   # overwrite existing file
./bin/arx server start --config /etc/arx-ca/server.yaml
```

Optional `service` block (included in defaults from `config init`) for Infrastructure as Code and `server service install`:

```yaml
service:
  run_as_user: arx-ca
  install_dir: /opt/arx
```

CLI flags `--run-as-user` and `--install-dir` override these values when set. When flags are omitted and the fields are empty, install falls back to `arx-ca` and `/opt/arx`.

Environment overrides use the `ARX_` prefix (Viper). Legacy `CA_API_*` and `OTEL_*` variables are populated from YAML when unset. Copy [.env.example](.env.example) for Docker Compose.

### PostgreSQL (optional)

Set `database.driver` to `postgres` (or `postgresql`) and provide host, credentials, and `dbname` in `server.yaml`. When `database.host` is non-empty and `driver` is omitted, PostgreSQL is selected automatically for backward compatibility.

## Production install (Linux, systemd)

The `arx` binary is **self-installing**: it copies itself under the install directory, bootstraps `server.yaml`, registers a hardened `arx-server` systemd unit (with self-healing `ExecStartPre` permission fixes), and starts the service. No separate bash installer is required.

```bash
# Interactive wizard (recommended)
make build
sudo ./bin/arx server setup

# Non-interactive: flags override server.yaml service.*, then defaults
sudo ./bin/arx server service install
sudo ./bin/arx server service install --run-as-user arx-ca --install-dir /opt/arx

# Remove unit, install directory, and the service user
sudo ./bin/arx server service uninstall
```

| Source | `run_as_user` | `install_dir` |
| ------ | ------------- | ------------- |
| CLI flags (when set) | `--run-as-user` | `--install-dir` |
| `server.yaml` `service` block | `service.run_as_user` | `service.install_dir` |
| Fallback | `arx-ca` | `/opt/arx` |

After install, edit `/opt/arx/server.yaml` (JWT secret, bootstrap password hash). Check status with `systemctl status arx-server` and logs with `journalctl -u arx-server -f`.

Non-Linux platforms return an error from `server service install`.

## Docker Compose

```bash
cp .env.example .env
# Set CA_API_JWT_SECRET in .env

make docker-build
make docker-up
```

The container listens on **8080**; PKI data is mounted at `./data/arx-ca`.

## API overview

| Method | Path | Auth | Description |
| ------ | ---- | ---- | ----------- |
| `GET` | `/api/v1/health` | — | Health and runtime metrics |
| `GET` | `/api/v1/ca/root` | — | Root CA PEM |
| `GET` | `/api/v1/ca/crl` | — | CRL (DER; `?pem` for PEM) |
| `GET` | `/api/v1/public/certificates` | — | Public certificate inventory |
| `POST` | `/ocsp` | — | OCSP responder |
| `POST` | `/api/v1/auth/login` | — | Admin JWT |
| `POST` | `/api/v1/certificates/auto` | Service / Admin | Generate key + sign |
| `POST` | `/api/v1/certificates/revoke` | Service / Admin | Revoke by serial |
| `GET` | `/api/v1/certificates` | Service / Admin | List certificates |
| `GET` | `/acme/directory` | ACME JWS | ACME directory (when enabled) |
| `*` | `/scep/...` | SCEP | SCEP enrollment (when enabled) |

JSON envelope: `{ "data": …, "error": null | { "message": "…" } }`.

## Bootstrap admin

| Item | Value |
| ---- | ----- |
| Username | `admin` |
| Default password | `ArxRootCA-Bootstrap-Admin-2026!` |

API login checks the bootstrap verifier in code. `bootstrap.admin_password_hash` in `server.yaml` seeds the `users` table when empty on first start. **Production:** set a custom hash before first boot (`arx hash`), rotate credentials after initial access, and set `CA_API_JWT_SECRET` (or `security.jwt_secret`) explicitly.

## Documentation

| Document | Contents |
| -------- | -------- |
| [docs/architecture.md](docs/architecture.md) | Single-binary layout, Cobra command tree, DDD layers, database design |
| [docs/cli_reference.md](docs/cli_reference.md) | Stateful CLI, commands, example output |
| [docs/acme.md](docs/acme.md) | ACME directory, challenges, reverse-proxy integration |

## Project layout

| Path | Purpose |
| ---- | ------- |
| `cmd/arx` | Unified `arx` entrypoint |
| `internal/cmd/arx` | Cobra command definitions |
| `internal/api` | HTTP handlers and middleware |
| `internal/ca` | PKI engine (step-ca), ACME/SCEP/NDES |
| `internal/acmeprotocol` | RFC 8555 routing, JWS, challenge validation |
| `internal/database` | SQLite/PostgreSQL application store and ACME persistence |
| `internal/config` | Viper YAML for server and CLI |
| `internal/cli` | Admin API client, login, TUI |
| `internal/agent` | Local stores, trust install, public cert download |
| `internal/server/service` | Self-installing systemd deployment (`server service install`) |

## License

See the repository license file.
