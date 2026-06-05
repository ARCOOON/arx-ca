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
| **Agent renewal daemon** | `arx-agent run` renews local PEM files via `agent.yaml` using the native API or ACMEv2 client mode |
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

## One-line install

Scripts in [`scripts/`](scripts/) download the latest [GitHub Release](https://github.com/ARCOOON/arx-ca/releases) (`arx-<os>-<arch>` and `webui-dist.tar.gz`), install the binary and WebUI assets, and expose `arx` on your PATH. Existing `server.yaml` and `.pki/` in the install directory are preserved on upgrade.

| Scope | Linux / macOS | Windows |
| ----- | ------------- | ------- |
| **User** (default) | `$HOME/.arx`, symlink `$HOME/.local/bin/arx` | `%LOCALAPPDATA%\arx`, User PATH |
| **System** (elevated) | `/opt/arx`, symlink `/usr/local/bin/arx` | `%ProgramFiles%\arx`, Machine PATH |

**Linux / macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/ARCOOON/arx-ca/main/scripts/install.sh | bash
curl -fsSL https://raw.githubusercontent.com/ARCOOON/arx-ca/main/scripts/install.sh | bash -s -- --system   # requires sudo
curl -fsSL https://raw.githubusercontent.com/ARCOOON/arx-ca/main/scripts/uninstall.sh | bash
```

Or from a checkout: `./scripts/install.sh`, `./scripts/install.sh --system`, `./scripts/uninstall.sh`.

**Windows (PowerShell):**

```powershell
# Recommended when execution policy is unknown (typical on locked-down hosts)
powershell -ExecutionPolicy Bypass -NoProfile -Command "irm https://raw.githubusercontent.com/ARCOOON/arx-ca/main/scripts/install.ps1 | iex"

# When RemoteSigned or Unrestricted already allows scripts
irm https://raw.githubusercontent.com/ARCOOON/arx-ca/main/scripts/install.ps1 | iex

.\scripts\install.ps1 -System   # Run as Administrator
.\scripts\uninstall.ps1
```

After install, initialize and start the CA (adjust `--config` for your scope):

```bash
arx server config init --config ~/.arx/server.yaml    # user scope
sudo arx server config init --config /opt/arx/server.yaml
arx server start --config ~/.arx/server.yaml
```

For systemd production deployment, use `arx server setup` or `arx server service install` after installing the binary. See [Production install](#production-install-linux-systemd).

## Quick Start (production, Linux + systemd)

**CA server:** build or download `arx`, then run the interactive installer as root.

```bash
make build
sudo ./bin/arx server setup
```

From a [GitHub Release](https://github.com/ARCOOON/arx-ca/releases):

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
3. Copies the running executable to `<install-dir>/arx`, runs `server config init` if needed, writes `arx.service`, and starts the unit.

After CA install:

```bash
sudo nano /opt/arx/server.yaml   # optional: JWT secret and bootstrap password (auto-secured on first start)
sudo systemctl status arx
journalctl -u arx -f
```

```bash
arx login --url https://ca.example.com
arx ui
```

Non-interactive CA install (IaC): set `service.run_as_user` and `service.install_dir` in `server.yaml`, then `sudo arx server service install --system`. User-scoped installs use `arx server service install --user`. See [Production install](#production-install-linux-systemd).

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

Generate a bcrypt hash for `bootstrap.admin_password`:

```bash
./bin/arx hash 'MySecureAdminPassword!'
```

## Build

Run `make` or `make help` for a self-documented list of targets.

```bash
make build                 # bin/arx and bin/arx-agent (native host)
make build-agent           # bin/arx-agent only
make build-all             # build/* cross binaries + webui-dist.tar.gz
make webui                 # npm install/build + webui-dist.tar.gz
make build-arx-linux-amd64 # single platform under build/
make test
make clean
```

Cross-compiled release binaries use the same names as GitHub releases (for example `build/arx-linux-amd64`, `build/arx-windows-amd64.exe`). The WebUI tarball is `webui-dist.tar.gz` at the repository root.

Without Make:

```bash
mkdir -p bin
VERSION=v1.2.3
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS="-X main.Version=${VERSION} -X main.Commit=${COMMIT} -s -w"
CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o bin/arx ./cmd/arx
CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o bin/arx-agent ./cmd/arx-agent
```

Tagged releases (`v*`) trigger [.github/workflows/release.yml](.github/workflows/release.yml). A `create-release` job runs first: it writes a `## What's Changed` section to `CHANGELOG.md` with bullet lines linking each commit subject to `https://github.com/ARCOOON/arx-ca/commit/<full-sha>` (commits since the previous tag, or all commits on the first release), then creates the GitHub Release with `body_path: CHANGELOG.md` (no `generate_release_notes`). Four build jobs then run in parallel—Linux binaries (amd64/arm64), Windows binaries (amd64), Darwin binaries (amd64/arm64), and a WebUI build from the monorepo `webui/` subdirectory (`npm ci`, `npm run build`) packaged as `webui-dist.tar.gz`. All `arx` / `arx-agent` builds use `CGO_ENABLED=0`, embed the tag and Git commit as `main.Version` and `main.Commit`, and upload assets to the same release via `softprops/action-gh-release` without modifying the release body.

Set local build metadata with Make: `make build VERSION=v1.2.3` (defaults to `v0.0.0-dev`; commit defaults to `git rev-parse --short HEAD` or `unknown`).

### Self-update

Both binaries can replace themselves in place from the latest GitHub release (SemVer comparison, atomic swap via `minio/selfupdate`):

```bash
arx util update
arx-agent update
```

Requires outbound HTTPS to `api.github.com` and `github.com`. Use `sudo` when the binary lives in a protected path (e.g. `/opt/arx/` or `/opt/arx-agent/`). Restart the CA server or `arx-agent` service after updating. See [docs/cli_reference.md](docs/cli_reference.md).

## Configuration

| File | Purpose |
| ---- | ------- |
| `server.yaml` | Server bind, database, CA paths, security, bootstrap, telemetry, optional `service` and `webui` blocks (beside `arx`, or `--config`) |
| `~/.arx/cli.yaml` | CLI defaults: `server_url`, `log_level` (mode `0600`, dir `0700`) |
| `~/.arx/config.json` | Saved JWT session after `arx login` (used by `arx-agent enroll` / `run` when renewing) |
| `~/.arx-cert-service/agent.yaml` | Agent-only renewal daemon config (`protocol: api` or `acme` per cert); see [docs/agent.md](docs/agent.md) |
| `.pki/` | Root/intermediate certs, `ca.json`, step-ca Badger certificate DB |
| `arx.db` | Application SQLite DB (default path relative to `server.yaml` directory) |

```bash
./bin/arx server config init
./bin/arx server start --config /etc/arx/server.yaml
./bin/arx-agent config init
./bin/arx-agent run --config /opt/arx-agent/agent.yaml
```

Each `managed_certs` entry selects **`protocol: api`** (Arx REST API + `arx login`) or **`protocol: acme`** (RFC 8555 client with HTTP-01). Examples: [docs/agent.md](docs/agent.md).

Optional `service` block in `server.yaml` for IaC and `arx server service install`:

```yaml
service:
  run_as_user: arx-ca
  install_dir: /opt/arx   # user installs: set to $HOME/.arx (config init sets this beside the binary)
```

The `ca.max_ttl` field (default `8760h`, one year) caps certificate lifetimes for `POST /api/v1/certificates/issue` and `POST /api/v1/certificates/generate`. On startup, arx-ca synchronizes this limit into step-ca `ca.json` claims so the embedded PKI honors the same maximum.

### Dedicated WebUI server

The REST API and the browser UI run on **separate listeners**. Enable the WebUI with `webui.enabled: true` and place built static assets in `webui.ui_dir` (absolute path; `arx server config init` sets it to `<executable-dir>/ui`, must contain `index.html`).

| Parameter | Default | Description |
| --------- | ------- | ----------- |
| `webui.enabled` | `false` | Start the dedicated WebUI HTTP server |
| `webui.ui_dir` | `/opt/arx/ui` | Static SPA root directory |
| `webui.path_prefix` | `/` | URL path prefix (e.g. `/ui` → console at `https://host:8443/ui/`) |
| `webui.listen_address` | `:8443` | WebUI bind address (independent of `server.port`) |
| `webui.max_body_size` | `2097152` | Max request body size (bytes) |
| `webui.read_timeout` / `write_timeout` | `10s` | Server timeouts |
| `webui.tls.enabled` | `true` | HTTPS for the WebUI listener |
| `webui.tls.cert_file` / `key_file` | (empty) | TLS certificate and key; when omitted or missing, an ephemeral ECDSA cert is generated with SANs (`localhost`, `127.0.0.1`, `::1`, and detected host IPs) |
| `webui.cors.allowed_origins` | `["*"]` | CORS origins for static assets and for API cross-origin calls when WebUI is enabled |
| `webui.cors.allowed_methods` | `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS` | Allowed methods; `*` expands to the full REST set |
| `webui.cors.allowed_headers` | `Authorization`, `Content-Type`, `Accept`, `X-API-Key`, `*` | Allowed request headers; `*` mirrors preflight `Access-Control-Request-Headers` |
| `webui.proxy_api` | `true` | Reverse-proxy `/api/`, `/ocsp`, `/acme/`, `/scep/`, `/certsrv/` on the WebUI listener to the API process (same-origin drop-in UI) |

**`path_prefix` and deployment:** A prefix of `/` serves the SPA at the WebUI listener root. A prefix of `/ui` strips `/ui` before looking up files under `ui_dir`, so operators can colocate the API behind one hostname (port 8080) and the console on another port or path without mixing handlers. Configure your frontend build `base` to match `path_prefix`. Unmatched paths under the prefix fall back to `index.html` for client-side routing.

Example (UI under `/ui` on HTTPS port 8443):

```yaml
webui:
  enabled: true
  ui_dir: /opt/arx/ui
  path_prefix: /ui
  listen_address: ":8443"
  max_body_size: 2097152
  read_timeout: 10s
  write_timeout: 10s
  tls:
    enabled: true
    cert_file: /opt/arx/certs/webui.crt
    key_file: /opt/arx/certs/webui.key
  cors:
    allowed_origins:
      - "*"
    allowed_methods:
      - GET
      - POST
      - PUT
      - PATCH
      - DELETE
      - OPTIONS
    allowed_headers:
      - Authorization
      - Content-Type
      - Accept
      - X-API-Key
      - "*"
  proxy_api: true
```

`arx server config init` writes this block with defaults. For production, set `webui.tls.cert_file` and `key_file` to operator-managed certificates. For local drop-in use, leave paths empty to auto-generate a SAN-bearing ephemeral server certificate. With `proxy_api: true`, the Vue app can call same-origin `/api/v1` on the WebUI port without CORS or a compile-time `VITE_API_BASE_URL`. Client certificates presented to the WebUI HTTPS listener are forwarded to the API on loopback via `X-Forwarded-Client-Cert`, so mTLS endpoints such as `POST /api/v1/certificates/renew` work through the WebUI port.

The Vue 3 management console (`webui/`) uses a flat dark/light theme with consistent corner radii (6px controls, 8px panels; pill-shaped toggle tracks). Tokens are defined in `webui/src/assets/theme.css` and applied via `ui-*` classes in `webui/src/style.css`. Client-side routing runs under the authenticated shell:

| Route | Purpose |
| ----- | ------- |
| `/dashboard` | Health metrics, uptime, and CA backend status |
| `/certificates` | Issued certificate inventory and CSR signing |
| `/acme` | ACME directory URL and enrollment policy |
| `/scep` | SCEP base URL and provisioner status |
| `/settings` | Session, API base URL, and UI preferences |

Develop locally with `cd webui && npm ci && npm run dev`, or ship assets via `npm run build` and `arx server ui download`.

**Automated WebUI install:** `arx server ui download` reads `server.yaml`, fetches `webui-dist.tar.gz` from GitHub (matching the `arx` binary version, or the latest release when the binary is `v0.0.0-dev`), extracts files into `webui.ui_dir`, and sets `webui.enabled: true`. Pass `--version v1.0.2` to download a specific release tag instead. Use `sudo` when the default path is `/opt/arx/ui`.

```bash
arx server config init
sudo arx server ui download
arx server ui download --version v1.0.1
arx server start
```

Environment overrides use the `ARX_` prefix (Viper) and `ARX_AGENT_` for agent daemon settings. For PostgreSQL, prefer `ARX_DATABASE_PASSWORD` or `CA_API_DB_DATA_SOURCE` over storing credentials in `server.yaml`. See [.env.example](.env.example) for Docker Compose.

### PostgreSQL (optional)

Set `database.driver` to `postgres` (or `postgresql`) and provide host, credentials, and `dbname`. When `database.host` is non-empty and `driver` is omitted, PostgreSQL is selected automatically.

## Production install (Linux, systemd)

### CA server (`arx`)

| Command | Description |
| ------- | ----------- |
| `sudo arx server setup` | Interactive wizard (recommended, system scope) |
| `sudo arx server service install --system` | System unit at `/etc/systemd/system/arx.service`, binary `/opt/arx/arx` |
| `arx server service install --user` | User unit at `~/.config/systemd/user/arx.service`, binary `~/.arx/arx` |
| `arx server service uninstall --user` / `sudo ... --system` | Remove unit and install tree for the selected scope |

System defaults: user `arx-ca`, install dir `/opt/arx`. User defaults: install dir `~/.arx`. Flags `--run-as-user` and `--install-dir` override `server.yaml` when set. On Windows, `--system` registers a Windows Service; `--user` creates a logon scheduled task under `%LOCALAPPDATA%\arx`.

### Renewal agent (`arx-agent`)

| Command | Description |
| ------- | ----------- |
| `sudo arx-agent service install` | Copy binary to `/opt/arx-agent`, bootstrap `agent.yaml`, start `arx-agent.service` |
| `sudo arx-agent service uninstall` | Remove unit, install tree, and service user |

Defaults: user `arx-agent`, install dir `/opt/arx-agent`. The unit runs `arx-agent run --config <install-dir>/agent.yaml`.

For **API** renewal entries, run `arx login` on the agent host (or copy credentials). For **ACME** entries, configure `acme_directory_url`, `acme_email`, and HTTP-01 `webroot` or `challenge_listen_port` — no JWT required. Run `arx-agent config init` for a commented template.

`arx server service install` is supported on Linux and Windows. Other platforms return an error. Set the step-ca symmetric key at runtime with `ARX_CA_PASSWORD` (never bcrypt-hashed in `server.yaml`).

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
| `GET` | `/api/v1/ca/chain` | — | Intermediate + Root CA chain PEM download (`ca-chain.crt`) |
| `POST` | `/api/v1/auth/login` | — | Admin JWT (`email` + `password`) |
| `POST` | `/api/v1/certificates/issue` | Service / Admin | Sign CSR (certificate only); archives public PEM |
| `POST` | `/api/v1/certificates/generate` | Service / Admin | Native key generation + sign; returns key PEM and escrows it encrypted at rest |
| `GET` | `/api/v1/certificates/{serial}` | Service / Admin | Archived certificate detail (requestor, PEM, validity, escrow flag) |
| `GET` | `/api/v1/certificates/{serial}/key` | SuperAdmin (JWT) | Decrypt and return escrowed private key PEM |
| `GET` | `/api/v1/certificates/{serial}/bundle` | SuperAdmin (JWT) | Minimal ZIP bundle (`certificate.crt`, `private.key`) |
| `POST` | `/api/v1/certificates/auto` | Admin | Generate ECDSA P-384 key + sign |
| `GET` | `/api/v1/crl` | — | CRL (DER or PEM; alias of `/api/v1/ca/crl`) |
| `POST` | `/api/v1/certificates/renew` | Admin / mTLS | Renew existing certificate |
| `GET` | `/acme/directory` | ACME JWS | ACME directory (when enabled) |

Full table: [docs/api_reference.md](docs/api_reference.md). JSON envelope: `{ "data": …, "error": null | { "message": "…" } }`.

## Bootstrap admin

| Item | Default |
| ---- | ------- |
| Email | `admin@arx.local` |
| Password | `ArxRootCA-Bootstrap-Admin-2026!` |

**Production:** set `bootstrap.admin_password` with `arx hash` before first boot, or supply a plaintext value once — the server auto-bcrypts it and rewrites `server.yaml` on startup. An empty `security.jwt_secret` is also auto-generated and persisted. Rotate credentials after initial access.

**Zero-touch startup:** On first `arx server start`, the server inspects `server.yaml` and automatically:
- Generates and persists a 256-bit `security.jwt_secret` when empty.
- Bcrypt-hashes plaintext values in `bootstrap.admin_password` (cost 12) and rewrites the file with mode `0600`.

YAML comments in `server.yaml` may be lost when the file is rewritten (standard `yaml.v3` marshalling). Database passwords in `server.yaml` are **not** encrypted — SQL drivers require clear-text DSN components. Override PostgreSQL credentials via `ARX_DATABASE_PASSWORD` or the full DSN through `CA_API_DB_DATA_SOURCE` / `ARX_DATABASE_*` environment variables instead of storing secrets on disk.

## Documentation

| Document | Contents |
| -------- | -------- |
| [docs/architecture.md](docs/architecture.md) | Split-binary layout, SQLite/CGO, systemd, DDD layers |
| [docs/cli_reference.md](docs/cli_reference.md) | `arx` and `arx-agent` command reference |
| [docs/agent.md](docs/agent.md) | `agent.yaml`, API vs ACME renewal |
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
| `internal/server/service` | Dual-scope daemon self-install (Linux/Windows) |
| `internal/agent/service` | `arx-agent` systemd self-install |

## License

See the repository license file.
