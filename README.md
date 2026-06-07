# arx — Enterprise X.509 & SSH Certificate Authority

**arx** is an enterprise-grade Certificate Authority platform built with **Go** (control plane, PKI engine, REST API) and **Vue 3** (operator WebUI). It ships two static binaries — `arx` (CA server) and `arx-agent` (renewal daemon) — with zero external database required for default deployments.

| Binary | Role |
| ------ | ---- |
| **`arx`** | Control plane — HTTP API, WebUI, admin CLI, ACME / SCEP / NDES |
| **`arx-agent`** | Data plane — certificate renewal, local trust stores, CRL self-quarantine |

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

Production systemd deployment: `sudo arx server setup` — see the [Project Wiki](https://github.com/ARCOOON/arx-ca/wiki) for full deployment guides.

---

## Trust CA (one-liner)

Distribute root trust to Linux hosts with a single remote script:

```bash
curl -sL https://ca.example.com/trust.sh | sudo bash
```

The script installs the arx root CA into the system trust store and configures renewal-agent hooks. Host-specific variants (Debian, Proxmox VE, RHEL) are documented in the [Wiki → SSH CA Setup](https://github.com/ARCOOON/arx-ca/wiki/SSH-CA-Setup).

---

## Documentation

> **All technical documentation lives in the [Project Wiki](https://github.com/ARCOOON/arx-ca/wiki).**
>
> The `wiki/` directory in this repository is a **Git submodule** linked to the official GitHub Wiki (`ARCOOON/arx-ca.wiki`). Clone with `git clone --recurse-submodules` or run `git submodule update --init wiki` after checkout.

| Topic | Wiki page |
| ----- | --------- |
| **Architecture** — dual-listener model, DDD layers, `server.yaml` reference | [Architecture](https://github.com/ARCOOON/arx-ca/wiki/Architecture) |
| **Deployment** — systemd, Docker, PostgreSQL, production hardening | [Architecture → Configuration](https://github.com/ARCOOON/arx-ca/wiki/Architecture#configuration-reference-serveryaml) |
| **SSH CA** — `sshd` trust, principals, Proxmox & Debian | [SSH CA Setup](https://github.com/ARCOOON/arx-ca/wiki/SSH-CA-Setup) |
| **REST API** — full endpoint matrix | [API Reference](https://github.com/ARCOOON/arx-ca/wiki/API-Reference) |
| **Audit & forensics** — immutable log, filters | [Audit Log](https://github.com/ARCOOON/arx-ca/wiki/Audit-Log) |
| **Webhooks & notifications** — SSE, Discord, Slack | [Webhooks & Notifications](https://github.com/ARCOOON/arx-ca/wiki/Webhooks-&-Notifications) |

Developer-oriented references remain in [`docs/`](docs/) (CLI flags, API schemas, agent spec).

---

## Highlights

- X.509 issuance, revocation, OCSP, CRL
- SSH Certificate Authority (user + host certs)
- ACMEv2, SCEP, NDES enrollment
- Immutable audit log with WebUI forensics dashboard
- Webhook + SSE notification engine
- Embedded SQLite — optional PostgreSQL for enterprise scale

---

## Build

```bash
make build-all    # bin/arx, bin/arx-agent, webui-dist.tar.gz
make test
```

Tagged releases build via [`.github/workflows/release.yml`](.github/workflows/release.yml).

---

## License

See the repository license file.
