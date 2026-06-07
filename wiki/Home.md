# arx CA — Operator Wiki

**arx** is a Certificate Authority platform built on the [step-ca SDK](https://github.com/smallstep/certificates).

## What ships

| Binary | Role |
| ------ | ---- |
| `arx` | Control plane — HTTP API, WebUI, admin CLI |
| `arx-agent` | Data plane — renewal daemon, local trust stores |

## Core capabilities

- X.509 issuance, revocation, OCSP, CRL
- ACMEv2 (`/acme/directory`), SCEP, NDES
- SSH Certificate Authority (user + host certs)
- Immutable audit log with forensic query API
- Webhook + SSE notification engine
- Vue 3 WebUI with App Shell layout

## Default deployment

- **API:** `http://localhost:8080` (`server.port`)
- **WebUI:** `https://localhost:8443` when `webui.enabled: true`
- **App DB:** SQLite `arx.db` beside `server.yaml`
- **PKI:** step-ca Badger store under `.pki/`

## Wiki map

| Page | Contents |
| ---- | -------- |
| [Architecture](Architecture) | Component diagram, Vue/Go split, SQLite schema |
| [SSH CA Setup](SSH-CA-Setup) | Linux `sshd` trust + issuance sequence |
| [Webhooks & Notifications](Webhooks-&-Notifications) | Events, payloads, dispatcher flow |
| [API Reference](API-Reference) | All `/api/v1` endpoints |
| [Audit Log](Audit-Log) | Forensics, skipped methods, filters |

## Quick start

```bash
arx server config init
arx server start
```

Open the WebUI listener (default `:8443`) and log in with the bootstrap admin credentials.
