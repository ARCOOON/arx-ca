# Architecture

This document describes how **arx** is structured in its enterprise form: one static binary, a Cobra command surface, domain-driven internal packages, dual persistence (application SQL + step-ca Badger), and a self-installing Linux deployment path.

## Design goals

1. **Deployability** — Ship one binary with embedded SQLite so operators are not required to run Postgres, Docker, or external installers.
2. **Portability** — Build fully static Linux/Windows/macOS artifacts with `CGO_ENABLED=0` for CI and air-gapped installs.
3. **Compatibility** — Keep step-ca as the signing engine and RFC 8555 ACME for ecosystem tools (Traefik, Caddy, Certbot).
4. **Separation of concerns** — HTTP handlers stay thin; business logic lives in services and the PKI engine; persistence is isolated in `internal/database` and repositories.
5. **Future Web UI** — RESTful, stateless JSON API with a consistent `{ "data", "error" }` envelope.

## Single-binary pattern

All capabilities compile into `cmd/arx`:

```
cmd/arx/main.go
    └── internal/cmd/arx.NewRootCmd()
            ├── server   (API process, config, setup, systemd)
            ├── login    (admin auth → ~/.arx/)
            ├── ui       (admin TUI)
            ├── cert     (certificate admin API)
            ├── util     (helpers)
            ├── hash     (bcrypt; alias of util hash)
            └── agent    (local machine operations)
```

There are no separate `arx-ca-server`, `arx-ca-cli`, or `arx-cert-service` binaries. Historical multi-binary layouts and bash-based installers are not part of the current distribution model.

### Production deployment flow

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

## Cobra command tree

| Top-level | Subcommands | Role |
| --------- | ----------- | ---- |
| `server` | `start`, `config init`, `setup`, `service install\|uninstall` | Run CA API, generate `server.yaml`, guided or scripted systemd install |
| `login` | — | Obtain admin JWT; persist URL in `~/.arx/cli.yaml` and token in `~/.arx/config.json` |
| `ui` | — | Interactive terminal admin UI (requires login) |
| `cert` | `list`, `revoke <serial>` | Authenticated certificate management |
| `util` | `hash <password>` | Administrative helpers |
| `hash` | `<password>` | Top-level alias for `util hash` |
| `agent` | `enroll`, `local`, `trust`, `cert` | Local stores, trust anchors, public certs, optional enroll |

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

`server start` registers routes in `internal/cmd/arx/server_start.go`, mounts ACME at `/acme/` when the ACME provisioner is active, and handles graceful shutdown on `SIGINT`/`SIGTERM`.

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

The release workflow builds four artifacts on tag push `v*`:

- `arx-linux-amd64`, `arx-linux-arm64`
- `arx-windows-amd64.exe`
- `arx-darwin-arm64`

Each step sets `CGO_ENABLED=0` explicitly before `go build -trimpath -ldflags="-s -w" -o … ./cmd/arx`.

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
| `internal/agent` | OS certificate stores, trust installation, enroll helper |
| `internal/server/service` | systemd unit render, binary copy, install/uninstall |

## ACME integration (summary)

When the step-ca config includes an ACME provisioner and `CA_API_ACME_DISABLED` is not `true`:

1. `InitACMEServer` wires `database.NewACMEStore` into the step-ca ACME DB interface.
2. `acmeprotocol.NewFlatLinker` exposes URLs under `/acme/...` (no provisioner name in the path).
3. `acmeprotocol.NewRouter` mounts handlers; the public directory is **`/acme/directory`**.

Challenge validation (**HTTP-01**, **DNS-01**, **TLS-ALPN-01**) runs in `internal/acmeprotocol` via `Validator.ValidateChallenge`. Details: [acme.md](acme.md).

## Configuration and secrets

| Concern | Mechanism |
| ------- | --------- |
| Server config | `server.yaml` beside binary; override with `arx server --config` |
| Env overrides | `ARX_*` via Viper; exported to `CA_API_*` when unset |
| CLI state | `~/.arx/cli.yaml` (`server_url`), `~/.arx/config.json` (JWT) |
| Agent state | `~/.arx-cert-service/` (trust + enroll artifacts) |
| Secrets on disk | `password_file`, `provisioner_password_file`, `database.password_file` via `ResolveSecret` |

JWT signing uses `security.jwt_secret` or `CA_API_JWT_SECRET`. An empty secret triggers one-time generation in development; production deployments should set it explicitly.

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
