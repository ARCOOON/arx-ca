# Architecture

This document describes how **arx** and **arx-agent** are structured: two static binaries sharing internal packages, separate Cobra entrypoints, domain-driven internal packages, dual persistence on the server (application SQL + step-ca Badger), and self-installing Linux deployment paths for both the CA and the renewal agent.

## Design goals

1. **Deployability** — Ship one binary with embedded SQLite so operators are not required to run Postgres, Docker, or external installers.
2. **Portability** — Build fully static Linux/Windows/macOS artifacts with `CGO_ENABLED=0` for CI and air-gapped installs.
3. **Compatibility** — Keep step-ca as the signing engine and RFC 8555 ACME for ecosystem tools (Traefik, Caddy, Certbot).
4. **Separation of concerns** — HTTP handlers stay thin; business logic lives in services and the PKI engine; persistence is isolated in `internal/database` and repositories.
5. **Dedicated Web UI** — Optional static WebUI on a separate listener (`webui` in `server.yaml`); the REST API remains stateless with a `{ "data", "error" }` envelope.

## Split-binary pattern

Control plane and data plane compile from separate entrypoints but share libraries under `internal/`:

```
cmd/arx/main.go
    └── internal/cmd/arx.NewRootCmd()
            ├── server   (API process, config, setup, arx-server systemd)
            ├── login    (admin auth → ~/.arx/)
            ├── ui       (admin TUI)
            ├── cert     (certificate admin API)
            ├── util     (helpers)
            └── hash     (bcrypt; alias of util hash)

cmd/arx-agent/main.go
    └── internal/cmd/arxagent.NewRootCmd()
            ├── daemon / run   (renewal loop)
            ├── enroll         (admin JWT auto-issue)
            ├── local          (OS cert stores)
            ├── trust          (root/intermediate install)
            ├── cert           (public catalog)
            └── service        (arx-agent systemd self-install)
```

`arx-agent` does **not** link `internal/database`, `internal/ca`, `internal/api/handlers`, or `internal/acmeprotocol`, keeping the client binary small (~10 MB vs ~41 MB for `arx` in typical release builds).

### CA server deployment flow

```
Operator runs:  sudo ./arx server setup
                      │
                      ▼
        internal/server/service.Install
                      │
        ┌─────────────┴─────────────┐
        ▼                           ▼
  Copy executable            server config init
  to /opt/arx/arx            → /opt/arx/server.yaml
        │                           │
        └─────────────┬─────────────┘
                      ▼
        Write /etc/systemd/system/arx-server.service
        systemctl enable --now arx-server
                      │
                      ▼
        ExecStart=/opt/arx/arx server start --config /opt/arx/server.yaml
```

Install parameters can be declared in `server.yaml` under `service` for Infrastructure as Code, or supplied at install time via `--run-as-user` and `--install-dir`.

### Renewal agent deployment flow

```
Operator runs:  sudo ./arx-agent service install
                      │
                      ▼
        internal/agent/service.Install
                      │
        ┌─────────────┴─────────────┐
        ▼                           ▼
  Copy executable            EnsureAgentConfigFile
  to /opt/arx-agent/arx-agent → /opt/arx-agent/agent.yaml
        │                           │
        └─────────────┬─────────────┘
                      ▼
        Write /etc/systemd/system/arx-agent.service
        systemctl enable --now arx-agent
                      │
                      ▼
        ExecStart=/opt/arx-agent/arx-agent run --config /opt/arx-agent/agent.yaml
```

Defaults: user `arx-agent`, install dir `/opt/arx-agent`. Override with `--run-as-user` and `--install-dir` at install time.

## Cobra command tree

| Top-level | Subcommands | Role |
| --------- | ----------- | ---- |
| `server` | `start`, `config init`, `setup`, `service install\|uninstall` | Run CA API, generate `server.yaml`, guided or scripted systemd install |
| `login` | — | Obtain admin JWT; persist URL in `~/.arx/cli.yaml` and token in `~/.arx/config.json` |
| `ui` | — | Interactive terminal admin UI (requires login) |
| `cert` | `list`, `revoke <serial>` | Authenticated certificate management |
| `util` | `hash <password>` | Administrative helpers |
| `hash` | `<password>` | Top-level alias for `util hash` |

**`arx-agent` binary:**

| Top-level | Subcommands | Role |
| --------- | ----------- | ---- |
| `daemon` / `run` | `--config` | Renewal loop over `agent.yaml` (`run` is the systemd entrypoint) |
| `enroll` | `--domain`, `--ttl`, `--url` | Issue and store a leaf cert via admin JWT |
| `local` | `list`, `view <id>` | Inspect OS certificate stores |
| `trust` | `install-root`, `install-intermediate`, uninstall variants | Local trust anchors |
| `cert` | `list`, `download` | Public certificate catalog (no JWT) |
| `service` | `install`, `uninstall` | Self-install `arx-agent.service` on Linux |

### Server command lifecycle

```
arx server config init [--force]
        │
        ▼
   server.yaml (beside binary or --config path, mode 0600)
        │
        ▼
arx server start [--config path]
        │
        ├── database.Open + Migrate + EnsureBootstrapAdmin
        ├── ca.InitCA (step-ca / .pki)
        ├── pkiEngine.InitACMEServer()
        └── net/http.ServeMux (Go 1.22 method patterns)
```

`server` uses `PersistentPreRunE` to load `server.yaml` via Viper for `server start` only. These subcommands **skip** server config init at startup because they create or manage deployment artifacts:

- `server config *`
- `server setup`
- `server service *`

### Self-installing binary (Linux)

`internal/server/service` implements `arx server service install|uninstall` (build tag `linux`). `server setup` calls the same `Install` path after an interactive prompt.

| Step | `install` | `uninstall` |
| ---- | --------- | ----------- |
| Privilege | root (`EUID == 0`) | root |
| Filesystem | Create `<install-dir>` (default `/opt/arx`), copy running binary to `<install-dir>/arx` | `os.RemoveAll(installDir)` |
| Config | Run `<install-dir>/arx server config init` if `server.yaml` missing | — |
| Identity | `useradd --system` for service user (default `arx-ca`) | `userdel` |
| systemd | Write `arx-server.service`, `daemon-reload`, `enable`, `restart` | stop, disable, remove unit |

`server start` registers routes in `internal/cmd/arx/server_start.go`, mounts ACME at `/acme/` when the ACME provisioner is active, optionally starts the dedicated WebUI server (`internal/server/webui.go`), and handles graceful shutdown on `SIGINT`/`SIGTERM` for both listeners.

### Dedicated WebUI server

When `webui.enabled` is `true`, the CA API process also runs an **isolated** `net/http` server for static frontend assets. It does not share the API listener (`server.host` / `server.port`); operators bind it separately (default `:8443`) and can enable TLS with dedicated certificate paths.

```
                    ┌──────────────────────────────────────────────┐
  Browser / CDN ──► │  WebUI listener (webui.listen_address)        │
                    │  /api/* proxy (optional) + PathPrefix + UIDir │
                    └──────────────────────────────────────────────┘
                                        │ loopback proxy (proxy_api)
                                        ▼
                    ┌─────────────────────────────────────┐
  Clients / CLI ──► │  API listener (server.host:port)     │
                    │  /api/v1/*, /acme/, /ocsp, …         │
                    └─────────────────────────────────────┘
```

When `webui.proxy_api` is `true` (default), the WebUI listener reverse-proxies `/api/`, `/ocsp`, `/acme/`, `/scep/`, and `/certsrv/` to the API process on loopback. The Vue app then uses same-origin `/api/v1` without cross-origin calls. Set `webui.proxy_api: false` only when a separate ingress terminates API and UI on one host without the built-in proxy.

The SPA (`webui/`) mounts authenticated routes inside `AppShell` (collapsible sidebar, top bar with roles and logout): `/dashboard` (health and inventory summary), `/certificates` (list + CSR issue modal), `/acme` and `/scep` (read-only status from `GET /api/v1/acme/status` and `GET /api/v1/scep/status`), and `/settings` (session and API client metadata). Axios attaches the admin JWT from Pinia on every protected call; `401` responses trigger logout except for `POST /auth/login`.

| Setting | Default | Role |
| ------- | ------- | ---- |
| `webui.enabled` | `false` | Turn on the dedicated WebUI server |
| `webui.ui_dir` | `/opt/arx/ui` | Directory of built SPA assets (`index.html` required) |
| `webui.path_prefix` | `/` | URL path under which the UI is served (see below) |
| `webui.listen_address` | `:8443` | Bind address for the WebUI only |
| `webui.max_body_size` | `2097152` (2 MiB) | Request body cap via `http.MaxBytesReader` |
| `webui.read_timeout` | `10s` | `http.Server.ReadTimeout` |
| `webui.write_timeout` | `10s` | `http.Server.WriteTimeout` |
| `webui.tls.enabled` | `true` | HTTPS on the WebUI listener (`tls.Listen` with optional client certs) |
| `webui.tls.cert_file` / `key_file` | (empty) | TLS material; when missing, ephemeral ECDSA P-256 with SANs (`DNS:localhost`, `IP:127.0.0.1`, `IP:::1`, host interface addresses) |
| `webui.cors.allowed_origins` | `["*"]` | CORS `Access-Control-Allow-Origin` on the WebUI listener **and** on the API listener when `webui.enabled` is `true` |
| `webui.cors.allowed_methods` | REST verbs + `OPTIONS` | Allowed methods; entry `*` expands to the full REST set |
| `webui.cors.allowed_headers` | `Authorization`, `Content-Type`, … | Allowed headers; entry `*` mirrors preflight `Access-Control-Request-Headers` |
| `webui.proxy_api` | `true` | Loopback reverse proxy for API paths on the WebUI listener (drop-in same-origin UI) |

**Path prefix and deployment architecture**

`path_prefix` defines the **base URL path** for the SPA, not a filesystem path. The WebUI mux strips this prefix before resolving files under `ui_dir`:

| `path_prefix` | User opens | Static file served from |
| ------------- | ---------- | ------------------------ |
| `/` | `https://ca.example:8443/` | `ui_dir/index.html` |
| `/ui` | `https://ca.example:8443/ui/` | `ui_dir/index.html` (after strip) |
| `/admin` | `https://ca.example:8443/admin/app` | `ui_dir/app` or SPA fallback to `index.html` |

Use a non-root prefix when the same host also terminates other paths (reverse proxy, shared ingress) or when operators want the API on `/` and the console under `/ui`. The frontend build must set its router `base` / `homepage` to match `path_prefix` so client-side routes resolve correctly.

Requests that do not match a file under `ui_dir` receive **SPA fallback** (`index.html`) so deep links work. Middleware applies configured CORS and `max_body_size` before the file handler.

When the WebUI is enabled, the **API listener** applies the same `webui.cors` policy for direct cross-origin API access (for example UI on `https://ui.example` and API on `https://ca.example:8080` with `webui.proxy_api: false`). Preflight `OPTIONS` requests return `200 OK` with configured `Access-Control-Allow-*` headers. With `proxy_api` enabled (default), the browser uses same-origin `/api/v1` on the WebUI port and CORS is not required for routine dashboard traffic.

Startup logs include the effective URL, for example: `WebUI server starting url=https://0.0.0.0:8443/ui` when `path_prefix` is `/ui` and TLS is enabled.

Relative `ui_dir` and TLS paths resolve against the directory containing `server.yaml`, consistent with `database.path`.

**mTLS through the WebUI proxy:** The WebUI TLS stack uses `tls.RequestClientCert` (optional client certificates). When a client presents a certificate during the WebUI TLS handshake, the reverse proxy encodes the leaf PEM in `X-Forwarded-Client-Cert` (Envoy `Cert=` URL-encoded form) on the loopback request to the API. The API `ClientCertValidator` accepts this header only from `127.0.0.1` / `::1`, validating the certificate the same way as a direct mTLS connection to the API listener.

## Systemd self-healing (`ExecStartPre`)

The generated unit (`internal/server/service/unit.go`) runs the CA as an unprivileged service user with filesystem hardening (`ProtectSystem=full`, `PrivateTmp`, `ProtectHome`, `NoNewPrivileges`).

Administrators often edit `server.yaml` with `sudo`, which can leave root-owned files or loose permissions that prevent the service user from reading config or executing the binary. To recover automatically on **every** start, the unit uses privileged `ExecStartPre=+` hooks:

```ini
ExecStartPre=+/usr/bin/chown -R {{RunAsUser}}:{{RunAsUser}} {{InstallDir}}
ExecStartPre=+/usr/bin/chmod 600 {{ConfigPath}}
ExecStartPre=+/usr/bin/chmod 700 {{ExecPath}}
ExecStart={{ExecPath}} server start --config {{ConfigPath}}
```

The `+` prefix allows these commands to run as root even when `User=` is set for the main process. Together with install-time `chown` of the install tree, this **self-heals** permission drift without a separate maintenance script.

## Pure-Go SQLite and `CGO_ENABLED=0`

### Application database (SQLite default, PostgreSQL optional)

Configured under `database` in `server.yaml`:

| Setting | Default | Meaning |
| ------- | ------- | ------- |
| `driver` | `sqlite` | `sqlite` or `postgres` / `postgresql` |
| `path` | `arx.db` | SQLite file relative to the directory containing `server.yaml` |
| `host`, `user`, `dbname`, … | (empty) | Used when driver is PostgreSQL |

Implementation: `internal/database/db.go` opens **`modernc.org/sqlite`** (pure Go) or **`pgx`** for Postgres. SQLite connections enable WAL mode and cap the connection pool (default max 4 open, 2 idle) so ACME and API handlers do not serialize on a single connection.

**Stored in the application DB:**

- Admin `users` (seeded from `bootstrap` on first run)
- ACME accounts, nonces, orders, authorizations, challenges (`internal/database/acme_store.go`)

This is the “zero-dependency” datastore: no separate DB container is required for a full CA deployment.

### Why `CGO_ENABLED=0` matters

The default SQLite driver is implemented entirely in Go (`modernc.org/sqlite`), not `github.com/mattn/go-sqlite3` (which requires CGO and a C toolchain on each target platform).

Setting **`CGO_ENABLED=0`** for release builds yields:

| Benefit | Explanation |
| ------- | ------------- |
| **Static binaries** | No libc/sqlite `.so` linkage; simpler copying to minimal containers and air-gapped hosts |
| **Cross-compilation** | `GOOS`/`GOARCH` builds from Linux CI without cross-compilers |
| **Reproducible CI** | Same flags in `Makefile` (`build-linux`, `build-windows`) and `.github/workflows/release.yml` |

On tag push `v*`, `.github/workflows/release.yml` runs a `create-release` job first, then four parallel build jobs. `create-release` checks out full history (`fetch-depth: 0`), resolves the previous tag with `git describe --tags --abbrev=0 HEAD^`, writes `CHANGELOG.md` with a `## What's Changed` header and bullet lines formatted as `* <subject> ([<short>](https://github.com/ARCOOON/arx-ca/commit/<full>))` for commits since that tag (or all commits when no prior tag exists), and creates the GitHub Release with `body_path: CHANGELOG.md` without `generate_release_notes`. Each Go build job sets `CGO_ENABLED=0` and uses `go build -ldflags="-X main.Version=<tag> -X main.Commit=<sha> -s -w" -o …` for `cmd/arx` or `cmd/arx-agent`. The WebUI job runs `npm ci` and `npm run build` in `webui/` and publishes `webui-dist.tar.gz` (contents of `webui/dist/`). Build jobs depend on `create-release` and upload assets only (`tag_name` + `files`); they do not set `generate_release_notes` or `append_body`, avoiding race conditions on the release body.

| Job | Role |
| --- | --- |
| `create-release` | `## What's Changed` changelog with commit hyperlinks; creates release body once |
| `build-linux` | `arx-linux-amd64`, `arx-linux-arm64`, `arx-agent-linux-amd64`, `arx-agent-linux-arm64` |
| `build-windows` | `arx-windows-amd64.exe`, `arx-agent-windows-amd64.exe` |
| `build-darwin` | `arx-darwin-amd64`, `arx-darwin-arm64`, `arx-agent-darwin-amd64`, `arx-agent-darwin-arm64` |
| `build-webui` | `webui-dist.tar.gz` |

Extract the WebUI tarball into `webui.ui_dir` (for example `/opt/arx/ui`): `tar -xzf webui-dist.tar.gz -C /opt/arx/ui`, or run `arx server ui download` to download the matching release asset, extract, and enable `webui` in `server.yaml` automatically.

PostgreSQL deployments still benefit from static binaries; only the **server** needs network access to Postgres at runtime.

### PKI / certificate database (step-ca Badger)

Configured in generated `.pki/config/ca.json` (derived from `ca.root_path`). step-ca persists issued certificates in embedded **BadgerDB** under the PKI data directory. On corruption, `internal/ca` can truncate and reopen the Badger value log (`badger_heal.go`).

**Do not confuse** `arx.db` (application SQL) with Badger paths under `.pki/` (PKI engine).

### Driver selection logic

`DatabaseConfig.EffectiveDriver()` (`internal/config/server.go`):

- Explicit `driver: sqlite` → SQLite
- Explicit `driver: postgres` → PostgreSQL
- Empty `driver` + non-empty `host` → PostgreSQL (backward compatibility)
- Empty `driver` + empty `host` → default `sqlite`

## Layered application structure (DDD)

```
┌─────────────────────────────────────────────────────────────┐
│  Handlers (internal/api/handlers)                           │
│  HTTP parsing, status codes, JSON envelope                  │
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│  Middleware (internal/api/middleware)                         │
│  Logging, JWT/API-key auth, RBAC permissions                  │
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│  PKI engine (internal/ca)                                   │
│  step-ca authority, issuance, revocation, ACME/SCEP/NDES    │
└───────────────────────────┬─────────────────────────────────┘
                            │
        ┌───────────────────┴───────────────────┐
        ▼                                       ▼
┌───────────────────┐               ┌───────────────────────┐
│  Application DB   │               │  step-ca cert DB      │
│  (internal/       │               │  Badger under .pki/   │
│   database)       │               │  via ca.json          │
│  users, ACME      │               │  issued certs, etc.   │
└───────────────────┘               └───────────────────────┘
```

| Package | Responsibility |
| ------- | -------------- |
| `internal/api/handlers` | REST endpoints, OCSP, public read-only catalog |
| `internal/api/middleware` | Authentication, authorization, request logging |
| `internal/auth` | JWT, API keys, bootstrap admin, RBAC permissions |
| `internal/ca` | step-ca wrapper, provisioners, enrollment protocols |
| `internal/acmeprotocol` | Flat `/acme/*` URLs, JWS middleware, challenge validators |
| `internal/database` | SQL migrations, user seeding, ACME state store |
| `internal/config` | `server.yaml`, `~/.arx/cli.yaml`, env bridging |
| `internal/cli` | Admin HTTP client, login flow, Bubble Tea TUI |
| `internal/agent` | OS certificate stores, trust installation, renewal daemon |
| `internal/cmd/arxagent` | Cobra surface for the `arx-agent` binary |
| `internal/server` | Dedicated WebUI static server (`webui.go`) |
| `internal/server/service` | `arx-server` systemd unit render, binary copy, install/uninstall |
| `internal/agent/service` | `arx-agent` systemd unit render, binary copy, install/uninstall |

## ACME integration (summary)

When the step-ca config includes an ACME provisioner and `CA_API_ACME_DISABLED` is not `true`:

1. `InitACMEServer` wires `database.NewACMEStore` into the step-ca ACME DB interface.
2. `acmeprotocol.NewFlatLinker` exposes URLs under `/acme/...` (no provisioner name in the path).
3. `acmeprotocol.NewRouter` mounts handlers; the public directory is **`/acme/directory`**.

Challenge validation (**HTTP-01**, **DNS-01**, **TLS-ALPN-01**) runs in `internal/acmeprotocol` via `Validator.ValidateChallenge`. Details: [acme.md](acme.md).

## Configuration and secrets

| Concern | Mechanism |
| ------- | --------- |
| Server config | `server.yaml` beside binary; override with `arx server --config`; optional `webui` block for static UI |
| Env overrides | `ARX_*` via Viper; exported to `CA_API_*` when unset |
| CLI state | `~/.arx/cli.yaml` (`server_url`), `~/.arx/config.json` (JWT) |
| Agent state | `~/.arx-cert-service/` (trust + enroll artifacts) |
| Secrets on disk | `password_file`, `provisioner_password_file`, `database.password_file` via `ResolveSecret` |

### Auto-securing startup (`HealServerConfig`)

Immediately after `server.yaml` is parsed, `InitServerConfig` runs config auto-healing before the API binds:

| Check | Action |
| ----- | ------ |
| Empty `security.jwt_secret` | Generate 32 random bytes, base64-encode, persist to `server.yaml` |
| Plaintext admin password | Detect values in `security.initial_admin_password` or `bootstrap.admin_password_hash` that lack a `$2a$` / `$2b$` / `$2y$` prefix; bcrypt-hash at cost 12 and persist |
| Database credentials | **Not modified** — connection strings remain clear-text for SQL drivers; use `ARX_DATABASE_PASSWORD` or `CA_API_DB_DATA_SOURCE` env overrides in production |

When any field is secured, the updated struct is marshalled back to YAML and written with mode `0600`. Inline YAML comments may be lost on rewrite. Runtime env overrides (`CA_API_JWT_SECRET`, `ARX_SECURITY_JWT_SECRET`) apply after the file is loaded and are not written back to disk.

JWT signing uses `security.jwt_secret` or `CA_API_JWT_SECRET`. Production deployments should still set secrets explicitly or rely on the one-time auto-generation during first boot.

## HTTP routing

Go 1.22 `net/http` pattern routing is used at the API layer (no third-party router). ACME internally uses `go-chi/chi` inside the mounted `/acme/` subtree.

Public and authenticated routes are registered in `runServer()`; enrollment protocols are mounted conditionally when enabled on the PKI engine.

## Security model (RBAC)

| Role | Credential | Typical use |
| ---- | ---------- | ----------- |
| Admin | JWT from `POST /api/v1/auth/login` | UI, `arx cert`, templates |
| Service account | `X-API-Key` or Bearer API key | Automation, issuance, renewals |
| ACME account | ACME account key JWS | Automated DV certificates |
| Provisioner token | JWK / OIDC in request body | End-entity or SSH flows |

Permissions are enforced per-route in middleware (`internal/auth/roles.go`).

## Related documents

- [cli_reference.md](cli_reference.md) — Operator commands and session files
- [acme.md](acme.md) — ACME directory and challenge types
- [../README.md](../README.md) — Quick start and build targets
