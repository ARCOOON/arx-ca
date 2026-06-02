# CLI Reference

The **arx** binary is the unified operator interface: server lifecycle, admin authentication, certificate management, utilities, and a local **agent** for trust stores and public certificate access. Commands that call the admin API load `~/.arx/cli.yaml` on startup via `withCLIConfig`.

## Stateful CLI design

Admin commands resolve the CA base URL and JWT from local files instead of requiring `--url` on every invocation.

### Configuration files

| File | Format | Permissions | Contents |
| ---- | ------ | ----------- | -------- |
| `~/.arx/cli.yaml` | YAML | file `0600`, dir `0700` | `server_url`, `log_level` |
| `~/.arx/config.json` | JSON | file `0600`, dir `0700` | `server_url`, `token`, `token_type`, `expires_at`, `email` |
| `~/.arx-cert-service/config.json` | JSON | (agent) | `api_url` — updated when agent commands pass `--url` |
| `~/.arx-cert-service/agent.yaml` | YAML | file `0600`, dir `0700` | Daemon renewal loop: `check_interval`, `renew_threshold`, `managed_certs` |

`cli.yaml` is created on first CLI use with default `log_level: info`. **`server_url` is not pre-populated** until you log in or pass `--url` on a command that persists it.

### How `arx login --url` persists state

On successful login (`internal/cli/login/login.go`):

1. `POST /api/v1/auth/login` with `email` and `password` returns a JWT.
2. Token and metadata are written to **`~/.arx/config.json`** (`config.Save`).
3. **`server_url` is written to `~/.arx/cli.yaml`** via `config.SetCLIServerURL` (mode `0600`).

Subsequent commands read `server_url` from Viper (`~/.arx/cli.yaml`) before falling back to `config.json`.

### URL resolution order

`internal/cli/runtime.ResolveServerURL` applies this precedence:

1. Non-empty `--url` / `-u` flag (when `PersistFlag` is true, the URL is written to `cli.yaml` and `config.json`, and optionally agent config)
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

Example success output:

```text
Logged in as admin@arx.local. Token saved to /home/user/.arx/config.json
Expires: 2026-06-02T12:00:00Z
Roles: SuperAdmin
```

Commands that require authentication (`ui`, `cert`, `agent enroll`, `agent daemon`) call `NewAuthenticatedClient` and fail with `not logged in; run arx login first` when no token is present.

### Login flags

| Flag | Description |
| ---- | ----------- |
| `--url`, `-u` | Server base URL; **persisted to `~/.arx/cli.yaml` on successful login** |
| `--email` | Admin email (skips prompt; default bootstrap: `admin@arx.local`) |
| `--password` | Admin password (skips prompt; automation only) |

Interactive login (TTY) may prompt for server URL (if not fully non-interactive), email, and password.

---

## Global conventions

- Run `arx <command> --help` for subcommand-specific flags.
- `arx server` accepts persistent `--config` for `server.yaml` location (except `config`, `setup`, and `service` subcommands).
- Examples use `bin/arx` after `make build`.

---

## `arx server`

Manage the CA API process, configuration, interactive installation, and systemd lifecycle.

Persistent flag (all subcommands except those that skip config init):

| Flag | Description |
| ---- | ----------- |
| `--config` | Path to `server.yaml` (default: `server.yaml` beside the executable) |

### `arx server config init`

Writes a default `server.yaml` beside the executable (or next to the path given by `--config`).

```bash
./bin/arx server config init
./bin/arx server config init --force
```

| Flag | Description |
| ---- | ----------- |
| `--force` | Overwrite an existing configuration file |

Success log:

```text
Configuration successfully generated at /path/to/bin/server.yaml. Please edit it before starting the server.
```

### `arx server setup`

Linux only. **Must run as root** (`sudo`). Interactive wizard for the **self-installing binary** pattern: copies the running executable, bootstraps `server.yaml`, registers `arx-server`, and starts the service. Declining the first prompt exits without changes.

```bash
sudo ./bin/arx server setup
```

Prompts (press Enter for defaults):

1. `Install ARX Certificate Authority as a systemd service? [Y/n]:`
2. `Service User [default: arx-ca]:`
3. `Install Directory [default: /opt/arx]:`

Uses the same install logic as `arx server service install` (in-process, not a subprocess). Success output matches `service install`.

If not root:

```text
server setup must be executed as root
```

### `arx server start`

Starts the HTTP server. Requires an existing `server.yaml` (run `config init` first).

```bash
./bin/arx server start
./bin/arx server --config /opt/arx/server.yaml start
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

### `arx server service install`

Linux only. **Must run as root** (`sudo`). Non-interactive self-install: copies the running `arx` executable to `<install-dir>/arx`, bootstraps configuration, applies ownership, writes a hardened `arx-server` systemd unit (with `ExecStartPre` permission self-heal), and starts the service.

```bash
sudo ./bin/arx server service install
sudo ./bin/arx server service install --run-as-user arx-ca --install-dir /opt/arx
```

| Flag | Description |
| ---- | ----------- |
| `--run-as-user` | Overrides `service.run_as_user` in `server.yaml` when set |
| `--install-dir` | Overrides `service.install_dir` in `server.yaml` when set |

Resolution when a flag is omitted: `server.yaml` `service` block, then `arx-ca` / `/opt/arx`.

Install steps:

1. Ensure the service account exists (`id`, else `useradd --system --no-create-home`).
2. Create install directory mode `0700`.
3. Copy current binary to `<install-dir>/arx` mode `0700`.
4. Run `<install-dir>/arx server config init` when `server.yaml` is absent.
5. `chown -R` install tree to the service user; `server.yaml` mode `0600`.
6. Write `/etc/systemd/system/arx-server.service`.
7. `systemctl daemon-reload`, `enable arx-server`, `restart arx-server`.

Example success output:

```text
arx CA server installed and started.
Service:  arx-server
Binary:   /opt/arx/arx
Config:   /opt/arx/server.yaml
Edit server.yaml (JWT secret, bootstrap password hash) before production use.
```

### `arx server service uninstall`

Linux only. **Must run as root.** Stops and disables `arx-server`, removes the unit file, deletes the install directory, and runs `userdel` for the service user.

```bash
sudo ./bin/arx server service uninstall
sudo ./bin/arx server service uninstall --install-dir /opt/arx --run-as-user arx-ca
```

Example success output:

```text
arx CA server uninstalled.
```

---

## `arx login`

Authenticate as the bootstrap (or seeded) admin user.

```bash
arx login --url http://localhost:8080
arx login --url https://ca.example.com --email admin@arx.local --password 'secret'
```

After success, use `arx ui` or `arx cert` without repeating `--url` unless you target a different CA.

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

### `arx cert revoke <serial>`

```bash
arx cert revoke a1b2c3d4e5f6 --reason "key compromise"
```

Example output:

```text
Revoked a1b2c3d4e5f6 at 2026-06-01T15:04:05Z
```

| Flag | Description |
| ---- | ----------- |
| `--url`, `-u` | Override server URL (persisted when set) |
| `--reason` | Informational revocation reason |

---

## `arx util`

Administrative helper commands.

### `arx util hash <password>`

Generate a bcrypt hash for `bootstrap.admin_password_hash` in `server.yaml`.

```bash
arx util hash 'MySecureAdminPassword!'
```

---

## `arx hash`

Top-level alias for `arx util hash`.

```bash
arx hash 'MySecureAdminPassword!'
```

---

## `arx agent`

Local certificate inspection, trust installation, public catalog download, and optional enrollment. **Does not download private keys from the server** except via `agent enroll`, which calls `POST /api/v1/certificates/auto` with the admin JWT and stores key material under `~/.arx-cert-service/enrolled/`.

### `arx agent enroll`

```bash
arx agent enroll --domain example.local --ttl 720h
arx agent enroll --domain example.local -u http://localhost:8080
```

| Flag | Description |
| ---- | ----------- |
| `--domain` | DNS name for the issued certificate (required) |
| `--ttl` | Certificate lifetime (e.g. `24h`, `720h`) |
| `--url`, `-u` | Override server URL |

### `arx agent daemon`

Runs a blocking renewal loop for certificates listed in `~/.arx-cert-service/agent.yaml`. On each check interval the agent reads local PEM files, compares remaining TTL against `renew_threshold`, and when renewal is needed calls `POST /api/v1/certificates/auto` with the stored admin JWT. Renewed PEM files are written with mode `0600`; optional `post_hook` shell commands run after a successful renewal (for example `systemctl reload nginx`).

Requires a prior `arx login`. The daemon logs to stderr via `slog`.

```bash
arx agent daemon
arx agent daemon --config /etc/arx/agent.yaml
```

Example `agent.yaml`:

```yaml
daemon:
  check_interval: 24h
  renew_threshold: 720h
  managed_certs:
    - cert_path: /etc/nginx/ssl/app.pem
      key_path: /etc/nginx/ssl/app-key.pem
      template: web-server
      common_name: app.internal
      post_hook: systemctl reload nginx
```

| Field | Default | Description |
| ----- | ------- | ----------- |
| `daemon.check_interval` | `24h` | How often the renewal loop wakes up |
| `daemon.renew_threshold` | `720h` (30 days) | Renew when remaining certificate TTL is below this value |
| `managed_certs[].cert_path` | — | Path to the local certificate PEM file (required) |
| `managed_certs[].key_path` | — | Path to the local private key PEM file (required) |
| `managed_certs[].template` | — | Certificate template / profile name sent as `template_id` |
| `managed_certs[].common_name` | — | Common Name requested from the CA (required) |
| `managed_certs[].post_hook` | — | Shell command executed after successful renewal |

| Flag | Description |
| ---- | ----------- |
| `--config` | Path to `agent.yaml` (default: `~/.arx-cert-service/agent.yaml`) |

Environment overrides use prefix `ARX_AGENT_` (for example `ARX_AGENT_DAEMON_CHECK_INTERVAL=12h`).

### `arx agent local list`

```bash
arx agent local list
arx agent local list --store system --store user
```

| Flag | Description |
| ---- | ----------- |
| `--store` | Filter: `system`, `user`, `browser` (repeatable) |

### `arx agent local view <id>`

```bash
arx agent local view ABCD1234
```

Shows thumbprint, store, subject, issuer, serial, validity, and DNS names.

### `arx agent trust`

| Command | Description |
| ------- | ----------- |
| `install-root [--url]` | Fetch root CA PEM and install locally |
| `install-intermediate [--url]` | Install intermediate CA |
| `uninstall-root` | Remove installed root |
| `uninstall-intermediate` | Remove installed intermediate |

`--url` on install commands is persisted to agent config when set.

### `arx agent cert list`

Public read-only catalog (no admin JWT).

```bash
arx agent cert list --url http://localhost:8080
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
| `--url` | Base URL of the CA server |

---

## Command tree (reference)

```text
arx
├── server [--config]
│   ├── start
│   ├── config
│   │   └── init [--force]
│   ├── setup
│   └── service
│       ├── install [--run-as-user] [--install-dir]
│       └── uninstall [--run-as-user] [--install-dir]
├── login [--url] [--email] [--password]
├── ui [--url]
├── cert
│   ├── list [--url]
│   └── revoke <serial> [--url] [--reason]
├── util
│   └── hash <password>
├── hash <password>
└── agent
    ├── enroll --domain <name> [--ttl] [--url]
    ├── daemon [--config]
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

### Production install + admin

```bash
make build
sudo ./bin/arx server setup
# edit /opt/arx/server.yaml
arx login --url https://ca.example.com
arx ui
```

### Development server + admin

```bash
make build
./bin/arx server config init
./bin/arx server start
./bin/arx login --url http://localhost:8080
./bin/arx ui
```

### Trust the CA on a workstation

```bash
arx agent trust install-root --url http://localhost:8080
arx agent trust install-intermediate --url http://localhost:8080
```

### Enroll a leaf cert (admin JWT)

```bash
arx login --url http://localhost:8080
arx agent enroll --domain app.internal --ttl 2160h
```

### Automated renewal daemon

```bash
arx login --url http://localhost:8080
# edit ~/.arx-cert-service/agent.yaml — add managed_certs entries
arx agent daemon
```

---

## See also

- [architecture.md](architecture.md) — persistence, systemd, and package layout
- [acme.md](acme.md) — automated enrollment via ACME
- [../README.md](../README.md) — quick start and API table
