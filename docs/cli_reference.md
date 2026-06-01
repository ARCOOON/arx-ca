# CLI Reference

The **arx** binary provides server lifecycle, admin authentication, certificate management, utilities, and a local **agent** for trust stores and public certificate access. Commands that talk to the API load `~/.arx/cli.yaml` on startup (`withCLIConfig`).

## Stateful CLI design

Admin and agent commands resolve the CA base URL and JWT from local files instead of requiring `--url` on every invocation.

### Configuration files

| File | Format | Permissions | Contents |
| ---- | ------ | ----------- | -------- |
| `~/.arx/cli.yaml` | YAML | file `0600`, dir `0700` | `server_url`, `log_level` |
| `~/.arx/config.json` | JSON | file `0600`, dir `0700` | `server_url`, `token`, `token_type`, `expires_at`, `username` |
| `~/.arx-cert-service/config.json` | JSON | (agent) | `api_url` — updated when agent commands pass `--url` |

`cli.yaml` is auto-created on first CLI use with default `log_level: info`. **`server_url` is not pre-populated**; set it with a successful login or an explicit flag.

### URL resolution order

`internal/cli/runtime.ResolveServerURL` applies this precedence:

1. Non-empty `--url` / `-u` flag (when `PersistFlag` is true for commands like `ui` and `cert`, the URL is written back to config files)
2. `server_url` from `~/.arx/cli.yaml` (Viper)
3. `server_url` from `~/.arx/config.json` (saved at login)
4. `api_url` from `~/.arx-cert-service/config.json` (only when `UseAgentState` is true — agent subcommands)

If no URL is found:

```text
Server URL not configured. Please run 'arx login --url <URL>' first, or provide the --url flag.
```

### Authentication flow

```bash
arx login --url https://ca.example.com
```

On success:

1. `POST /api/v1/auth/login` returns a JWT.
2. Token and metadata are written to `~/.arx/config.json`.
3. `server_url` is updated in `~/.arx/cli.yaml` via `SetCLIServerURL`.

Example success output:

```text
Logged in as admin. Token saved to /home/user/.arx/config.json
Expires: 2026-06-02T12:00:00Z
Roles: admin
```

Commands that require authentication (`ui`, `cert`, `agent enroll`) call `NewAuthenticatedClient` and fail with `not logged in; run arx login first` when no token is present.

### Login flags

| Flag | Description |
| ---- | ----------- |
| `--url`, `-u` | Server base URL (required for first login if not already configured) |
| `--username` | Skip username prompt |
| `--password` | Skip password prompt (automation only; avoid on shared systems) |

Interactive login (TTY) prompts for server URL (if not fully non-interactive), username, and password. Default bootstrap credentials are documented in [README.md](../README.md#bootstrap-admin).

---

## Global conventions

- Run `arx <command> --help` for subcommand-specific flags.
- Server subcommands accept persistent `--config` for `server.yaml` location.
- Paths in examples assume `bin/arx` from `make build`.

---

## `arx server`

Manage the CA API process and its configuration.

### `arx server config init`

Writes a default `server.yaml` beside the executable (or next to the path given by `--config`).

```bash
./bin/arx server config init
./bin/arx server config init --force
```

Success log:

```text
Configuration successfully generated at /path/to/bin/server.yaml. Please edit it before starting the server.
```

If the file exists and `--force` is omitted:

```text
Configuration already exists at ... Use --force to overwrite.
```

### `arx server start`

Starts the HTTP server. Requires an existing `server.yaml` (run `config init` first).

```bash
./bin/arx server start
./bin/arx server --config /etc/arx-ca/server.yaml start
```

Startup log (when ACME is enabled):

```text
ACME enabled; directory available at /acme/directory
arx server listening on :8080
```

Missing config:

```text
No configuration file found at ... Run 'arx server config init' to generate one.
```

### `arx server service install` / `uninstall`

Linux only, root required. Installs `/etc/systemd/system/arx-ca-server.service` with:

```ini
ExecStart=<arx-binary> server start --config <server.yaml>
```

Success:

```text
arx CA server systemd service installed and started.
Ensure the arx-ca user can read the executable, configuration file, and any paths referenced in server.yaml ...
```

---

## `arx login`

Authenticate as the bootstrap (or seeded) admin user.

```bash
arx login --url http://localhost:8080
arx login --url https://ca.example.com --username admin --password 'secret'
```

---

## `arx ui`

Bubble Tea terminal UI for CA administration. Requires prior `login`.

```bash
arx ui
arx ui --url http://localhost:8080   # overrides and persists URL when flag is set
```

---

## `arx cert`

Authenticated certificate operations against the admin API.

### `arx cert list`

```bash
arx cert list
arx cert list -u https://ca.example.com
```

Example output:

```text
SERIAL                          STATUS   SUBJECT                         NOT AFTER
a1b2c3d4...                     active   CN=example.local                2026-07-01
```

Empty inventory:

```text
No issued certificates found.
```

### `arx cert revoke <serial>`

```bash
arx cert revoke a1b2c3d4e5f6 --reason "key compromise"
```

Example output:

```text
Revoked a1b2c3d4e5f6 at 2026-06-01T15:04:05Z
```

---

## `arx hash` / `arx util hash`

Generate a bcrypt hash for `bootstrap.admin_password_hash` in `server.yaml`.

```bash
arx hash 'MySecureAdminPassword!'
arx util hash 'MySecureAdminPassword!'
```

Example output (one line):

```text
$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
```

---

## `arx agent`

Local certificate inspection, trust installation, public catalog download, and optional enrollment. **Does not download private keys from the server** except via `agent enroll`, which calls `POST /api/v1/certificates/auto` with the admin JWT and stores key material locally under `~/.arx-cert-service/enrolled/`.

### `arx agent enroll`

```bash
arx agent enroll --domain example.local --ttl 720h
arx agent enroll --domain example.local -u http://localhost:8080
```

Success:

```text
Enrolled certificate for example.local (serial ...).
Files saved under ~/.arx-cert-service/enrolled/
```

### `arx agent local list`

```bash
arx agent local list
arx agent local list --store system --store user
```

Example output:

```text
ID          STORE    LOCATION    SUBJECT              NOT AFTER
ABCD1234... system   LocalMachine CN=Example CA        2030-01-01
```

### `arx agent local view <id>`

```bash
arx agent local view ABCD1234
```

Example output:

```text
Thumbprint: ABCD1234...
Store:      system (LocalMachine)
Subject:    CN=Example
Issuer:     CN=Example Root
Serial:     01
Valid:      2024-01-01 — 2030-01-01
Is CA:      false
```

### `arx agent trust`

| Command | Description |
| ------- | ----------- |
| `install-root --url <base>` | Fetch root CA PEM and install locally |
| `install-intermediate --url <base>` | Install intermediate CA |
| `uninstall-root` | Remove installed root |
| `uninstall-intermediate` | Remove installed intermediate |

`--url` on install commands is persisted to agent config when set.

Success:

```text
Root CA installed into local trust stores.
State saved under ~/.arx-cert-service/
```

### `arx agent cert list`

Public read-only catalog (no admin JWT).

```bash
arx agent cert list --url http://localhost:8080
```

Example output:

```text
SERIAL    SUBJECT              NOT AFTER    REVOKED
01        CN=example.local     2026-07-01   no
```

### `arx agent cert download`

```bash
arx agent cert download --url http://localhost:8080 --kind root -o root.pem
arx agent cert download --serial <hex> --kind leaf -o leaf.pem
```

| Flag | Values |
| ---- | ------ |
| `--kind` | `leaf` (default), `intermediate`, `root` |
| `--serial` | Required for `leaf` |
| `-o`, `--output` | Output PEM path |

---

## Command tree (reference)

```text
arx
├── server
│   ├── start
│   ├── config
│   │   └── init [--force]
│   └── service
│       ├── install
│       └── uninstall
├── login [--url] [--username] [--password]
├── ui [--url]
├── cert
│   ├── list [--url]
│   └── revoke <serial> [--url] [--reason]
├── util
│   └── hash <password>
├── hash <password>
└── agent
    ├── enroll --domain <name> [--ttl] [--url]
    ├── local
    │   ├── list [--store ...]
    │   └── view <id>
    ├── trust
    │   ├── install-root [--url]
    │   ├── install-intermediate [--url]
    │   ├── uninstall-root
    │   └── uninstall-intermediate
    └── cert
        ├── list [--url]
        └── download [--url] [--kind] [--serial] [-o]
```

---

## Typical operator workflows

### First-time server + admin

```bash
make build
./bin/arx server config init
# edit bin/server.yaml (jwt_secret, bootstrap hash)
./bin/arx server start
./bin/arx login --url http://localhost:8080
./bin/arx ui
```

### Issue and list certificates

Use the TUI or API; then:

```bash
./bin/arx cert list
```

### Trust the local CA on a workstation

```bash
./bin/arx agent trust install-root --url http://localhost:8080
./bin/arx agent trust install-intermediate --url http://localhost:8080
```

### Enroll a leaf cert on the same machine (admin JWT)

```bash
./bin/arx login --url http://localhost:8080
./bin/arx agent enroll --domain app.internal --ttl 2160h
```

---

## See also

- [architecture.md](architecture.md) — persistence and package layout
- [acme.md](acme.md) — automated enrollment via ACME
- [../README.md](../README.md) — quick start and API table
