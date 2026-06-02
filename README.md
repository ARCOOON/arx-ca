# arx — Arx Certificate Authority

**arx** is a Certificate Authority platform built on the [step-ca SDK](https://github.com/smallstep/certificates). The project ships two static binaries:

| Binary | Role |
| ------ | ---- |
| **`arx`** | Control plane — HTTP API, admin CLI, setup wizard, systemd server install |
| **`arx-agent`** | Data plane — renewal daemon, local trust stores, public cert access on client nodes |

The default server deployment needs no external database: application state (admin users, ACME accounts, orders, and challenges) lives in **pure-Go SQLite** (`modernc.org/sqlite`). PKI material and issued-certificate metadata use step-ca storage under `.pki/`.

## Features

| Capability | Description |
| ---------- | ----------- |
| **Split binaries** | `arx` (~server/CLI) and `arx-agent` (~renewal agent) share internal packages but compile separately for smaller client deployments |
| **Zero-dependency datastore** | SQLite (`arx.db`) beside `server.yaml`; optional PostgreSQL for enterprise deployments |
| **Self-installation** | `arx server setup` or `arx server service install` for the CA; `arx-agent service install` for client renewal daemons |
| **Built-in ACMEv2** | RFC 8555 directory at `/acme/directory` with **HTTP-01**, **DNS-01**, and **TLS-ALPN-01** validation |
| **REST API** | X.509 issuance, revocation, OCSP, templates, SSH CA, enrollment status |
| **Enrollment protocols** | SCEP and NDES when configured in `ca.json` |
| **Stateful CLI** | `arx login --url` persists the CA base URL in `~/.arx/cli.yaml` and the JWT in `~/.arx/config.json` |
| **Agent renewal daemon** | `arx-agent run` monitors local PEM files and renews certificates before expiry via `agent.yaml` |
| **Cross-platform CI** | GitHub Actions release pipeline builds both binaries with `CGO_ENABLED=0` |

Works with standard ACME clients and reverse proxies (**Traefik**, **Caddy**, **Certbot**, and any RFC 8555 client). See [docs/acme.md](docs/acme.md).

## Architecture at a glance

```
┌──────────────────────────────────────────────────────────────────────────┐
│  arx server start  (control plane — bin/arx)                             │
│  REST API · OCSP · ACME (/acme/*) · SCEP · NDES · JWT/API-key RBAC       │
│  server.yaml · SQLite arx.db · step-ca PKI under .pki/                   │
└───────────────────────────────┬──────────────────────────────────────────┘
                                │ HTTPS / HTTP
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
  arx login / ui          arx cert list            arx-agent trust …
  arx cert revoke         (admin API)              arx-agent local list
  ~/.arx/cli.yaml         JWT from login           arx-agent run (daemon)
  ~/.arx/config.json                               ~/.arx-cert-service/
```

Deeper design: [docs/architecture.md](docs/architecture.md). Operator commands: [docs/cli_reference.md](docs/cli_reference.md).

## Quick Start (production, Linux + systemd)

**CA server:** build or download `arx`, then run the interactive installer as root.

```bash
make build
sudo ./bin/arx server setup
```

From a [GitHub Release](https://github.com/your-org/arx-ca/releases):

```bash
chmod +x arx-linux-amd64
sudo ./arx-linux-amd64 server setup
```

**Renewal agent on client nodes:**

```bash
chmod +x arx-agent-linux-amd64
sudo ./arx-agent-linux-amd64 service install
# Edit /opt/arx-agent/agent.yaml — add managed_certs
sudo systemctl status arx-agent
```

Authenticate once (from any machine with network access to the CA):

```bash
arx login --url https://ca.example.com
```

The wizard (server):

1. Confirms installation as a systemd service (decline to exit without changes).
2. Prompts for service user (default `arx-ca`) and install directory (default `/opt/arx`).
3. Copies the running executable to `<install-dir>/arx`, runs `server config init` if needed, writes `arx-server.service`, and starts the unit.

After CA install:

```bash
sudo nano /opt/arx/server.yaml   # JWT secret, bootstrap password hash
sudo systemctl status arx-server
journalctl -u arx-server -f
```

```bash
arx login --url https://ca.example.com
arx ui
```

Non-interactive CA install (IaC): set `service.run_as_user` and `service.install_dir` in `server.yaml`, then `sudo arx server service install`. See [Production install](#production-install-linux-systemd).

## Quick Start (development)

```bash
make build-all
./bin/arx server config init
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
make build            # bin/arx (control plane)
make build-agent      # bin/arx-agent (data plane)
make build-all        # both binaries
make build-linux      # Linux amd64 arx, CGO_ENABLED=0
make build-linux-agent
make test
make clean
```

Without Make:

```bash
mkdir -p bin
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/arx ./cmd/arx
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/arx-agent ./cmd/arx-agent
```

Tagged releases (`v*`) trigger [.github/workflows/release.yml](.github/workflows/release.yml), which cross-compiles **both** `arx` and `arx-agent` for Linux (amd64/arm64), Windows (amd64), and Darwin (arm64) with `CGO_ENABLED=0` and attaches all artifacts to a GitHub Release.

## Configuration

| File | Purpose |
| ---- | ------- |
| `server.yaml` | Server bind, database, CA paths, security, bootstrap, telemetry, optional `service` block (beside `arx`, or `--config`) |
| `~/.arx/cli.yaml` | CLI defaults: `server_url`, `log_level` (mode `0600`, dir `0700`) |
| `~/.arx/config.json` | Saved JWT session after `arx login` (used by `arx-agent enroll` / `run` when renewing) |
| `~/.arx-cert-service/agent.yaml` | Renewal daemon: `check_interval`, `renew_threshold`, `managed_certs` |
| `.pki/` | Root/intermediate certs, `ca.json`, step-ca Badger certificate DB |
| `arx.db` | Application SQLite DB (default path relative to `server.yaml` directory) |

```bash
./bin/arx server config init
./bin/arx server start --config /etc/arx/server.yaml
./bin/arx-agent run --config /opt/arx-agent/agent.yaml
```

Optional `service` block in `server.yaml` for IaC and `arx server service install`:

```yaml
service:
  run_as_user: arx-ca
  install_dir: /opt/arx
```

Environment overrides use the `ARX_` prefix (Viper) and `ARX_AGENT_` for agent daemon settings. See [.env.example](.env.example) for Docker Compose.

### PostgreSQL (optional)

Set `database.driver` to `postgres` (or `postgresql`) and provide host, credentials, and `dbname`. When `database.host` is non-empty and `driver` is omitted, PostgreSQL is selected automatically.

## Production install (Linux, systemd)

### CA server (`arx`)

| Command | Description |
| ------- | ----------- |
| `sudo arx server setup` | Interactive wizard (recommended) |
| `sudo arx server service install` | Non-interactive self-install → `arx-server.service` |
| `sudo arx server service uninstall` | Remove unit, install tree, and service user |

Defaults: user `arx-ca`, install dir `/opt/arx`. CLI flags `--run-as-user` and `--install-dir` override `server.yaml` when set.

### Renewal agent (`arx-agent`)

| Command | Description |
| ------- | ----------- |
| `sudo arx-agent service install` | Copy binary to `/opt/arx-agent`, bootstrap `agent.yaml`, start `arx-agent.service` |
| `sudo arx-agent service uninstall` | Remove unit, install tree, and service user |

Defaults: user `arx-agent`, install dir `/opt/arx-agent`. The unit runs `arx-agent run --config <install-dir>/agent.yaml`.

Before production renewal, run `arx login` on the agent host (or copy credentials) and configure `managed_certs` in `agent.yaml`.

Non-Linux platforms return an error from `service install` on both binaries.

## Docker Compose

```bash
cp .env.example .env
make docker-build
make docker-up
```

The container runs `arx server start` on **8080**; PKI data is mounted at `./data/arx-ca`.

## API overview

| Method | Path | Auth | Description |
| ------ | ---- | ---- | ----------- |
| `GET` | `/api/v1/health` | — | Health and runtime metrics |
| `GET` | `/api/v1/ca/root` | — | Root CA PEM |
| `POST` | `/api/v1/auth/login` | — | Admin JWT (`email` + `password`) |
| `POST` | `/api/v1/certificates/auto` | Service / Admin | Generate key + sign |
| `GET` | `/acme/directory` | ACME JWS | ACME directory (when enabled) |

Full table: [docs/api_reference.md](docs/api_reference.md). JSON envelope: `{ "data": …, "error": null | { "message": "…" } }`.

## Bootstrap admin

| Item | Default |
| ---- | ------- |
| Email | `admin@arx.local` |
| Password | `ArxRootCA-Bootstrap-Admin-2026!` |

**Production:** set `bootstrap.admin_password_hash` with `arx hash` before first boot, set `security.jwt_secret`, rotate credentials after initial access.

## Documentation

| Document | Contents |
| -------- | -------- |
| [docs/architecture.md](docs/architecture.md) | Split-binary layout, SQLite/CGO, systemd, DDD layers |
| [docs/cli_reference.md](docs/cli_reference.md) | `arx` and `arx-agent` command reference |
| [docs/acme.md](docs/acme.md) | ACME directory, challenge types |
| [docs/api_reference.md](docs/api_reference.md) | HTTP endpoint reference |

## Project layout

| Path | Purpose |
| ---- | ------- |
| `cmd/arx` | Control-plane entrypoint (`arx`) |
| `cmd/arx-agent` | Data-plane entrypoint (`arx-agent`) |
| `internal/cmd/arx` | Cobra commands for server and admin CLI |
| `internal/cmd/arxagent` | Cobra commands for renewal agent and local tools |
| `internal/api` | HTTP handlers and middleware |
| `internal/ca` | PKI engine (step-ca), ACME/SCEP/NDES |
| `internal/database` | SQLite/PostgreSQL application store |
| `internal/agent` | Local stores, trust, renewal daemon |
| `internal/server/service` | `arx-server` systemd self-install |
| `internal/agent/service` | `arx-agent` systemd self-install |

## License

See the repository license file.
