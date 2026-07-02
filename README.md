# ARX CA — Enterprise X.509 & SSH Certificate Authority

**ARX CA** is an enterprise-grade Certificate Authority platform: a **Go** control plane (PKI engine, REST API, enrollment protocols) and a **Vue 3** operator WebUI. Two static binaries ship with zero external database required for default deployments.

| Binary | Role |
| ------ | ---- |
| **`arx`** | Control plane — HTTP API, WebUI, admin CLI, ACME / SCEP / NDES |
| **`arx-agent`** | Data plane — certificate renewal, local trust stores, CRL self-quarantine |

---

## Documentation

> ### [📖 Project Wiki](https://github.com/ARCOOON/arx-ca/wiki)
>
> All technical documentation lives in the **GitHub Wiki**. The `wiki/` directory in this repository is a **Git submodule** linked to `ARCOOON/arx-ca.wiki`. Clone with `git clone --recurse-submodules` or run `git submodule update --init wiki` after checkout.

| Topic | Wiki page |
| ----- | --------- |
| Architecture & `server.yaml` | [Architecture](https://github.com/ARCOOON/arx-ca/wiki/Architecture) |
| SSH CA trust & principals | [SSH CA Setup](https://github.com/ARCOOON/arx-ca/wiki/SSH-CA-Setup) |
| CLI commands | [CLI Reference](https://github.com/ARCOOON/arx-ca/wiki/CLI-Reference) |
| REST API schemas | [API Reference](https://github.com/ARCOOON/arx-ca/wiki/API-Reference) |
| ACME enrollment | [ACME](https://github.com/ARCOOON/arx-ca/wiki/ACME) |
| Renewal agent | [Agent](https://github.com/ARCOOON/arx-ca/wiki/Agent) |

---

## Quick Start

**Install** (Linux / macOS):

```bash
curl -fsSL https://raw.githubusercontent.com/ARCOOON/arx-ca/main/scripts/install.sh | bash
```

**Initialize & start** the CA:

```bash
arx server config init
arx server start
```

**Authenticate** and open the admin console:

```bash
arx login --url https://ca.example.com
arx ui
```

Production systemd deployment: `sudo arx server setup` — see the [Wiki](https://github.com/ARCOOON/arx-ca/wiki) for full deployment guides.

---

## Trust CA (one-liner)

Distribute root trust to Linux hosts with a single remote script:

```bash
curl -sL https://ca.arx.local/trust.sh | sudo bash
```

Host-specific variants (Debian, Proxmox VE, RHEL) are documented in the [Wiki → SSH CA Setup](https://github.com/ARCOOON/arx-ca/wiki/SSH-CA-Setup).

---

## Highlights

- X.509 issuance, revocation, OCSP, CRL
- SSH Certificate Authority (user + host certs)
- ACMEv2, SCEP, NDES enrollment
- Immutable audit log with WebUI forensics dashboard
- Webhook + SSE notification engine
- Embedded SQLite — optional PostgreSQL for enterprise scale
- Operator WebUI aligned with the ARX ecosystem (`arx-dns`): Tailwind v4, Shadcn Vue, flat dark/light theming

---

## Build

```bash
make build-all    # bin/arx, bin/arx-agent, webui-dist.tar.gz
make test
```

Tagged releases are built and published automatically via [GoReleaser](https://goreleaser.com/) when a semantic version tag (`v*`) is pushed. The workflow ([`.github/workflows/release.yml`](.github/workflows/release.yml)) compiles the Vue WebUI, cross-compiles `arx` and `arx-agent` binaries, groups the changelog (**Features**, **Fixes**, **Dependency updates**), and uploads assets to GitHub Releases. Configuration lives in [`.goreleaser.yaml`](.goreleaser.yaml).

The `arx` server includes a background updater (`updater` block in `server.yaml`) that polls GitHub releases by channel and can notify administrators or auto-apply updates — see the [Architecture wiki](https://github.com/ARCOOON/arx-ca/wiki/Architecture#updater-block). Operators can manage updater settings from **Settings → Auto-Updater** in the WebUI or via `GET`/`PUT /api/v1/settings/config` (see [API Reference](https://github.com/ARCOOON/arx-ca/wiki/API-Reference#system-settings-serveryaml)). After an update, administrators can be prompted once with release notes fetched from GitHub (`view_changelog_after_update`, `GET /api/v1/updater/current-changelog`).

---

## License

See the repository license file.
