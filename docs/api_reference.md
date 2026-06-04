# Arx CA HTTP API Reference

Interactive reference for every HTTP endpoint exposed by **arx-ca-server**. Routes are registered in `internal/cmd/arx/server_start.go`. JSON management APIs under `/api/v1` use a standard envelope; ACME, OCSP, CRL, SCEP, and NDES use protocol-specific formats.

## Base URL and Versioning


| Item | Value |
| ---- | ----- |
| **API prefix** | `/api/v1` |
| **Default listen** | `http://localhost:8080` (configurable via server config) |
| **Content-Type (JSON APIs)** | `application/json; charset=utf-8` |

Example base: `https://ca.example.com/api/v1`

## Standard JSON Envelope


All `/api/v1/*` handlers return a standard envelope. Examples in endpoint sections show the **`data`** payload only; wrap successful responses in the envelope when calling the API.

<details>
  <summary><strong>View Success Envelope</strong></summary>

```json
{
  "error": null,
  "data": { }
}
```

</details>

<details>
  <summary><strong>View Error Envelope</strong></summary>

```json
{
  "error": "invalid email or password",
  "data": null
}
```

</details>

## Authentication Overview


### Admin JWT

1. `POST /api/v1/auth/login` with email and password.
2. Send the returned token on protected routes:

```http
Authorization: Bearer <jwt>
```

| Field | Description |
| ----- | ----------- |
| `token_type` | Always `Bearer` |
| `roles` | `SuperAdmin`, `CA-Admin`, `Revocation-Manager`, `Read-Only` |

### Service account (Agent) API key

```http
X-API-Key: <api_key>
```

or

```http
Authorization: Bearer <api_key>
```

API keys are created via `POST /api/v1/auth/service-accounts` (admin only).

### Permission labels in this document

| Label | Meaning |
| ----- | ------- |
| `None` | No credentials required |
| `Admin` | Valid admin JWT required |
| `Agent` | Valid service-account API key required |
| `Admin \| Agent` | Either JWT or API key; RBAC capability noted when enforced |
| `Admin \| mTLS` | Valid admin JWT **or** verified mutual TLS client certificate (see below) |

Missing or invalid credentials → `401`. Valid credentials without RBAC capability → `403`.

### Mutual TLS (mTLS) client authentication

Selected endpoints accept **mutual TLS** as an alternative to admin JWT. The client must present a **valid, non-revoked certificate** issued by this CA.

| Requirement | Detail |
| ----------- | ------ |
| Direct API TLS | HTTPS with a client certificate when `server.tls.enabled: true` |
| WebUI proxy | HTTPS to `webui.listen_address` with `--cert` / `--key`; the WebUI forwards the leaf cert via `X-Forwarded-Client-Cert` to the API on loopback |
| Certificate | Issued by this CA, not expired, not revoked |
| Identity binding | On `/renew` and `/rekey`, the client certificate **Common Name (CN)** must match the CN of the certificate being renewed |

<details>
  <summary><strong>View mTLS Renewal Example (curl)</strong></summary>

```bash
curl --cert client.crt --key client.key --cacert root_ca.crt \
  -X POST https://ca.example.com:8443/api/v1/certificates/renew \
  -H "Content-Type: application/json" \
  -d '{"certificate_pem": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----\n"}'
```

The client certificate CN must match the subject CN of `certificate_pem`. No `Authorization` header is required when mTLS identity is sufficient.

</details>

### RBAC capability matrix

| Capability | Roles that grant it |
| ---------- | ------------------- |
| `certificates:issue` | SuperAdmin, CA-Admin |
| `certificates:revoke` | SuperAdmin, Revocation-Manager |
| `certificates:read` | SuperAdmin, CA-Admin, Revocation-Manager, Read-Only |
| `certificates:lint` | SuperAdmin, CA-Admin, Revocation-Manager |
| `certificates:renew` | SuperAdmin, CA-Admin |
| `provisioners:token` | SuperAdmin, CA-Admin |
| `templates:manage` | SuperAdmin, CA-Admin |
| `templates:read` | SuperAdmin, CA-Admin, Read-Only |
| `service_accounts:manage` | SuperAdmin |
| `acme:eab` | SuperAdmin, CA-Admin |
| `enrollment:status` | SuperAdmin, CA-Admin, Revocation-Manager, Read-Only |
| `ssh:sign_host` | SuperAdmin, CA-Admin |
| `ssh:inspect` | SuperAdmin, CA-Admin |

## Table of Contents


1. [Health & Status](#health--status)
2. [Authentication](#authentication-endpoints)
3. [CA Certificates & Revocation Lists](#ca-certificates--revocation-lists)
4. [Public / Agent Read-Only API](#public--agent-read-only-api)
5. [Certificate Management](#certificate-management)
6. [Certificate Renewal & Rekey](#certificate-renewal--rekey)
7. [Provisioners & Enrollment Status](#provisioners--enrollment-status)
8. [Certificate Templates](#certificate-templates)
9. [SSH Certificate Authority](#ssh-certificate-authority)
10. [OCSP Responder](#ocsp-responder)
11. [ACMEv2 (RFC 8555)](#acmev2-rfc-8555)
12. [SCEP (optional)](#scep-optional)
13. [NDES (optional)](#ndes-optional)

---

## Health & Status


### GET /api/v1/health
> Returns process uptime, memory statistics, API version, and PKI engine status.

- **Authentication:** Not required
- **Permissions:** `None`

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `uptime` | object | Yes | Server uptime (see below) |
| `memory` | object | Yes | Go runtime memory stats (see below) |
| `api` | object | Yes | API layer status (see below) |
| `ca_backend` | object | Yes | PKI engine status (see below) |

**`uptime` Object:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `seconds` | int64 | Yes | Uptime in seconds |
| `human` | string | Yes | Human-readable uptime |

**`memory` Object:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `alloc_bytes` | uint64 | Yes | Bytes allocated and in use |
| `total_alloc_bytes` | uint64 | Yes | Cumulative bytes allocated |
| `sys_bytes` | uint64 | Yes | Bytes obtained from OS |
| `heap_alloc_bytes` | uint64 | Yes | Heap bytes allocated |
| `heap_inuse_bytes` | uint64 | Yes | Heap bytes in use |
| `heap_objects` | uint64 | Yes | Number of heap objects |
| `stack_inuse_bytes` | uint64 | Yes | Stack bytes in use |
| `num_gc` | uint32 | Yes | Completed GC cycles |
| `last_gc_unix` | int64 | Yes | Last GC time (Unix seconds); `0` if none |
| `goroutines` | int | Yes | Active goroutine count |

**`api` Object:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | API health (`healthy`) |
| `version` | string | Yes | API version (`v1`) |

**`ca_backend` Object:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | Engine status |
| `message` | string | No | Additional status detail |
| `engine` | string | Yes | Engine identifier (`step-ca`) |
| `initialized` | bool | Yes | Whether PKI engine initialized |

**Example JSON (`data`):**
```json
{
  "uptime": {"seconds": 3600, "human": "1h 0m 0s"},
  "memory": {
    "alloc_bytes": 10485760,
    "total_alloc_bytes": 52428800,
    "sys_bytes": 33554432,
    "heap_alloc_bytes": 8388608,
    "heap_inuse_bytes": 12582912,
    "heap_objects": 42150,
    "stack_inuse_bytes": 524288,
    "num_gc": 12,
    "last_gc_unix": 1717347845,
    "goroutines": 18
  },
  "api": {"status": "healthy", "version": "v1"},
  "ca_backend": {
    "status": "healthy",
    "message": "",
    "engine": "step-ca",
    "initialized": true
  }
}
```

</details>

**Error Codes:**
* `405 Method Not Allowed` — non-GET request.
* `500 Internal Server Error` — encoding failure.

---
## Authentication Endpoints


### POST /api/v1/auth/login
> Authenticates an admin user and returns a short-lived HS256 JWT with assigned roles.

- **Authentication:** Not required
- **Permissions:** `None`

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `email` | string | Yes | Admin email address |
| `password` | string | Yes | Admin password |

**Example JSON:**
```json
{"email": "admin@example.com", "password": "secretpassword"}
```

</details>

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | string | Yes | JWT access token |
| `expires_at` | string (RFC3339) | Yes | Token expiration |
| `token_type` | string | Yes | Token scheme (`Bearer`) |
| `roles` | string[] | No | Assigned RBAC roles |

**Example JSON (`data`):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2026-06-03T12:00:00Z",
  "token_type": "Bearer",
  "roles": ["SuperAdmin"]
}
```

</details>

**Error Codes:**
* `400 Bad Request` — invalid JSON payload.
* `401 Unauthorized` — invalid email or password.
* `405 Method Not Allowed` — non-POST request.
* `500 Internal Server Error` — login failure.

---
### POST /api/v1/auth/service-accounts
> Creates a service account and returns a one-time API key for automation clients.

- **Authentication:** Required (Bearer JWT)
- **Permissions:** `Admin` — requires RBAC `service_accounts:manage`

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique service account name |
| `roles` | string[] | No | RBAC roles; unknown roles are ignored |

**Example JSON:**
```json
{"name": "ci-deploy-bot", "roles": ["CA-Admin"]}
```

</details>

#### Response
<details>
  <summary><strong>View Response (201 Created)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Service account ID |
| `name` | string | Yes | Service account name |
| `roles` | string[] | Yes | Assigned roles |
| `api_key` | string | Yes | Plaintext API key (shown once) |
| `created_at` | string (RFC3339) | Yes | Creation timestamp |

**Example JSON (`data`):**
```json
{
  "id": "sa_01HXYZABCDEF",
  "name": "ci-deploy-bot",
  "roles": ["CA-Admin"],
  "api_key": "arx_sk_live_abcdefghijklmnopqrstuvwxyz",
  "created_at": "2026-06-02T10:00:00Z"
}
```

</details>

**Error Codes:**
* `400 Bad Request` — invalid JSON or service account name.
* `401 Unauthorized` — missing or invalid JWT.
* `403 Forbidden` — insufficient permissions.
* `409 Conflict` — duplicate name.
* `500 Internal Server Error` — creation failure.

---
## CA Certificates & Revocation Lists


### GET /api/v1/ca/root
> Returns the Root CA certificate PEM for trust store installation.

- **Authentication:** Not required
- **Permissions:** `None`

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pem` | string | Yes | Root CA certificate PEM |

**Example JSON (`data`):**
```json
{"pem": "-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----\n"}
```

</details>

**Error Codes:**
* `405 Method Not Allowed` — non-GET request.
* `500 Internal Server Error` — root certificate unavailable.

---
### GET /api/v1/ca/info
> Returns parsed X.509 metadata and PEM strings for the active Root and Intermediate CA certificates.

- **Authentication:** Required (JWT or service account API key)
- **Permissions:** `certificates:read`

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `root` | object | Yes | Root CA certificate metadata |
| `intermediate` | object | Yes | Intermediate CA certificate metadata |

**`root` / `intermediate` object:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `subject` | object | Yes | Parsed certificate subject DN |
| `issuer` | object | Yes | Parsed certificate issuer DN |
| `not_before` | string | Yes | Validity start (RFC3339 UTC) |
| `not_after` | string | Yes | Validity end (RFC3339 UTC) |
| `fingerprint` | string | Yes | SHA-256 fingerprint (lowercase hex) |
| `pem` | string | Yes | PEM-encoded certificate |

**`subject` / `issuer` object:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `common_name` | string | Yes | Common Name (CN) |
| `organization` | string[] | No | Organization (O) |
| `organizational_unit` | string[] | No | Organizational Unit (OU) |
| `country` | string[] | No | Country (C) |
| `province` | string[] | No | State or province (ST) |
| `locality` | string[] | No | Locality (L) |
| `street_address` | string[] | No | Street address |
| `postal_code` | string[] | No | Postal code |
| `serial_number` | string | No | Subject serial number |

**Example JSON (`data`):**
```json
{
  "root": {
    "subject": {
      "common_name": "ARX Root CA",
      "organization": ["ARX Infrastructure"],
      "country": ["AT"]
    },
    "issuer": {
      "common_name": "ARX Root CA",
      "organization": ["ARX Infrastructure"],
      "country": ["AT"]
    },
    "not_before": "2026-06-04T12:00:00Z",
    "not_after": "2036-06-02T12:00:00Z",
    "fingerprint": "a1b2c3d4e5f6…",
    "pem": "-----BEGIN CERTIFICATE-----\n…\n-----END CERTIFICATE-----\n"
  },
  "intermediate": {
    "subject": {
      "common_name": "ARX Intermediate CA",
      "organization": ["ARX Infrastructure"],
      "country": ["AT"]
    },
    "issuer": {
      "common_name": "ARX Root CA",
      "organization": ["ARX Infrastructure"],
      "country": ["AT"]
    },
    "not_before": "2026-06-04T12:00:00Z",
    "not_after": "2036-06-02T12:00:00Z",
    "fingerprint": "f6e5d4c3b2a1…",
    "pem": "-----BEGIN CERTIFICATE-----\n…\n-----END CERTIFICATE-----\n"
  }
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — caller lacks `certificates:read`.
* `405 Method Not Allowed` — non-GET request.
* `500 Internal Server Error` — CA certificates unavailable.

---
### GET /api/v1/ca/crl
> Returns the current Certificate Revocation List in DER or PEM format. Not wrapped in the JSON envelope.

- **Authentication:** Not required
- **Permissions:** `None`
- **Alias:** `GET /api/v1/crl` (identical handler and response)
- **Query Parameters:** `pem` (flag) — if present, response is PEM; otherwise DER (`application/pkix-crl`)

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Response body:** raw CRL bytes (DER or PEM).

**Response headers:**

| Header | Description |
|--------|-------------|
| `Content-Type` | `application/pkix-crl` or `application/x-pem-file` |
| `Content-Disposition` | `attachment; filename="crl.crl"` or `crl.pem` |
| `Expires` | CRL next-update (RFC1123) |

*(Binary body — no JSON example.)*

</details>

**Error Codes:**
* `405 Method Not Allowed` — non-GET request.
* `500 Internal Server Error` — CRL unavailable (plain text via `http.Error`).

---
## Public / Agent Read-Only API

> Unauthenticated endpoints used by the **arx agent** for trust installation and certificate discovery. They never expose private keys or signing operations.
### GET /api/v1/public/ca/intermediate
> Returns the Intermediate CA certificate PEM.

- **Authentication:** Not required
- **Permissions:** `None`

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pem` | string | Yes | Intermediate CA certificate PEM |

**Example JSON (`data`):**
```json
{"pem": "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----\n"}
```

</details>

**Error Codes:**
* `405 Method Not Allowed` — non-GET request.
* `500 Internal Server Error` — intermediate unavailable.

---
### GET /api/v1/public/certificates
> Lists issued certificates with public metadata (no private key material).

- **Authentication:** Not required
- **Permissions:** `None`

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificates` | array | Yes | Certificate summaries (see below) |
| `total` | int | Yes | Number of entries |

**`certificates[]` Element:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `serial` | string | Yes | Certificate serial |
| `subject` | string | Yes | Distinguished name |
| `dns_names` | string[] | No | DNS SANs |
| `ip_addresses` | string[] | No | IP SANs |
| `not_before` | string | Yes | Validity start |
| `not_after` | string | Yes | Validity end |
| `revoked` | bool | Yes | Revocation flag |

**Example JSON (`data`):**
```json
{
  "certificates": [
    {
      "serial": "12345678901234567890",
      "subject": "CN=www.example.com",
      "dns_names": ["www.example.com", "example.com"],
      "ip_addresses": ["203.0.113.10"],
      "not_before": "2026-06-01T00:00:00Z",
      "not_after": "2026-09-01T00:00:00Z",
      "revoked": false
    }
  ],
  "total": 1
}
```

</details>

**Error Codes:**
* `405 Method Not Allowed` — non-GET request.
* `500 Internal Server Error` — list failure.

---
### GET /api/v1/public/certificates/{serial}
> Downloads a single certificate PEM by decimal serial number.

- **Authentication:** Not required
- **Permissions:** `None`
- **Query Parameters:** `serial` (path) — certificate serial number

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificate_pem` | string | Yes | Certificate PEM |
| `serial` | string | Yes | Serial number |

**Example JSON (`data`):**
```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "serial": "12345678901234567890"
}
```

</details>

**Error Codes:**
* `400 Bad Request` — empty serial.
* `404 Not Found` — certificate not found.
* `405 Method Not Allowed` — non-GET request.
* `500 Internal Server Error` — retrieval failure.

---
## Certificate Management


### POST /api/v1/certificates/issue
> Signs a PEM-encoded CSR with the intermediate CA. The server never generates or returns a private key.

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `certificates:issue`

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `csr` | string | Yes | PEM-encoded certificate signing request |
| `ttl` | string | No | Certificate lifetime (e.g. `720h`, `24h`). Must not exceed `ca.max_ttl` in `server.yaml` (default `8760h`). |
| `template_id` | string | No | Issuance template identifier |
| `metadata` | object | No | Arbitrary string-keyed metadata map |
| `organization` | string | No | X.509 subject organization (`O`) |
| `organizational_unit` | string | No | X.509 subject organizational unit (`OU`) |
| `country` | string | No | X.509 subject country (`C`) |
| `state` | string | No | X.509 subject state or province (`ST`) |
| `locality` | string | No | X.509 subject locality (`L`) |
| `is_server_auth` | boolean | No | When `true`, adds `ExtKeyUsageServerAuth` |
| `is_client_auth` | boolean | No | When `true`, adds `ExtKeyUsageClientAuth` |

**Example JSON:**
```json
{
  "csr": "-----BEGIN CERTIFICATE REQUEST-----\nMIIC...\n-----END CERTIFICATE REQUEST-----\n",
  "ttl": "720h",
  "template_id": "web-server",
  "organization": "Example Corp",
  "organizational_unit": "Platform",
  "country": "US",
  "state": "CA",
  "locality": "San Francisco",
  "is_server_auth": true,
  "metadata": {
    "owner": "platform-team",
    "environment": "production"
  }
}
```

</details>

#### Response
<details>
  <summary><strong>View Response (201 Created)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificate_pem` | string | Yes | Issued certificate PEM |
| `serial` | string | Yes | Certificate serial number |
| `not_before` | string | Yes | Validity start (RFC3339) |
| `not_after` | string | Yes | Validity end (RFC3339) |

**Example JSON (`data`):**
```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "serial": "12345678901234567890",
  "not_before": "2026-06-02T10:00:00Z",
  "not_after": "2026-07-02T10:00:00Z"
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.
* `400 Bad Request` — missing or invalid CSR, invalid `ttl`, or `ttl` greater than configured `ca.max_ttl`.
* `500 Internal Server Error` — signing failure.

---
### POST /api/v1/certificates/generate
> Generates a private key and CSR in memory, signs the certificate, and returns **both** PEM blobs. The private key is never stored on the server.

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `certificates:issue`

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `common_name` | string | Yes | Primary common name |
| `sans` | string[] | No | Additional DNS names or IP addresses |
| `ttl` | string | No | Certificate lifetime (e.g. `720h`, `30d`). Must not exceed `ca.max_ttl` in `server.yaml` (default `8760h`). |
| `key_algo` | string | Yes | `RSA2048` or `ECDSA256` |
| `organization` | string | No | X.509 subject organization (`O`) |
| `organizational_unit` | string | No | X.509 subject organizational unit (`OU`) |
| `country` | string | No | X.509 subject country (`C`) |
| `state` | string | No | X.509 subject state or province (`ST`) |
| `locality` | string | No | X.509 subject locality (`L`) |
| `is_server_auth` | boolean | No | When `true`, adds `ExtKeyUsageServerAuth` |
| `is_client_auth` | boolean | No | When `true`, adds `ExtKeyUsageClientAuth` |

**Example JSON:**
```json
{
  "common_name": "www.example.com",
  "sans": ["www.example.com", "api.example.com", "203.0.113.10"],
  "ttl": "720h",
  "key_algo": "ECDSA256",
  "organization": "Example Corp",
  "organizational_unit": "Engineering",
  "country": "US",
  "state": "CA",
  "locality": "San Francisco",
  "is_server_auth": true,
  "is_client_auth": true
}
```

</details>

#### Response
<details>
  <summary><strong>View Response (201 Created)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificate_pem` | string | Yes | Issued certificate PEM |
| `private_key_pem` | string | Yes | Generated private key PEM (PKCS#8) |

**Example JSON (`data`):**
```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "private_key_pem": "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----\n"
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.
* `400 Bad Request` — missing `common_name`, invalid `key_algo`, invalid `sans`, invalid `ttl`, or `ttl` greater than configured `ca.max_ttl`.
* `500 Internal Server Error` — generation or signing failure.

---
### POST /api/v1/certificates/issue-with-token
> Signs a CSR using a provisioner-issued single-use JWK token.

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `certificates:issue`

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | string | Yes | Provisioner signing token |
| `csr` | string | Yes | PEM-encoded CSR |
| `ttl` | string | No | Certificate lifetime |
| `template_id` | string | No | Issuance template ID |
| `metadata` | object | No | Arbitrary metadata map |

**Example JSON:**
```json
{
  "token": "eyJhbGciOiJFUzI1NiIs...",
  "csr": "-----BEGIN CERTIFICATE REQUEST-----\nMIIC...\n-----END CERTIFICATE REQUEST-----\n",
  "ttl": "24h",
  "metadata": {"workload": "batch-job-42"}
}
```

</details>

#### Response
<details>
  <summary><strong>View Response (201 Created)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificate_pem` | string | Yes | Issued certificate PEM |
| `serial` | string | Yes | Certificate serial number |
| `not_before` | string | Yes | Validity start (RFC3339) |
| `not_after` | string | Yes | Validity end (RFC3339) |

**Example JSON (`data`):**
```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "serial": "12345678901234567890",
  "not_before": "2026-06-02T10:00:00Z",
  "not_after": "2026-07-02T10:00:00Z"
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.
* `400 Bad Request` — missing `token` or `csr`, or invalid token/CSR.

---
### POST /api/v1/certificates/auto
> Generates an ECDSA P-384 key pair and CSR internally, signs the certificate, and returns **both** the certificate and private key. Used by `arx agent enroll`.

- **Authentication:** Required (Bearer admin JWT only)
- **Permissions:** `Admin` — requires RBAC `certificates:issue`

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `common_name` | string | Yes | Primary common name |
| `dns_sans` | string[] | No | Additional DNS SANs |
| `ip_sans` | string[] | No | IP address SANs |
| `ttl` | string | No | Certificate lifetime |
| `template_id` | string | No | Issuance template ID |
| `metadata` | object | No | Arbitrary metadata map |

**Example JSON:**
```json
{
  "common_name": "www.example.com",
  "dns_sans": ["www.example.com", "example.com"],
  "ip_sans": ["203.0.113.10"],
  "ttl": "2160h",
  "metadata": {"agent": "arx", "host": "web-01"}
}
```

</details>

#### Response
<details>
  <summary><strong>View Response (201 Created)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificate_pem` | string | Yes | Issued certificate PEM |
| `private_key_pem` | string | Yes | Generated ECDSA P-384 private key PEM |
| `serial` | string | Yes | Certificate serial |
| `not_before` | string | Yes | Validity start |
| `not_after` | string | Yes | Validity end |

**Example JSON (`data`):**
```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "private_key_pem": "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----\n",
  "serial": "12345678901234567890",
  "not_before": "2026-06-02T10:00:00Z",
  "not_after": "2027-06-02T10:00:00Z"
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.
* `400 Bad Request` — missing `common_name` or invalid SANs.

---
### POST /api/v1/certificates/revoke
> Revokes a certificate by serial number in the CA database.

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `certificates:revoke`

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `serial` | string | Yes | Certificate serial to revoke |
| `reason` | string | No | Revocation reason string |
| `reason_code` | int | No | RFC 5280 reason code |

**Example JSON:**
```json
{"serial": "12345678901234567890", "reason": "keyCompromise", "reason_code": 1}
```

</details>

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `serial` | string | Yes | Revoked serial |
| `revoked_at` | string | Yes | Revocation timestamp |

**Example JSON (`data`):**
```json
{"serial": "12345678901234567890", "revoked_at": "2026-06-02T11:30:00Z"}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.
* `400 Bad Request` — missing serial.
* `404 Not Found` — certificate not found.
* `409 Conflict` — already revoked.

---
### GET /api/v1/certificates
> Lists all certificates in the CA database with operator metadata.

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `certificates:read`

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificates` | array | Yes | Certificate summaries (see below) |
| `total` | int | Yes | Entry count |

**`certificates[]` Element:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `serial` | string | Yes | Certificate serial |
| `subject` | string | Yes | Distinguished name |
| `dns_names` | string[] | No | DNS SANs |
| `ip_addresses` | string[] | No | IP SANs |
| `not_before` | string (RFC3339) | Yes | Validity start |
| `not_after` | string (RFC3339) | Yes | Validity end |
| `revoked` | bool | Yes | Revocation flag |
| `provisioner_id` | string | No | Provisioner resource ID |
| `provisioner` | string | No | Provisioner name |

**Example JSON (`data`):**
```json
{
  "certificates": [
    {
      "serial": "12345678901234567890",
      "subject": "CN=www.example.com",
      "dns_names": ["www.example.com"],
      "not_before": "2026-06-01T00:00:00Z",
      "not_after": "2026-09-01T00:00:00Z",
      "revoked": false,
      "provisioner_id": "acme/account-abc",
      "provisioner": "acme"
    }
  ],
  "total": 1
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.
* `500 Internal Server Error` — list failure.

---
### POST /api/v1/certificates/lint
> Runs RFC 5280 / CA/Browser Forum lint checks on a PEM certificate. Not available on Windows server builds.

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `certificates:lint`

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificate_pem` | string | Yes | PEM-encoded certificate |

**Example JSON:**
```json
{"certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n"}
```

</details>

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `findings` | array | Yes | Lint findings (see below) |
| `summary` | object | Yes | Severity counts (see below) |

**`findings[]` Element:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `lint` | string | Yes | Lint rule identifier |
| `source` | string | Yes | Lint engine source |
| `severity` | string | Yes | `fatal`, `error`, `warn`, or `notice` |
| `message` | string | No | Human-readable detail |

**`summary` Object:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `errors` | int | Yes | Error count |
| `warnings` | int | Yes | Warning count |
| `notices` | int | Yes | Notice count |
| `fatals` | int | Yes | Fatal count |

**Example JSON (`data`):**
```json
{
  "findings": [
    {
      "lint": "n_subject_common_name_included",
      "source": "zlint",
      "severity": "warn",
      "message": "Subject Common Name is present"
    }
  ],
  "summary": {"errors": 0, "warnings": 1, "notices": 0, "fatals": 0}
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.
* `400 Bad Request` — missing or unparsable `certificate_pem`.
* `501 Not Implemented` — Windows build.
* `500 Internal Server Error` — lint engine failure.

---
## Certificate Renewal & Rekey


### POST /api/v1/certificates/renew
> Re-issues a certificate with the same public key (non-ACME renewal flow).

- **Authentication:** Required — **Admin JWT** or **mTLS client certificate** (service-account API keys are **not** accepted)
- **Permissions:** `Admin | mTLS` — admin JWT requires RBAC `certificates:renew`; mTLS identity is bound to the certificate CN

<details>
  <summary><strong>View Authentication Options</strong></summary>

**Option 1 — Admin JWT**

```http
Authorization: Bearer <jwt>
```

Requires a role with `certificates:renew` (SuperAdmin or CA-Admin). Admins may renew any certificate.

**Option 2 — Mutual TLS**

Present a valid client certificate issued by this CA over HTTPS (`server.tls.enabled: true`). The client certificate **CN must match** the CN of the certificate identified by `certificate_pem` or `renew_token`. Service-account API keys cannot be used on this endpoint.

</details>

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificate_pem` | string | No* | Existing certificate PEM |
| `renew_token` | string | No* | Renewal authorization token |

*At least one of `certificate_pem` or `renew_token` is required.

**Example JSON:**
```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "renew_token": ""
}
```

</details>

#### Response
<details>
  <summary><strong>View Response (201 Created)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificate_pem` | string | Yes | Issued certificate PEM |
| `serial` | string | Yes | Certificate serial number |
| `not_before` | string | Yes | Validity start (RFC3339) |
| `not_after` | string | Yes | Validity end (RFC3339) |

**Example JSON (`data`):**
```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "serial": "12345678901234567890",
  "not_before": "2026-06-02T10:00:00Z",
  "not_after": "2026-07-02T10:00:00Z"
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials (JWT or mTLS).
* `403 Forbidden` — insufficient RBAC permissions, or mTLS CN does not match the target certificate.
* `405 Method Not Allowed` — wrong HTTP method.
* `400 Bad Request` — missing renewal credential or parse errors.

---
### POST /api/v1/certificates/rekey
> Re-issues a certificate with a new key from the supplied CSR.

- **Authentication:** Required — **Admin JWT** or **mTLS client certificate** (service-account API keys are **not** accepted)
- **Permissions:** `Admin | mTLS` — admin JWT requires RBAC `certificates:renew`; mTLS identity is bound to the certificate CN

<details>
  <summary><strong>View Authentication Options</strong></summary>

Same authentication model as `POST /api/v1/certificates/renew`: admin JWT with `certificates:renew`, or mTLS with CN binding to the certificate being rekeyed.

</details>

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificate_pem` | string | No* | Existing certificate PEM |
| `renew_token` | string | No* | Renewal authorization token |
| `csr` | string | Yes | PEM-encoded CSR for the new key |

*At least one of `certificate_pem` or `renew_token` is required.

**Example JSON:**
```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "renew_token": "",
  "csr": "-----BEGIN CERTIFICATE REQUEST-----\nMIIC...\n-----END CERTIFICATE REQUEST-----\n"
}
```

</details>

#### Response
<details>
  <summary><strong>View Response (201 Created)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificate_pem` | string | Yes | Issued certificate PEM |
| `serial` | string | Yes | Certificate serial number |
| `not_before` | string | Yes | Validity start (RFC3339) |
| `not_after` | string | Yes | Validity end (RFC3339) |

**Example JSON (`data`):**
```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "serial": "12345678901234567890",
  "not_before": "2026-06-02T10:00:00Z",
  "not_after": "2026-07-02T10:00:00Z"
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials (JWT or mTLS).
* `403 Forbidden` — insufficient RBAC permissions, or mTLS CN does not match the target certificate.
* `405 Method Not Allowed` — wrong HTTP method.
* `400 Bad Request` — missing `csr` or renewal credential.

---
## Provisioners & Enrollment Status


### POST /api/v1/provisioners/token
> Mints a single-use JWK signing token for JWK/OIDC/K8s-style provisioners.

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `provisioners:token`

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `provisioner` | string | No | Provisioner name (default from CA config) |
| `common_name` | string | Yes | Subject common name |
| `dns_sans` | string[] | No | DNS subject alternative names |
| `ip_sans` | string[] | No | IP subject alternative names |
| `token_ttl` | string | No | Token lifetime (e.g. `5m`) |

**Example JSON:**
```json
{
  "provisioner": "jwk",
  "common_name": "app.example.com",
  "dns_sans": ["app.example.com"],
  "token_ttl": "5m"
}
```

</details>

#### Response
<details>
  <summary><strong>View Response (201 Created)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | string | Yes | JWK signing token |
| `provisioner` | string | Yes | Provisioner name |
| `provisioner_type` | string | Yes | Provisioner type (e.g. `JWK`) |
| `expires_in` | int | Yes | Token lifetime in seconds |
| `audience` | string | Yes | Token audience URL |

**Example JSON (`data`):**
```json
{
  "token": "eyJhbGciOiJFUzI1NiIs...",
  "provisioner": "jwk",
  "provisioner_type": "JWK",
  "expires_in": 300,
  "audience": "https://ca.example.com/1.0/sign"
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.
* `400 Bad Request` — missing `common_name` or invalid SANs/TTL.

---
### GET /api/v1/k8s/status
> Reports Kubernetes Service Account provisioner configuration.

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `enrollment:status`

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | Yes | Whether K8s provisioner is configured |
| `provisioner` | string | No | Provisioner name |
| `review_mode` | string | No | Token review mode |
| `has_public_keys` | bool | Yes | Whether public keys are configured |
| `uses_token_review_api` | bool | Yes | Whether TokenReview API is used |

**Example JSON (`data`):**
```json
{
  "enabled": true,
  "provisioner": "k8s",
  "review_mode": "tokenreview",
  "has_public_keys": true,
  "uses_token_review_api": true
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.

---
### GET /api/v1/acme/status
> Reports ACME directory URL, configured challenges, and EAB requirements.

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `enrollment:status`

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | Yes | Whether ACME is active |
| `directory_url` | string | No | Public directory URL (when enabled) |
| `provisioner` | string | No | Internal provisioner name (`acme`) |
| `challenges` | string[] | No | Supported challenge types |
| `dns_name` | string | No | Directory hostname |
| `require_eab` | bool | Yes | Whether EAB is mandatory |
| `device_attest_enabled` | bool | Yes | Device attestation enabled |
| `attestation_formats` | string[] | No | Supported attestation formats |

**Example JSON (`data`):**
```json
{
  "enabled": true,
  "directory_url": "https://ca.example.com/acme/directory",
  "provisioner": "acme",
  "challenges": ["http-01", "dns-01", "tls-alpn-01"],
  "dns_name": "ca.example.com",
  "require_eab": false,
  "device_attest_enabled": false,
  "attestation_formats": []
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.

---
### POST /api/v1/acme/eab-keys
> Creates an ACME External Account Binding (EAB) credential pair.

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `acme:eab`

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `provisioner` | string | No | ACME provisioner name |
| `reference` | string | No | External reference label |

**Example JSON:**
```json
{"provisioner": "acme", "reference": "customer-42"}
```

</details>

#### Response
<details>
  <summary><strong>View Response (201 Created)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `key_id` | string | Yes | EAB key identifier |
| `provisioner` | string | Yes | Provisioner name |
| `hmac_key` | string | Yes | Base64url-encoded HMAC secret |
| `reference` | string | No | External reference |
| `created_at` | string | Yes | Creation timestamp |

**Example JSON (`data`):**
```json
{
  "key_id": "eab-key-01HXYZ",
  "provisioner": "acme",
  "hmac_key": "base64url-encoded-hmac-secret",
  "reference": "customer-42",
  "created_at": "2026-06-02T10:00:00Z"
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.
* `400 Bad Request` — invalid provisioner or reference.

---
### GET /api/v1/scep/status
> Reports SCEP endpoint availability when a SCEP provisioner is configured.

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `enrollment:status`

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | Yes | Whether SCEP is active |
| `base_url` | string | No | SCEP base URL |
| `provisioner` | string | No | SCEP provisioner name |
| `challenge_hint` | string | No | Challenge configuration hint |

**Example JSON (`data`):**
```json
{
  "enabled": true,
  "base_url": "http://localhost:8080/scep/scep",
  "provisioner": "scep",
  "challenge_hint": "configured"
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.

---
### GET /api/v1/ndes/status
> Reports NDES connector endpoints for AD CS migration paths.

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `enrollment:status`

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | Yes | Whether NDES is active |
| `scep_endpoint` | string | No | MSCEP DLL URL |
| `admin_endpoint` | string | No | MSCEP admin DLL URL |
| `connectors` | string[] | No | Configured connector names |
| `adcs_compatible` | bool | Yes | AD CS compatibility flag |

**Example JSON (`data`):**
```json
{
  "enabled": true,
  "scep_endpoint": "http://localhost:8080/certsrv/mscep/mscep.dll",
  "admin_endpoint": "http://localhost:8080/certsrv/mscep_admin/mscep_admin.dll",
  "connectors": ["default"],
  "adcs_compatible": true
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.

---
## Certificate Templates


### POST /api/v1/templates
> Creates a certificate issuance template (Go `text/template` body rendering JSON SAN/extension data).

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `templates:manage`

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique template name |
| `description` | string | No | Human-readable description |
| `body` | string | Yes | Go text/template rendering issuance JSON |

**Example JSON:**
```json
{
  "name": "web-server",
  "description": "Standard TLS web server template",
  "body": "{\"dnsNames\": [\"{{ .CommonName }}\"], \"keyUsage\": [\"digitalSignature\", \"keyEncipherment\"]}"
}
```

</details>

#### Response
<details>
  <summary><strong>View Response (201 Created)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Template ID |
| `name` | string | Yes | Template name |
| `description` | string | No | Description |
| `body` | string | Yes | Template body |
| `created_at` | string (RFC3339) | Yes | Created timestamp |
| `updated_at` | string (RFC3339) | Yes | Updated timestamp |

**Example JSON (`data`):**
```json
{
  "id": "tpl_01HXYZ",
  "name": "web-server",
  "description": "Standard TLS web server template",
  "body": "{\"dnsNames\": [\"{{ .CommonName }}\"]}",
  "created_at": "2026-06-02T10:00:00Z",
  "updated_at": "2026-06-02T10:00:00Z"
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.
* `400 Bad Request` — validation or duplicate name errors.

---
### GET /api/v1/templates
> Lists all certificate issuance templates.

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `templates:read`

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `templates` | array | Yes | Template objects (same shape as create response) |
| `total` | int | Yes | Template count |

**Example JSON (`data`):**
```json
{
  "templates": [
    {
      "id": "tpl_01HXYZ",
      "name": "web-server",
      "description": "Standard TLS web server template",
      "body": "{\"dnsNames\": [\"{{ .CommonName }}\"]}",
      "created_at": "2026-06-02T10:00:00Z",
      "updated_at": "2026-06-02T10:00:00Z"
    }
  ],
  "total": 1
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.
* `500 Internal Server Error` — list failure.

---
## SSH Certificate Authority


### POST /api/v1/ssh/sign-user
> Issues a short-lived SSH user certificate. Without a body `token`, callers must authenticate so the server can mint an internal sign token.

- **Authentication:** Optional (Bearer JWT or X-API-Key) — required unless `token` is set in the body
- **Permissions:** `None` (with body `token`) | `Admin | Agent` (without body `token`)

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `public_key` | string | Yes | SSH public key (authorized_keys format) |
| `principal` | string | No* | Primary principal |
| `principals` | string[] | No* | Additional principals |
| `ttl` | string | No | Certificate lifetime |
| `token` | string | No | OIDC/provisioner token (bypasses API auth) |
| `provisioner` | string | No | Provisioner name when using `token` |

*Either `principal` or a non-empty `principals` array is required.

**Example JSON:**
```json
{
  "public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ...",
  "principal": "alice",
  "principals": ["alice", "admin"],
  "ttl": "8h",
  "provisioner": "ssh-oidc"
}
```

</details>

#### Response
<details>
  <summary><strong>View Response (201 Created)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificate` | string | Yes | OpenSSH certificate string |
| `certificate_type` | string | Yes | `user` |
| `key_id` | string | Yes | Certificate key ID |
| `principals` | string[] | Yes | Allowed principals |
| `serial` | uint64 | Yes | Certificate serial |
| `valid_after` | string | Yes | Validity start |
| `valid_before` | string | Yes | Validity end |

**Example JSON (`data`):**
```json
{
  "certificate": "ssh-rsa-cert-v01@openssh.com AAAAB3NzaC1yc2EAAAADAQABAAABAQ...",
  "certificate_type": "user",
  "key_id": "alice",
  "principals": ["alice", "admin"],
  "serial": 42,
  "valid_after": "2026-06-02T10:00:00",
  "valid_before": "2026-06-02T18:00:00"
}
```

</details>

**Error Codes:**
* `400 Bad Request` — missing `public_key` or principals.
* `401 Unauthorized` — no credentials and no body `token`.
* `500 Internal Server Error` — signing failure.

---
### POST /api/v1/ssh/sign-host
> Issues an SSH host certificate.

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `ssh:sign_host`

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `public_key` | string | Yes | SSH public key |
| `hostname` | string | No* | Primary hostname principal |
| `principals` | string[] | No* | Host principals |
| `ttl` | string | No | Certificate lifetime |
| `provisioner` | string | No | SSH host provisioner name |

*Either `hostname` or a non-empty `principals` array is required.

**Example JSON:**
```json
{
  "public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...",
  "hostname": "web-01.example.com",
  "principals": ["web-01.example.com", "localhost"],
  "ttl": "8760h"
}
```

</details>

#### Response
<details>
  <summary><strong>View Response (201 Created)</strong></summary>

**Properties (`data`):** same shape as `sign-user`; `certificate_type` is `host`.

**Example JSON (`data`):**
```json
{
  "certificate": "ssh-ed25519-cert-v01@openssh.com AAAAC3NzaC1lZDI1NTE5AAAAI...",
  "certificate_type": "host",
  "key_id": "web-01.example.com",
  "principals": ["web-01.example.com", "localhost"],
  "serial": 7,
  "valid_after": "2026-06-02T10:00:00",
  "valid_before": "2027-06-02T10:00:00"
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.
* `400 Bad Request` — missing `public_key` or hostname/principals.

---
### POST /api/v1/ssh/inspect
> Decodes an SSH certificate and returns metadata.

- **Authentication:** Required (Bearer JWT or X-API-Key)
- **Permissions:** `Admin | Agent` — requires RBAC `ssh:inspect`

#### Request
<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificate` | string | Yes | OpenSSH certificate string |

**Example JSON:**
```json
{"certificate": "ssh-rsa-cert-v01@openssh.com AAAAB3NzaC1yc2EAAAADAQABAAABAQ..."}
```

</details>

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificate_type` | string | Yes | `user` or `host` |
| `key_id` | string | Yes | Key ID |
| `principals` | string[] | Yes | Principals |
| `serial` | uint64 | Yes | Serial number |
| `valid_after` | string | Yes | Validity start |
| `valid_before` | string | Yes | Validity end |
| `public_key_type` | string | Yes | Underlying public key type |
| `critical_options` | object | No | Critical options map |
| `extensions` | object | No | Extensions map |
| `signature_key` | string | No | CA signature key |

**Example JSON (`data`):**
```json
{
  "certificate_type": "user",
  "key_id": "alice",
  "principals": ["alice"],
  "serial": 42,
  "valid_after": "2026-06-02T10:00:00",
  "valid_before": "2026-06-02T18:00:00",
  "public_key_type": "ssh-rsa",
  "critical_options": {},
  "extensions": {"permit-pty": ""},
  "signature_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ..."
}
```

</details>

**Error Codes:**
* `401 Unauthorized` — missing or invalid credentials.
* `403 Forbidden` — insufficient RBAC permissions.
* `405 Method Not Allowed` — wrong HTTP method.
* `400 Bad Request` — missing or invalid certificate.

---
### GET /api/v1/ssh/roots
> Returns SSH CA public keys for `known_hosts` / `authorized_keys` trust configuration.

- **Authentication:** Not required
- **Permissions:** `None`

#### Response
<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties (`data`):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `user_keys` | array | Yes | SSH user CA keys (see below) |
| `host_keys` | array | Yes | SSH host CA keys (see below) |

**`user_keys[]` / `host_keys[]` Element:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `public_key` | string | Yes | Public key (authorized_keys format) |
| `key_type` | string | Yes | Key algorithm |
| `fingerprint` | string | Yes | SHA256 fingerprint |

**Example JSON (`data`):**
```json
{
  "user_keys": [
    {
      "public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ...",
      "key_type": "ssh-rsa",
      "fingerprint": "SHA256:abcdef1234567890"
    }
  ],
  "host_keys": [
    {
      "public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...",
      "key_type": "ssh-ed25519",
      "fingerprint": "SHA256:fedcba0987654321"
    }
  ]
}
```

</details>

**Error Codes:**
* `404 Not Found` — SSH CA not configured.
* `405 Method Not Allowed` — non-GET request.
* `500 Internal Server Error` — retrieval failure.

---

## OCSP Responder


RFC 6960 endpoints. Responses are **DER** `application/ocsp-response`, not the JSON envelope.

### POST /ocsp
> Submits an OCSP request in the request body.

- **Authentication:** Not required
- **Permissions:** `None`

#### Request

<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**Body:** `application/ocsp-request` — DER-encoded OCSP request (max 64 KiB).

</details>

#### Response

<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Body:** DER `application/ocsp-response`.

**Headers:** `Content-Type: application/ocsp-response`, `Cache-Control: no-cache`

</details>

**Error Codes:**

* `400 Bad Request` — unreadable or invalid OCSP request.
* `405 Method Not Allowed` — non-POST request.

---

### GET /ocsp/{request}
> Submits an OCSP request as a URL-safe Base64 (or standard Base64) path segment.

- **Authentication:** Not required
- **Permissions:** `None`
- **Query Parameters:** `request` (path) — Base64url, padded Base64url, or standard Base64 OCSP request DER

#### Response

<details>
  <summary><strong>View Response (200 OK)</strong></summary>

Same as `POST /ocsp`.

</details>

**Error Codes:**

* `400 Bad Request` — empty path, decode failure, or invalid OCSP request.
* `405 Method Not Allowed` — non-GET request.

---

## ACMEv2 (RFC 8555)


Mounted at `/acme/` when ACME is enabled (`CA_API_ACME_DISABLED` is not `true` and an ACME provisioner exists). Public URLs use a **flat** layout (no provisioner name in paths). Mutating requests use **JWS** (`Content-Type: application/jose+json`), not the arx JSON envelope.

**Internal provisioner name:** `acme`

### GET /acme/directory

> Returns the ACME directory object listing all resource URLs.

- **Authentication:** Not required
- **Permissions:** `None`

#### Response

<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `newNonce` | string | Yes | Nonce endpoint URL |
| `newAccount` | string | Yes | Account registration URL |
| `newOrder` | string | Yes | Order creation URL |
| `revokeCert` | string | Yes | Certificate revocation URL |
| `keyChange` | string | Yes | Account key change URL |
| `meta` | object | No | Directory metadata (see below) |

**`meta` Object:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `termsOfService` | string | No | Terms of service URL |
| `website` | string | No | CA website URL |
| `caaIdentities` | string[] | No | CAA identities |
| `externalAccountRequired` | bool | No | Whether EAB is required |

**Example JSON:**
```json
{
  "newNonce": "https://ca.example.com/acme/new-nonce",
  "newAccount": "https://ca.example.com/acme/new-account",
  "newOrder": "https://ca.example.com/acme/new-order",
  "revokeCert": "https://ca.example.com/acme/revoke-cert",
  "keyChange": "https://ca.example.com/acme/key-change",
  "meta": {
    "termsOfService": "https://ca.example.com/tos",
    "website": "https://ca.example.com",
    "caaIdentities": ["ca.example.com"],
    "externalAccountRequired": false
  }
}
```

</details>

**Error Codes:**

* `404 Not Found` — ACME disabled.
* `500 Internal Server Error` — provisioner load failure.

---

### HEAD /acme/new-nonce
> Issues a fresh anti-replay nonce for JWS requests (header only, no body).

- **Authentication:** Not required
- **Permissions:** `None`

#### Response
<details>
  <summary><strong>View Response (204 No Content)</strong></summary>

**Header:** `Replay-Nonce: <nonce>`

</details>

**Error Codes:**
* `404 Not Found` — ACME disabled.

---
### GET /acme/new-nonce
> Issues a fresh anti-replay nonce for JWS requests (same semantics as HEAD).

- **Authentication:** Not required
- **Permissions:** `None`

#### Response
<details>
  <summary><strong>View Response (204 No Content)</strong></summary>

**Header:** `Replay-Nonce: <nonce>`

</details>

**Error Codes:**
* `404 Not Found` — ACME disabled.

---

### POST /acme/new-account

> Creates or looks up an ACME account (empty JWS payload looks up an existing account per RFC 8555).

- **Authentication:** Required (JWS signed with account key; EAB when `externalAccountRequired` is true)
- **Permissions:** `None`

#### Request

<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

**JWS payload (decoded JSON):**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `termsOfServiceAgreed` | bool | No | ToS acceptance |
| `contact` | string[] | No | Contact URIs |
| `externalAccountBinding` | object | No | EAB JWS object (see below) |

**`externalAccountBinding` Object:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `protected` | string | Yes | Base64url-encoded protected header |
| `payload` | string | Yes | Base64url-encoded payload |
| `signature` | string | Yes | Base64url-encoded signature |

**Example JSON (decoded JWS payload):**
```json
{
  "termsOfServiceAgreed": true,
  "contact": ["mailto:admin@example.com"]
}
```

</details>

#### Response

<details>
  <summary><strong>View Response (201 Created or 200 OK)</strong></summary>

**Properties:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | Account status (`valid`, etc.) |
| `contact` | string[] | No | Contact URIs |
| `orders` | string | Yes | Orders collection URL |
| `key` | object | Yes | Account public JWK |

**Example JSON:**
```json
{
  "status": "valid",
  "contact": ["mailto:admin@example.com"],
  "orders": "https://ca.example.com/acme/account/acc-01/orders",
  "key": {
    "kty": "EC",
    "crv": "P-256",
    "x": "WKn-ZIGevcwGIhoz7tZp6ueBrvPgtmqlAGTb4YiZKl7s",
    "y": "b2z7pECbQgynqjY3l-9SrS82l7yBYe5i30WnI5V6I0s"
  }
}
```

</details>

**Error Codes:**

* `400 Bad Request` — malformed JWS or payload.
* `401 Unauthorized` — invalid EAB or account key.
* `403 Forbidden` — policy violation.
* `404 Not Found` — ACME disabled.
* `409 Conflict` — account URL mismatch.
* `500 Internal Server Error` — server error.

---

### POST /acme/new-order

> Creates a certificate order for one or more identifiers.

- **Authentication:** Required (JWS with account `kid`)
- **Permissions:** `None`

#### Request

<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `identifiers` | array | Yes | Identifiers to authorize (see below) |
| `notBefore` | string | No | Certificate validity start |
| `notAfter` | string | No | Certificate validity end |

**`identifiers[]` Element:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | `dns` or `ip` |
| `value` | string | Yes | Identifier value |

**Example JSON (decoded JWS payload):**
```json
{
  "identifiers": [{"type": "dns", "value": "www.example.com"}],
  "notBefore": "2026-06-02T00:00:00Z",
  "notAfter": "2026-09-02T00:00:00Z"
}
```

</details>

#### Response

<details>
  <summary><strong>View Response (201 Created)</strong></summary>

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | Yes | Order status |
| `expires` | string | Yes | Order expiration |
| `identifiers` | array | Yes | Requested identifiers |
| `authorizations` | string[] | Yes | Authorization URLs |
| `finalize` | string | Yes | Finalize URL |
| `certificate` | string | null | Certificate URL when issued |

**Example JSON:**
```json
{
  "status": "pending",
  "expires": "2026-06-03T00:00:00Z",
  "identifiers": [{"type": "dns", "value": "www.example.com"}],
  "authorizations": ["https://ca.example.com/acme/authz/authz-01"],
  "finalize": "https://ca.example.com/acme/order/ord-01/finalize",
  "certificate": null
}
```

</details>

**Error Codes:**

* `400 Bad Request` — invalid identifiers.
* `401 Unauthorized` — invalid JWS.
* `403 Forbidden` — unauthorized identifier.
* `404 Not Found` — ACME disabled.
* `500 Internal Server Error` — order creation failure.

---

### POST /acme/order/{orderID}/finalize

> Submits a CSR to finalize a ready order.

- **Authentication:** Required (JWS with account `kid`)
- **Permissions:** `None`
- **Query Parameters:** `orderID` (path) — order identifier

#### Request

<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `csr` | string | Yes | Base64url-encoded CSR DER |

**Example JSON (decoded JWS payload):**
```json
{"csr": "MIIB...base64url-encoded-CSR-DER..."}
```

</details>

#### Response

<details>
  <summary><strong>View Response (200 OK)</strong></summary>

Order object with `status` `valid` and `certificate` URL set when successful.

**Example JSON:**
```json
{
  "status": "valid",
  "expires": "2026-06-03T00:00:00Z",
  "identifiers": [{"type": "dns", "value": "www.example.com"}],
  "authorizations": ["https://ca.example.com/acme/authz/authz-01"],
  "finalize": "https://ca.example.com/acme/order/ord-01/finalize",
  "certificate": "https://ca.example.com/acme/certificate/cert-01"
}
```

</details>

**Error Codes:**

* `400 Bad Request` — invalid CSR or order state.
* `401 Unauthorized` — invalid JWS.
* `403 Forbidden` — finalize not allowed.
* `404 Not Found` — unknown order.
* `500 Internal Server Error` — signing failure.

---

### GET /acme/order/{orderID}

> Polls order status (typically via JWS POST-as-GET per RFC 8555).

- **Authentication:** Required (JWS with account `kid`)
- **Permissions:** `None`
- **Query Parameters:** `orderID` (path) — order identifier

#### Response

<details>
  <summary><strong>View Response (200 OK)</strong></summary>

Same shape as order creation/finalize responses.

</details>

**Error Codes:**

* `401 Unauthorized` — invalid JWS.
* `404 Not Found` — unknown order.
* `500 Internal Server Error` — retrieval failure.

---

### GET /acme/authz/{authzID}

> Returns authorization status and associated challenges.

- **Authentication:** Required (JWS with account `kid`)
- **Permissions:** `None`
- **Query Parameters:** `authzID` (path) — authorization identifier

#### Response

<details>
  <summary><strong>View Response (200 OK)</strong></summary>

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `identifier` | object | Yes | Authorized identifier (see below) |
| `status` | string | Yes | Authorization status |
| `expires` | string | Yes | Authorization expiration |
| `challenges` | array | Yes | Challenge objects (see below) |
| `wildcard` | bool | No | Whether identifier is a wildcard |

**`identifier` Object:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | `dns` or `ip` |
| `value` | string | Yes | Identifier value |

**`challenges[]` Element:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | `http-01`, `dns-01`, or `tls-alpn-01` |
| `url` | string | Yes | Challenge URL |
| `status` | string | Yes | Challenge status |
| `token` | string | Yes | Challenge token |
| `validated` | string | No | Validation timestamp when valid |

**Example JSON:**
```json
{
  "identifier": {"type": "dns", "value": "www.example.com"},
  "status": "pending",
  "expires": "2026-06-03T00:00:00Z",
  "challenges": [
    {
      "type": "http-01",
      "url": "https://ca.example.com/acme/challenge/authz-01/ch-01",
      "status": "pending",
      "token": "token-abc123"
    }
  ],
  "wildcard": false
}
```

</details>

**Error Codes:**

* `401 Unauthorized` — invalid JWS.
* `404 Not Found` — unknown authorization.
* `500 Internal Server Error` — retrieval failure.

---

### POST /acme/challenge/{authzID}/{challengeID}

> Triggers challenge validation. arx performs outbound http-01, dns-01, or tls-alpn-01 checks.

- **Authentication:** Required (JWS with account `kid`)
- **Permissions:** `None`
- **Query Parameters:** `authzID` (path), `challengeID` (path)

#### Request

<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

Empty JSON object `{}` (JWS payload may be empty string).

</details>

#### Response

<details>
  <summary><strong>View Response (200 OK)</strong></summary>

Challenge object with updated `status` (`valid` or `invalid`).

**Example JSON:**
```json
{
  "type": "http-01",
  "url": "https://ca.example.com/acme/challenge/authz-01/ch-01",
  "status": "valid",
  "validated": "2026-06-02T10:05:00Z",
  "token": "token-abc123"
}
```

</details>

**Error Codes:**

* `400 Bad Request` — malformed JWS.
* `401 Unauthorized` — invalid JWS.
* `403 Forbidden` — challenge not acceptable.
* `404 Not Found` — unknown challenge.
* `500 Internal Server Error` — validation error.

---

### GET /acme/certificate/{certID}

> Downloads the issued certificate chain (PEM or DER per `Accept` header).

- **Authentication:** Required (JWS POST-as-GET per RFC 8555)
- **Permissions:** `None`
- **Query Parameters:** `certID` (path) — certificate resource identifier

#### Response

<details>
  <summary><strong>View Response (200 OK)</strong></summary>

PEM certificate chain (`application/pem-certificate-chain`) or DER per client `Accept` negotiation.

</details>

**Error Codes:**

* `401 Unauthorized` — invalid JWS.
* `404 Not Found` — unknown certificate.
* `500 Internal Server Error` — download failure.

---

### POST /acme/revoke-cert

> Revokes a certificate via ACME.

- **Authentication:** Required (JWS with account `kid` or certificate key `jwk`)
- **Permissions:** `None`

#### Request

<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `certificate` | string | Yes | Base64url-encoded certificate DER |
| `reason` | int | No | Revocation reason code |

**Example JSON (decoded JWS payload):**
```json
{"certificate": "MIIB...base64url-encoded-cert-DER...", "reason": 1}
```

</details>

#### Response

<details>
  <summary><strong>View Response (200 OK)</strong></summary>

Empty body on success.

</details>

**Error Codes:**

* `400 Bad Request` — malformed request.
* `401 Unauthorized` — invalid JWS.
* `403 Forbidden` — revocation not permitted.
* `404 Not Found` — ACME disabled.
* `500 Internal Server Error` — revocation failure.

---

### POST /acme/key-change

> Rotates the ACME account key (RFC 8555 key rollover).

- **Authentication:** Required (JWS signed by old and new keys)
- **Permissions:** `None`

#### Request

<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `account` | string | Yes | Account URL |
| `oldKey` | object | Yes | Previous account public JWK |

**Example JSON (decoded JWS payload):**
```json
{
  "account": "https://ca.example.com/acme/account/acc-01",
  "oldKey": {
    "kty": "EC",
    "crv": "P-256",
    "x": "WKn-ZIGevcwGIhoz7tZp6ueBrvPgtmqlAGTb4YiZKl7s",
    "y": "b2z7pECbQgynqjY3l-9SrS82l7yBYe5i30WnI5V6I0s"
  }
}
```

</details>

#### Response

<details>
  <summary><strong>View Response (200 OK)</strong></summary>

Updated account object.

</details>

**Error Codes:**

* `400 Bad Request` — invalid rollover proof.
* `401 Unauthorized` — invalid JWS.
* `403 Forbidden` — rollover denied.
* `404 Not Found` — ACME disabled.
* `500 Internal Server Error` — update failure.

---

### POST /acme/account/{accountID}

> Updates account contact information or deactivates the account.

- **Authentication:** Required (JWS with account `kid`)
- **Permissions:** `None`
- **Query Parameters:** `accountID` (path) — account identifier

#### Request

<details>
  <summary><strong>View Request Schema & JSON</strong></summary>

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `contact` | string[] | No | Updated contact URIs |
| `status` | string | No | Set to `deactivated` to deactivate |

**Example JSON (decoded JWS payload):**
```json
{"contact": ["mailto:ops@example.com"], "status": "deactivated"}
```

</details>

#### Response

<details>
  <summary><strong>View Response (200 OK)</strong></summary>

**Example JSON:**
```json
{
  "status": "deactivated",
  "contact": ["mailto:ops@example.com"],
  "orders": "https://ca.example.com/acme/account/acc-01/orders",
  "key": {
    "kty": "EC",
    "crv": "P-256",
    "x": "WKn-ZIGevcwGIhoz7tZp6ueBrvPgtmqlAGTb4YiZKl7s",
    "y": "b2z7pECbQgynqjY3l-9SrS82l7yBYe5i30WnI5V6I0s"
  }
}
```

</details>

**Error Codes:**

* `400 Bad Request` — invalid payload.
* `401 Unauthorized` — invalid JWS.
* `404 Not Found` — unknown account.
* `500 Internal Server Error` — update failure.

---

ACME errors use `application/problem+json` problem documents (`type`, `detail`, `status`) per RFC 8555, not the arx `error`/`data` envelope.

---

## SCEP (optional)


Registered at `/scep/` when a SCEP provisioner exists in `ca.json`. Uses the **smallstep SCEP** protocol (PKCS#7/CMS), not JSON.

**Base path:** `/scep/{provisioner}` (default provisioner name: `scep`)

| Operation | Description |
| --------- | ----------- |
| GetCACaps | Query CA capabilities |
| GetCACert | Retrieve CA certificate(s) |
| PKIOperation | Enrollment and renewal messages |

- **Authentication:** SCEP challenge password when `CA_API_SCEP_CHALLENGE` is configured
- **Permissions:** `None` (protocol credentials, not arx RBAC)

**Status discovery (JSON API):** `GET /api/v1/scep/status`

**Error Codes:** Protocol-specific HTTP statuses; `404` when SCEP is not enabled.

---

## NDES (optional)


Registered at `/certsrv/` for Microsoft AD CS–compatible paths when NDES connectors are configured.

### GET /certsrv/mscep/mscep.dll

> Proxies to the configured SCEP connector (same protocol as `/scep/scep`).

- **Authentication:** SCEP challenge password
- **Permissions:** `None`

#### Response

<details>
  <summary><strong>View Response (200 OK)</strong></summary>

SCEP PKCS#7 payloads (not JSON).

</details>

**Error Codes:**

* `503 Service Unavailable` — connector not configured.
* `404 Not Found` — NDES disabled.

---

### GET /certsrv/mscep_admin/mscep_admin.dll

> Returns the SCEP challenge password for NDES enrollment workflows.

- **Authentication:** Optional `X-NDES-Admin-Secret` header or `?secret=` query when admin secret is configured
- **Permissions:** `None`

#### Response

<details>
  <summary><strong>View Response (200 OK)</strong></summary>

`text/plain` challenge password body.

</details>

**Error Codes:**

* `401 Unauthorized` — invalid admin secret.
* `405 Method Not Allowed` — non-GET request.
* `503 Service Unavailable` — `CA_API_SCEP_CHALLENGE` not set.

**Status discovery (JSON API):** `GET /api/v1/ndes/status`

---

## Related Documentation


- [acme.md](acme.md) — ACME challenge behavior, environment variables, reverse-proxy examples
- [cli_reference.md](cli_reference.md) — `arx` CLI and agent commands
- [architecture.md](architecture.md) — system layout and databases
