# Architecture

This document describes how **arx** is structured after the single-binary migration: one executable, a Cobra command surface, domain-driven internal packages, and a dual-database persistence model.

## Design goals

1. **Deployability** — Ship one static binary with embedded SQLite for operators who do not want Postgres or Docker.
2. **Compatibility** — Keep step-ca as the signing engine and RFC 8555 ACME for ecosystem tools (Traefik, Caddy, Certbot).
3. **Separation of concerns** — HTTP handlers stay thin; business logic lives in services and the PKI engine; persistence is isolated in repositories/database packages.
4. **Future Web UI** — RESTful, stateless JSON API with a consistent `{ "data", "error" }` envelope.

## Single-binary pattern

All capabilities compile into `cmd/arx`:

```
cmd/arx/main.go
    └── internal/cmd/arx.NewRootCmd()
            ├── server   (API process)
            ├── login    (admin auth)
            ├── ui       (admin TUI)
            ├── cert     (certificate admin API)
            ├── util     (helpers)
            ├── hash     (bcrypt; same as util hash)
            └── agent    (local machine operations)
```

There are no separate `arx-ca-server`, `arx-ca-cli`, or `arx-cert-service` binaries. Systemd units and documentation refer to the same `arx` executable with subcommands (for example `ExecStart=/usr/local/bin/arx server start --config /etc/arx-ca/server.yaml`).

## Cobra command tree

| Top-level | Subcommands | Role |
| --------- | ----------- | ---- |
| `server` | `start`, `config init`, `service install\|uninstall` | Run CA API, generate `server.yaml`, manage systemd |
| `login` | — | Obtain admin JWT; persist URL and token |
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
   server.yaml (beside binary, mode 0600)
        │
        ▼
arx server start [--config path]
        │
        ├── database.Open + Migrate + SeedInitialAdmin
        ├── ca.InitCA (step-ca / .pki)
        ├── pkiEngine.InitACMEServer()
        └── net/http.ServeMux (Go 1.22 method patterns)
```

`server` uses `PersistentPreRunE` to load `server.yaml` via Viper, except for `server config *` and `server service *` (config generation and systemd do not require an existing server config).

`server start` registers routes in `internal/cmd/arx/server_start.go`, mounts ACME at `/acme/` when the ACME provisioner is active, and handles graceful shutdown on `SIGINT`/`SIGTERM`.

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

## Database architecture

arx uses **two persistence layers** with different roles.

### Application database (SQLite default, PostgreSQL optional)

Configured under `database` in `server.yaml`:

| Setting | Default | Meaning |
| ------- | ------- | ------- |
| `driver` | `sqlite` | `sqlite` or `postgres` / `postgresql` |
| `path` | `arx.db` | SQLite file relative to the directory containing `server.yaml` |
| `host`, `user`, `dbname`, … | (empty / template values) | Used when driver is PostgreSQL |

Implementation: `internal/database/db.go` opens `modernc.org/sqlite` (pure Go, **CGO_ENABLED=0** friendly) or `pgx` for Postgres. SQLite connections use WAL mode and a small connection pool so ACME and API handlers can run concurrently.

**Stored here:**

- Admin `users` table (seeded from `bootstrap` on first run)
- ACME accounts, nonces, orders, authorizations, challenges (`internal/database/acme_store.go`)

This is the database referenced in “zero-dependency” deployments: no separate DB container is required.

### PKI / certificate database (step-ca Badger)

Configured in generated `.pki/config/ca.json` (derived from `ca.root_path`). step-ca persists issued certificates and related metadata in embedded **BadgerDB** under the PKI data directory. On corruption, `internal/ca` can truncate and reopen the Badger value log (`badger_heal.go`).

**Stored here:**

- Certificate issuance history as managed by step-ca
- Authority configuration consumed by the signing engine

Do not confuse `arx.db` (application) with Badger paths under `.pki/` (PKI engine).

### Driver selection logic

`DatabaseConfig.EffectiveDriver()` (`internal/config/server.go`):

- Explicit `driver: sqlite` → SQLite
- Explicit `driver: postgres` → PostgreSQL
- Empty `driver` + non-empty `host` → PostgreSQL (backward compatibility)
- Empty `driver` + empty `host` → default `sqlite`

## ACME integration (summary)

When the step-ca config includes an ACME provisioner and `CA_API_ACME_DISABLED` is not `true`:

1. `InitACMEServer` wires `database.NewACMEStore` into the step-ca ACME DB interface.
2. `acmeprotocol.NewFlatLinker` exposes URLs under `/acme/...` (no provisioner name in the path).
3. `acmeprotocol.NewRouter` mounts handlers; the server strips `/acme` and serves the directory at **`/acme/directory`**.

Challenge validation (HTTP-01, DNS-01, TLS-ALPN-01) is implemented in `internal/acmeprotocol` and invoked from the custom challenge handler. Details: [acme.md](acme.md).

## Configuration and secrets

| Concern | Mechanism |
| ------- | --------- |
| Server config | `server.yaml` beside binary; override with `arx server --config` |
| Env overrides | `ARX_*` via Viper; exported to `CA_API_*` when unset |
| CLI state | `~/.arx/cli.yaml`, `~/.arx/config.json` |
| Agent state | `~/.arx-cert-service/` (trust + enroll artifacts) |
| Secrets on disk | `password_file`, `provisioner_password_file`, `database.password_file` resolved by `ResolveSecret` |

JWT signing uses `security.jwt_secret` or `CA_API_JWT_SECRET`. An empty secret triggers one-time generation in development; production deployments should set it explicitly.

## External integrations (optional)

| Integration | Configuration surface |
| ----------- | --------------------- |
| AWS / GCP KMS, Vault | `internal/config` crypto backends (CA keys) |
| OIDC provisioner | `CA_API_OIDC_*` / `ca.json` |
| OpenTelemetry | `telemetry` block or `OTEL_*` |
| Remote step-ca | `ca.stepca_url` |

The default **SoftCAS** layout keeps keys on disk under `.pki/`.

## HTTP routing

Go 1.22 `net/http` pattern routing is used exclusively (no third-party router at the API layer). ACME internally uses `go-chi/chi` inside the mounted `/acme/` subtree.

Public and authenticated routes are registered in `runServer()`; enrollment protocols are mounted conditionally when enabled on the PKI engine.

## Security model (RBAC)

| Role | Credential | Typical use |
| ---- | ---------- | ----------- |
| Admin | JWT from `POST /api/v1/auth/login` | UI, service accounts, templates |
| Service account | `X-API-Key` or Bearer API key | Automation, issuance, renewals |
| ACME account | ACME account key JWS | Automated DV certificates |
| Provisioner token | JWK / OIDC in request body | End-entity or SSH flows |

Permissions are enforced per-route in middleware (`internal/auth/roles.go`).

## Related documents

- [cli_reference.md](cli_reference.md) — Operator commands and session files
- [acme.md](acme.md) — ACME directory and challenge types
- [../README.md](../README.md) — Quick start and build targets
