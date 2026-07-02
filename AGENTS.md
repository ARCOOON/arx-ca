# AGENTS.md

## Cursor Cloud specific instructions

ARX CA is a Go control-plane (`arx` server + `arx-agent`) plus a Vue 3 WebUI in `webui/`.
Standard build/test/run commands live in `Makefile`, `README.md`, and `webui/README.md`;
prefer those. Notes below are the non-obvious caveats discovered while running this in the
cloud environment.

### Services

| Service | Dir | Dev command | Port |
| ------- | --- | ----------- | ---- |
| `arx` API / control plane | repo root | `./bin/arx server start --config <path>/server.yaml` | 8080 |
| WebUI (Vite dev server) | `webui/` | `npm run dev` | 5173 (proxies `/api` -> 127.0.0.1:8080) |

- Build binaries with `make build` (outputs `bin/arx`, `bin/arx-agent`). Go 1.25 + Node 22 are available.
- Run Go tests with `make test` (`go test ./...`). There are currently no `_test.go` files, so this only compiles.
- The WebUI `npm run build` runs `vue-tsc` type-checking first — it is the effective lint/typecheck gate for the frontend.

### Running the server (non-obvious)

- `arx server start` **chdir's into the directory containing `server.yaml`** and creates all
  state relative to it: `arx.db` (SQLite), `.pki/` (root/intermediate/SSH CA keys), etc.
  Run it against a dedicated data dir (this repo uses `/.dev-data/`, which is git-ignored) so
  PKI material and the DB never land in the repo root.
- First `server config init` then `server start`. On first boot the server auto-generates the
  JWT secret, the full PKI (step-ca based), and seeds the bootstrap admin — no external DB
  needed (SQLite is the default; PostgreSQL is optional).
- The bootstrap admin (`admin@arx.local`) is only seeded when the `users` table is empty, and
  the seeded password comes from `bootstrap.admin_password` in `server.yaml`, which **must be a
  bcrypt hash** (generate one with `arx hash '<password>'`). The default generated `server.yaml`
  ships an unknown-plaintext hash, so to get a known login either set your own bcrypt hash in
  `bootstrap.admin_password` before first boot, or delete `arx.db` and restart after changing it.
  The `CA_API_BOOTSTRAP_ADMIN_PASSWORD` env var is ignored when the config value is already a
  bcrypt hash.
- CORS in the generated config already allows `http://localhost:5173`, so the Vite dev server
  proxy works out of the box. `webui.enabled: false` in `server.yaml` is fine for dev — the Vue
  app is served by Vite, not the Go binary.
- The background updater is enabled by default and will log "new release available" against the
  dev version; harmless in dev (`notify_only: true`).
