# arx-ca

A Go-based Certificate Authority API server built on the [step-ca SDK](https://github.com/smallstep/certificates). It exposes a RESTful HTTP interface for X.509 and SSH PKI operations while keeping the signing engine lean, auditable, and ready for a future Web UI.

## Design principle: Local-First, Cloud-Optional

**arx-ca is designed to run entirely on your infrastructure without any cloud dependency.**

| Layer             | Default (local)                                                                                     | Optional (plugins)                                                      |
| ----------------- | --------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| **Cryptography**  | SoftCAS — keys and certificates on disk under the PKI directory                                     | AWS KMS, GCP Cloud KMS, HashiCorp Vault (via step-ca CAS configuration) |
| **Persistence**   | Embedded BadgerDB (`badgerv2`) for single-node bootstrap; **PostgreSQL** recommended for production | MySQL and other step-ca–supported backends                              |
| **Identity**      | JWT admin login and in-process API key service accounts                                             | OIDC / Okta-style SSO (`CA_API_OIDC_*`)                                 |
| **Observability** | Structured request logging                                                                          | OpenTelemetry traces and metrics (OTLP exporters)                       |

All core PKI workflows — issue, renew, rekey, revoke, list, lint, ACME enrollment, and SSH signing — operate with **local cryptography and a local database**. Cloud integrations are **strictly optional**: enable them only when you deliberately configure external KMS, IdP, or telemetry backends.

## Features

### X.509 Certificate Authority

- **Automated PKI bootstrap** — generates Root CA, Intermediate CA, and default JWK provisioner on first start (ECDSA P-256).
- **Certificate issuance** — sign CSRs with configurable TTL (`POST /api/v1/certificates/issue`).
- **Provisioner-token issuance** — sign CSRs with a one-time JWK or OIDC token (`POST /api/v1/certificates/issue-with-token`).
- **Auto-enrollment** — generate a key pair and certificate in one request (`POST /api/v1/certificates/auto`).
- **Renewal and rekey** — renew or rotate certificates while preserving subject/SANs (`POST /api/v1/certificates/renew`, `POST /api/v1/certificates/rekey`).
- **Revocation** — passive revocation with RFC 5280 OCSP reason codes (`POST /api/v1/certificates/revoke`).
- **Certificate inventory** — list issued certificates with revocation state (`GET /api/v1/certificates`).
- **Compliance linting** — RFC 5280 and CA/Browser Forum checks via zlint (`POST /api/v1/certificates/lint`).
- **Root distribution** — export the Root CA PEM (`GET /api/v1/ca/root`).

### Revocation transparency (OCSP / CRL)

- Revocation records **OCSP reason codes** aligned with [RFC 6960](https://datatracker.ietf.org/doc/html/rfc6960).
- Revoked serials are persisted in the local certificate database for **OCSP responders and CRL generation** through the step-ca authority (configure OCSP/CRL endpoints in `ca.json` for your deployment).
- Certificate list responses include a `revoked` flag for operational visibility.

### ACME automation

- **ACME v2** provisioner enabled at bootstrap (HTTP-01, DNS-01, and TLS-ALPN-01).
- **External Account Binding (EAB)** — admins mint MAC keys via `POST /api/v1/acme/eab-keys` for gated account registration (`CA_API_ACME_REQUIRE_EAB=true`).
- **Device Attestation** — optional `device-attest-01` challenge for TPM, Apple, and YubiKey-style hardware keys (`CA_API_ACME_DEVICE_ATTEST=true`).
- ACME directory served under `/acme/` when enabled.
- ACME status and directory URL via the API (`GET /api/v1/acme/status`).
- Tunable via `CA_API_ACME_*` environment variables.

### SCEP and NDES (AD CS migration)

- **SCEP** provisioner from the step-ca SDK; protocol endpoint at `/scep/{provisioner}` (default `/scep/scep`).
- **NDES** connector registry routes Microsoft AD CS paths (`/certsrv/mscep/mscep.dll`) to the SCEP backend as a drop-in replacement.
- SCEP and NDES status via `GET /api/v1/scep/status` and `GET /api/v1/ndes/status`.
- RSA SCEP decrypter keys are generated automatically when the provisioner is first enabled.

### Provisioners and federation

- **JWK provisioners** — mint short-lived signing tokens (`POST /api/v1/provisioners/token`).
- **OIDC / SSO provisioners** — optional IdP integration (Okta, Azure AD, Google, etc.) through environment-driven `ca.json` updates.
- Provisioner inventory exposed through the PKI engine for automation and auditing.

### SSH Certificate Authority

- **Ed25519 SSH CA keys** generated at bootstrap (user and host CAs).
- **User certificates** — principals, TTL, OIDC or API-authenticated signing (`POST /api/v1/ssh/sign-user`).
- **Host certificates** — hostname principals for infrastructure (`POST /api/v1/ssh/sign-host`).
- **Inspection** — decode principals, validity, and metadata (`POST /api/v1/ssh/inspect`).
- **Trust anchors** — publish SSH CA public keys for `known_hosts` / `TrustedUserCAKeys` (`GET /api/v1/ssh/roots`).

### Security and RBAC

- **Role-based access control** at the HTTP middleware layer:
  - **Admin** — JWT Bearer tokens from `POST /api/v1/auth/login` (bootstrap admin account).
  - **Service account** — API keys (`X-API-Key` or Bearer) for automation; admins may create keys via `POST /api/v1/auth/service-accounts`.
- Protected routes require **admin JWT** or **service-account API key**; SSH user signing accepts either model plus provisioner tokens.
- Stateless API suitable for reverse proxies and future Web UI integration.

### Operations and observability

- **Health endpoint** — process uptime, memory, API version, and CA backend status (`GET /api/v1/health`).
- **OpenTelemetry** — distributed tracing and metrics through the step-ca dependency graph (`otelhttp`, OTLP SDK); export to your collector without coupling PKI logic to a vendor.
- **Request logging** — structured HTTP access logs via middleware.
- **Graceful shutdown** — SIGINT/SIGTERM handling with bounded drain timeout.
- **Container images** — multi-stage Docker build and Compose stack for local or air-gapped deployment.

### API conventions

- JSON envelope: `{ "data": …, "error": null | { "message": "…" } }`.
- Go 1.22+ `net/http` routing (method-aware patterns, no heavy web framework).
- RESTful, versioned paths under `/api/v1/`.

## Architecture

```
                    ┌─────────────────────────────────────────┐
                    │           Clients / Automation          │
                    │  (CLI, CI, agents, future Web UI)       │
                    └────────────────────┬────────────────────┘
                                         │ HTTPS
                    ┌────────────────────▼────────────────────┐
                    │  API Layer (internal/api)               │
                    │  • Handlers (REST)                      │
                    │  • RBAC middleware (JWT / API keys)     │
                    │  • Logger / OTel-instrumented HTTP      │
                    └────────────────────┬────────────────────┘
                                         │
                    ┌────────────────────▼────────────────────┐
                    │  PKI Engine (internal/ca)               │
                    │  • PKIEngine / InitCA                   │
                    │  • X.509, ACME, SSH, provisioners       │
                    │  • Revocation (OCSP codes → CRL/OCSP)   │
                    └────────────────────┬────────────────────┘
                                         │
         ┌───────────────────────────────┼───────────────────────────────┐
         │                               │                               │
         ▼                               ▼                               ▼
┌─────────────────┐           ┌─────────────────┐           ┌─────────────────┐
│  SoftCAS        │           │  PostgreSQL /   │           │  OpenTelemetry  │
│  (local keys)   │           │  Badger (local) │           │  (optional OTLP)│
└─────────────────┘           └─────────────────┘           └─────────────────┘
         │                               │
         │ optional plugins              │
         ▼                               ▼
┌─────────────────┐           ┌─────────────────┐
│ AWS / GCP KMS   │           │ Okta / OIDC IdP │
│ HashiCorp Vault │           │ (provisioners)  │
└─────────────────┘           └─────────────────┘
```

### Layer responsibilities

| Layer          | Package                   | Responsibility                                                                    |
| -------------- | ------------------------- | --------------------------------------------------------------------------------- |
| **Handlers**   | `internal/api/handlers`   | HTTP translation, input validation, standardized JSON responses                   |
| **Middleware** | `internal/api/middleware` | RBAC (`RequireAdmin`, `RequireServiceAccountOrAdmin`), logging, future OTel spans |
| **Auth**       | `internal/auth`           | JWT issuance, API key store, bootstrap admin credentials                          |
| **PKI engine** | `internal/ca`             | Business logic: sign, revoke, renew, ACME, SSH, lint; wraps `authority.Authority` |
| **Models**     | `internal/models`         | Request/response DTOs shared by handlers and engine                               |

The PKI engine delegates cryptographic operations to **step-ca** (`github.com/smallstep/certificates`). That keeps OCSP/CRL semantics, provisioner types, and CAS plugins aligned with upstream while arx-ca owns the HTTP contract and deployment ergonomics.

### RBAC model

| Role                  | Credential                    | Typical use                                                          |
| --------------------- | ----------------------------- | -------------------------------------------------------------------- |
| **Admin**             | JWT from `/api/v1/auth/login` | Create service accounts, break-glass operations                      |
| **Service account**   | `X-API-Key` header            | CI/CD issuance, renewals, SSH host signing                           |
| **Provisioner token** | JWK or OIDC token in body     | End-entity certificate or SSH user flows without long-lived API keys |

Middleware enforces roles **before** handlers invoke the PKI engine, so signing paths never run without an authenticated principal.

### OCSP and CRL

Revocation flows through the step-ca authority: serial numbers and OCSP reason codes are written to the configured database. For production, point `ca.json` at **PostgreSQL** and enable step-ca **OCSP** and **CRL** configuration so relying parties can validate certificate status independently of the REST API.

### OpenTelemetry

OpenTelemetry is enabled on the HTTP router (`internal/telemetry`). Traces and metrics export over OTLP/HTTP by default to `http://localhost:4318` (Jaeger, Grafana Alloy, or similar). Set `OTEL_SDK_DISABLED=true` to turn telemetry off.

## API overview

| Method | Path                                    | Auth                | Description                     |
| ------ | --------------------------------------- | ------------------- | ------------------------------- |
| `GET`  | `/api/v1/health`                        | —                   | Health and runtime metrics      |
| `GET`  | `/api/v1/ca/root`                       | —                   | Root CA PEM                     |
| `GET`  | `/api/v1/ca/crl`                        | —                   | CRL (DER; add `?pem` for PEM)   |
| `POST` | `/ocsp`                                 | —                   | OCSP responder (DER request)    |
| `GET`  | `/ocsp/{base64}`                        | —                   | OCSP responder (URL-safe request) |
| `POST` | `/api/v1/auth/login`                    | —                   | Admin JWT                       |
| `POST` | `/api/v1/auth/service-accounts`         | Admin               | Create API key                  |
| `POST` | `/api/v1/certificates/issue`            | Service / Admin     | Sign CSR                        |
| `POST` | `/api/v1/certificates/issue-with-token` | Service / Admin     | Sign CSR with provisioner token |
| `POST` | `/api/v1/certificates/auto`             | Service / Admin     | Generate key + certificate      |
| `POST` | `/api/v1/certificates/revoke`           | Service / Admin     | Revoke by serial                |
| `POST` | `/api/v1/certificates/lint`             | Service / Admin     | Lint PEM certificate            |
| `GET`  | `/api/v1/certificates`                  | Service / Admin     | List certificates               |
| `POST` | `/api/v1/certificates/renew`            | Service / Admin     | Renew certificate               |
| `POST` | `/api/v1/certificates/rekey`            | Service / Admin     | Rekey certificate               |
| `GET`  | `/api/v1/acme/status`                   | Service / Admin     | ACME directory metadata         |
| `POST` | `/api/v1/acme/eab-keys`                 | Admin               | Create ACME EAB MAC key         |
| `GET`  | `/api/v1/scep/status`                   | Service / Admin     | SCEP endpoint metadata          |
| `GET`  | `/api/v1/ndes/status`                   | Service / Admin     | NDES connector metadata         |
| `POST` | `/api/v1/provisioners/token`            | Service / Admin     | Mint JWK signing token          |
| `POST` | `/api/v1/ssh/sign-user`                 | Token / API / Admin | SSH user certificate            |
| `POST` | `/api/v1/ssh/sign-host`                 | Service / Admin     | SSH host certificate            |
| `POST` | `/api/v1/ssh/inspect`                   | Service / Admin     | Decode SSH certificate          |
| `GET`  | `/api/v1/ssh/roots`                     | —                   | SSH CA public keys              |
| `*`    | `/acme/...`                             | ACME protocol       | ACME directory (when enabled)   |
| `*`    | `/scep/...`                             | SCEP protocol       | SCEP enrollment (when enabled)  |
| `*`    | `/certsrv/...`                          | NDES / AD CS paths  | NDES → SCEP connector           |

## Setup

### Prerequisites

- Go 1.22 or newer (module may require a newer toolchain; `GOTOOLCHAIN=auto` is used in Docker builds)
- Optional: Docker and Docker Compose for containerized runs
- Optional: PostgreSQL instance for production-grade persistence

### Local development

```bash
# Build server, Super Admin CLI, and local cert service
make build

# Run tests
make test

# Start the API (creates .pki/ on first run if missing)
export CA_API_JWT_SECRET="$(openssl rand -base64 32)"
./bin/arx-ca-server
```

Cross-compile all three binaries for Linux and Windows:

```bash
make build-server-linux build-server-windows \
     build-cli-linux build-cli-windows \
     build-agent-linux build-agent-windows
```

Windows artifacts are written to `bin/*.exe`. Linux artifacts omit the extension.

| Binary            | Role                                                                 |
| ----------------- | -------------------------------------------------------------------- |
| `arx-ca-server`   | CA API and enrollment protocols                                      |
| `arx-ca-cli`      | Super Admin terminal UI and CA management (`login`, `ui`)            |
| `arx-cert-service`| Local trust stores and read-only public certificate download         |

Default listen address: `:8080`. Default PKI path: `.pki/config/ca.json`.

### Docker

```bash
# Build image and start with Compose
export CA_API_JWT_SECRET="$(openssl rand -base64 32)"
make docker-build
make docker-up
```

PKI material is persisted under `./data/arx-ca` on the host (mounted to `/app/data` in the container).

### Configuration reference

| Variable             | Default               | Description                                                                                          |
| -------------------- | --------------------- | ---------------------------------------------------------------------------------------------------- |
| `CA_API_LISTEN_ADDR` | `:8080`               | HTTP listen address                                                                                  |
| `CA_API_CA_CONFIG`   | `.pki/config/ca.json` | Path to step-ca configuration                                                                        |
| `CA_API_JWT_SECRET`  | *(ephemeral)*         | HMAC secret for admin JWTs; **set in production**                                                    |
| `CA_API_JWT_ISSUER`  | *(empty)*             | JWT issuer claim                                                                                     |
| `CA_API_JWT_EXPIRY`  | `24h`                 | Admin token lifetime                                                                                 |
| `CA_API_CA_PASSWORD` | *(file or generated)* | Intermediate key encryption password                                                                 |
| `CA_API_DB_TYPE`     | `badgerv2`            | Local DB driver for bootstrap (`badgerv2`, `bbolt`, …); use `postgresql` in `ca.json` for production |
| `CA_API_OIDC_*`      | —                     | Optional OIDC provisioner (client ID, discovery URL, tenant, domains)                                |
| `CA_API_ACME_*`      | —                     | ACME HTTP/TLS ports, DNS name, strict FQDN, disable flag                                             |
| `CA_API_ACME_REQUIRE_EAB` | `false`          | Require EAB for new ACME accounts                                                                    |
| `CA_API_ACME_DEVICE_ATTEST` | `false`        | Enable `device-attest-01` challenge                                                                  |
| `CA_API_ACME_ATTESTATION_FORMATS` | `apple,step,tpm` | Device attestation formats enabled                                                       |
| `CA_API_ACME_ATTESTATION_ROOTS` | —              | PEM bundle path for custom attestation roots                                                         |
| `CA_API_SCEP_DISABLED` | `false`             | Skip SCEP provisioner bootstrap                                                                      |
| `CA_API_SCEP_CHALLENGE` | *(generated)*      | SCEP challenge password (pin for production)                                                         |
| `CA_API_NDES_DISABLED` | `false`            | Disable NDES AD CS path routing                                                                      |
| `CA_API_NDES_ADMIN_SECRET` | —               | Secret for `mscep_admin` password endpoint                                                           |
| `CA_API_CRL_DISABLED`  | `false`               | Skip automatic CRL block in `ca.json`                                                                |
| `OTEL_SERVICE_NAME`    | `arx-ca`              | OpenTelemetry service name                                                                           |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` | OTLP/HTTP collector endpoint (traces and metrics)                                          |
| `OTEL_EXPORTER_OTLP_INSECURE` | `true` for `http://` endpoints | Disable TLS for OTLP export                                                           |
| `OTEL_SDK_DISABLED`    | `false`               | Disable OpenTelemetry exporters                                                                      |
| `ARX_CA_OTEL_ENDPOINT` | —                     | Fallback OTLP endpoint when `OTEL_EXPORTER_OTLP_ENDPOINT` is unset                                   |

For **PostgreSQL**, set the `db` block in `ca.json` per [step-ca database documentation](https://smallstep.com/docs/step-ca/configuration#databases) after initial bootstrap, then restart the server.

### Bootstrap admin

On first login, use username `admin` with the bootstrap password defined in `internal/auth/admin.go` (change this hash before any production deployment).
