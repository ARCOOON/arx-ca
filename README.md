# arx — Arx Certificate Authority

**arx** is a single-binary Certificate Authority platform built on the [step-ca SDK](https://github.com/smallstep/certificates). One executable runs the HTTP API, stateful administration CLI, interactive setup wizard, and local certificate agent. The default deployment needs no external database: application state (admin users, ACME accounts, orders, and challenges) lives in **pure-Go SQLite** (`modernc.org/sqlite`). PKI material and issued-certificate metadata use step-ca storage under `.pki/`.

## Features

| Capability | Description |
| ---------- | ----------- |
| **Single binary** | One `arx` executable for server, CLI, and agent — no separate server/CLI/agent builds |
| **Zero-dependency datastore** | SQLite (`arx.db`) beside `server.yaml`; optional PostgreSQL for enterprise deployments |
| **Self-installation** | `arx server setup` (interactive) or `arx server service install` copies the binary, bootstraps config, and registers systemd — no bash installer |
| **Built-in ACMEv2** | RFC 8555 directory at `/acme/directory` with **HTTP-01**, **DNS-01**, and **TLS-ALPN-01** validation |
| **REST API** | X.509 issuance, revocation, OCSP, templates, SSH CA, enrollment status |
| **Enrollment protocols** | SCEP and NDES when configured in `ca.json` |
| **Stateful CLI** | `arx login --url` persists the CA base URL in `~/.arx/cli.yaml` and the JWT in `~/.arx/config.json` |
| **Cross-platform CI** | GitHub Actions release pipeline builds static binaries with `CGO_ENABLED=0` |

Works with standard ACME clients and reverse proxies (**Traefik**, **Caddy**, **Certbot**, and any RFC 8555 client). See [docs/acme.md](docs/acme.md).

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

Deeper design: [docs/architecture.md](docs/architecture.md). Operator commands: [docs/cli_reference.md](docs/cli_reference.md).

## Quick Start (production, Linux + systemd)

**Recommended:** build or download `arx`, then run the interactive installer as root.

```bash
# From source
make build
sudo ./bin/arx server setup
```

From a [GitHub Release](https://github.com/your-org/arx-ca/releases) artifact:

```bash
chmod +x arx-linux-amd64
sudo ./arx-linux-amd64 server setup
```

The wizard:

1. Confirms installation as a systemd service (decline to exit without changes).
2. Prompts for service user (default `arx-ca`) and install directory (default `/opt/arx`).
3. Copies the running executable to `<install-dir>/arx`, runs `server config init` if needed, writes `arx-server.service`, and starts the unit.

After install:

```bash
# Harden before production (JWT secret, bootstrap password hash)
sudo nano /opt/arx/server.yaml

sudo systemctl status arx-server
journalctl -u arx-server -f
```

Log in from any workstation that can reach the API:

```bash
arx login --url https://ca.example.com
# Email:    admin@arx.local
# Password: ArxRootCA-Bootstrap-Admin-2026!

arx ui
```

Non-interactive install (IaC): set `service.run_as_user` and `service.install_dir` in `server.yaml`, then `sudo arx server service install`. See [Production install](#production-install-linux-systemd).

## Quick Start (development)

```bash
make build
./bin/arx server config init
# Edit bin/server.yaml (jwt_secret, bootstrap hash) if needed
./bin/arx server start
```

Verify:

```bash
curl -s http://localhost:8080/api/v1/health | jq .
curl -s http://localhost:8080/api/v1/ca/root
```

Authenticate and open the admin TUI:

```bash
./bin/arx login --url http://localhost:8080
./bin/arx ui
```

Generate a bcrypt hash for `bootstrap.admin_password_hash`:

```bash
./bin/arx hash 'MySecureAdminPassword!'
```

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
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/arx ./cmd/arx
```

Tagged releases (`v*`) trigger [.github/workflows/release.yml](.github/workflows/release.yml), which cross-compiles `arx` for Linux (amd64/arm64), Windows (amd64), and Darwin (arm64) with `CGO_ENABLED=0` and attaches binaries to a GitHub Release.

## Configuration

| File | Purpose |
| ---- | ------- |
| `server.yaml` | Server bind, database, CA paths, security, bootstrap, telemetry, optional `service` block (beside the `arx` binary, or `--config`) |
| `~/.arx/cli.yaml` | CLI defaults: `server_url`, `log_level` (mode `0600`, dir `0700`) |
| `~/.arx/config.json` | Saved JWT session after `arx login` |
| `.pki/` | Root/intermediate certs, `ca.json`, step-ca Badger certificate DB |
| `arx.db` | Application SQLite DB (default path relative to `server.yaml` directory) |

```bash
./bin/arx server config init
./bin/arx server config init --force
./bin/arx server start --config /etc/arx/server.yaml
```

Optional `service` block (written by `config init`) for IaC and `server service install`:

```yaml
service:
  run_as_user: arx-ca
  install_dir: /opt/arx
```

CLI flags `--run-as-user` and `--install-dir` override `server.yaml` when set.

Environment overrides use the `ARX_` prefix (Viper). Legacy `CA_API_*` and `OTEL_*` variables are populated from YAML when unset. See [.env.example](.env.example) for Docker Compose.

### PostgreSQL (optional)

Set `database.driver` to `postgres` (or `postgresql`) and provide host, credentials, and `dbname`. When `database.host` is non-empty and `driver` is omitted, PostgreSQL is selected automatically.

## Production install (Linux, systemd)

| Command | Description |
| ------- | ----------- |
| `sudo arx server setup` | Interactive wizard (recommended) |
| `sudo arx server service install` | Non-interactive self-install |
| `sudo arx server service uninstall` | Remove unit, install tree, and service user |

Resolution order for install paths: CLI flags → `server.yaml` `service` block → defaults (`arx-ca`, `/opt/arx`).

The `arx-server` unit includes `ExecStartPre=+` hooks that re-apply ownership and file modes on every start so root-edited `server.yaml` does not block the service user. Details: [docs/architecture.md](docs/architecture.md#systemd-self-healing-execstartpre).

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
| `POST` | `/api/v1/auth/login` | — | Admin JWT (`email` + `password`) |
| `POST` | `/api/v1/certificates/auto` | Service / Admin | Generate key + sign |
| `POST` | `/api/v1/certificates/revoke` | Service / Admin | Revoke by serial |
| `GET` | `/api/v1/certificates` | Service / Admin | List certificates |
| `GET` | `/acme/directory` | ACME JWS | ACME directory (when enabled) |
| `*` | `/scep/...` | SCEP | SCEP enrollment (when enabled) |

JSON envelope: `{ "data": …, "error": null | { "message": "…" } }`.

## Bootstrap admin

Seeded into the `users` table on first start when no row exists for `bootstrap.admin_email`.

| Item | Default |
| ---- | ------- |
| Email | `admin@arx.local` |
| Password | `ArxRootCA-Bootstrap-Admin-2026!` |

**Production:** set `bootstrap.admin_password_hash` with `arx hash` before first boot, set `security.jwt_secret` (or `CA_API_JWT_SECRET`), rotate credentials after initial access.

## Documentation

| Document | Contents |
| -------- | -------- |
| [docs/architecture.md](docs/architecture.md) | Single-binary layout, SQLite/CGO, systemd, DDD layers |
| [docs/cli_reference.md](docs/cli_reference.md) | All command groups, session files, workflows |
| [docs/acme.md](docs/acme.md) | ACME directory, HTTP-01 / DNS-01 / TLS-ALPN-01 |
| [docs/api_reference.md](docs/api_reference.md) | HTTP endpoint reference |

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
| `internal/server/service` | Self-installing systemd deployment |

## License

See the repository license file.
