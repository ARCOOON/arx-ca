# Arx CA HTTP API Reference

Interactive reference for every HTTP endpoint exposed by **arx-ca-server**. Routes are registered in `internal/cmd/arx/server_start.go`. JSON management APIs use a standard envelope; ACME, OCSP, CRL, SCEP, and NDES use protocol-specific formats.

## Base URL and Versioning

| Item | Value |
| ---- | ----- |
| **API prefix** | `/api/v1` |
| **Default listen** | `http://localhost:8080` (configurable via server config) |
| **Content-Type (JSON APIs)** | `application/json; charset=utf-8` |

Example base: `https://ca.example.com/api/v1`

## Standard JSON Envelope

All `/api/v1/*` handlers return:

```json
{
  "error": null,
  "data": { }
}
```

On failure, `error` is a string message and `data` is `null`:

```json
{
  "error": "invalid email or password",
  "data": null
}
```

The examples below show the **`data`** payload only. Wrap successful responses in the envelope when calling the API.

## Authentication

### Admin JWT (interactive / bootstrap users)

1. `POST /api/v1/auth/login` with email and password.
2. Use the returned token on protected routes:

```http
Authorization: Bearer <token>
```

- **Token type:** `Bearer` (field `token_type` in login response).
- **Algorithm:** HS256 JWT; issuer and expiry come from server security config.
- **Roles:** JWT may include `roles` (`SuperAdmin`, `CA-Admin`, `Revocation-Manager`, `Read-Only`). Endpoints enforce RBAC permissions derived from roles.

### Service account API key

Automation clients may use either header:

```http
X-API-Key: <api_key>
```

or

```http
Authorization: Bearer <api_key>
```

API keys are created via `POST /api/v1/auth/service-accounts` (admin JWT required). Keys are shown once at creation.

### Permission matrix (RBAC)

| Permission | Typical roles |
| ---------- | --------------- |
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

Missing or invalid credentials → `401`. Valid credentials without permission → `403`.

---

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

Returns process uptime, memory statistics, API version, and PKI engine status.

**Authentication:** No

**Parameters / Query Strings:** None

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "uptime": {
    "seconds": 3600,
    "human": "1h 0m 0s"
  },
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
  "api": {
    "status": "healthy",
    "version": "v1"
  },
  "ca_backend": {
    "status": "healthy",
    "message": "",
    "engine": "step-ca",
    "initialized": true
  }
}
```

</details>

**Error codes:**

- `405 Method Not Allowed` — non-GET request
- `500 Internal Server Error` — unexpected failure encoding response

---

## Authentication Endpoints

### POST /api/v1/auth/login

Authenticates an admin user and returns a short-lived JWT.

**Authentication:** No

**Parameters / Query Strings:** None

<details>
<summary><strong>Request Body (JSON)</strong></summary>

```json
{
  "email": "admin@example.com",
  "password": "secretpassword"
}
```

</details>

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2026-06-03T12:00:00Z",
  "token_type": "Bearer",
  "roles": [
    "SuperAdmin"
  ]
}
```

</details>

**Error codes:**

- `400 Bad Request` — missing/invalid JSON, empty body, unknown fields
- `401 Unauthorized` — invalid email or password
- `405 Method Not Allowed` — non-POST request
- `500 Internal Server Error` — credential store or token generation failure

---

### POST /api/v1/auth/service-accounts

Creates a service account and returns a one-time API key.

**Authentication:** Required — Admin JWT (`Authorization: Bearer <token>`). Permission: `service_accounts:manage`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Request Body (JSON)</strong></summary>

```json
{
  "name": "ci-deploy-bot",
  "roles": [
    "CA-Admin"
  ]
}
```

</details>

<details>
<summary><strong>Response: 201 Created</strong></summary>

```json
{
  "id": "sa_01HXYZABCDEF",
  "name": "ci-deploy-bot",
  "roles": [
    "CA-Admin"
  ],
  "api_key": "arx_sk_live_abcdefghijklmnopqrstuvwxyz",
  "created_at": "2026-06-02T10:00:00Z"
}
```

</details>

**Error codes:**

- `400 Bad Request` — invalid name or JSON body
- `401 Unauthorized` — missing or invalid JWT
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-POST request
- `409 Conflict` — duplicate service account name
- `500 Internal Server Error` — creation failure

---

## CA Certificates & Revocation Lists

### GET /api/v1/ca/root

Returns the Root CA certificate PEM for trust store installation.

**Authentication:** No

**Parameters / Query Strings:** None

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "pem": "-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----\n"
}
```

</details>

**Error codes:**

- `405 Method Not Allowed` — non-GET request
- `500 Internal Server Error` — root certificate unavailable

---

### GET /api/v1/ca/crl

Returns the current Certificate Revocation List. **Not** wrapped in the JSON envelope.

**Authentication:** No

**Parameters / Query Strings:**

| Name | Type | Description |
| ---- | ---- | ----------- |
| `pem` | flag (presence) | If present, response is PEM-encoded CRL; otherwise DER (`application/pkix-crl`) |

**Response (success):**

- **Status:** `200 OK`
- **Body:** raw CRL bytes (DER or PEM)
- **Headers:** `Content-Type`, `Content-Disposition`, `Expires` (CRL next-update)

**Error codes:**

- `405 Method Not Allowed` — non-GET request
- `500 Internal Server Error` — CRL unavailable (plain text body via `http.Error`)

---

## Public / Agent Read-Only API

Unauthenticated endpoints used by the **arx agent** (`internal/agent/api`) for trust installation and certificate discovery. They never expose private keys or signing operations.

### GET /api/v1/public/ca/intermediate

Returns the Intermediate CA certificate PEM.

**Authentication:** No

**Parameters / Query Strings:** None

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "pem": "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----\n"
}
```

</details>

**Error codes:**

- `405 Method Not Allowed` — non-GET request
- `500 Internal Server Error` — intermediate certificate unavailable

---

### GET /api/v1/public/certificates

Lists issued certificates with public metadata (no PEM private data).

**Authentication:** No

**Parameters / Query Strings:** None

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "certificates": [
    {
      "serial": "12345678901234567890",
      "subject": "CN=www.example.com",
      "dns_names": [
        "www.example.com",
        "example.com"
      ],
      "ip_addresses": [
        "203.0.113.10"
      ],
      "not_before": "2026-06-01T00:00:00Z",
      "not_after": "2026-09-01T00:00:00Z",
      "revoked": false
    }
  ],
  "total": 1
}
```

</details>

**Error codes:**

- `405 Method Not Allowed` — non-GET request
- `500 Internal Server Error` — database or CA engine failure

---

### GET /api/v1/public/certificates/{serial}

Downloads a single certificate PEM by serial number.

**Authentication:** No

**Parameters / Query Strings:**

| Name | Location | Description |
| ---- | -------- | ----------- |
| `serial` | path | Certificate serial (decimal string) |

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "serial": "12345678901234567890",
  "not_before": "",
  "not_after": ""
}
```

</details>

Note: `not_before` and `not_after` may be empty strings when only PEM is returned from the backing store.

**Error codes:**

- `400 Bad Request` — empty serial
- `404 Not Found` — certificate not found
- `405 Method Not Allowed` — non-GET request
- `500 Internal Server Error` — retrieval failure

---

## Certificate Management

Protected routes accept **Admin JWT** or **Service account API key** with the listed permission.

### POST /api/v1/certificates/issue

Signs a PEM-encoded CSR with the intermediate CA.

**Authentication:** Required — JWT or API key. Permission: `certificates:issue`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Request Body (JSON)</strong></summary>

```json
{
  "csr": "-----BEGIN CERTIFICATE REQUEST-----\nMIIC...\n-----END CERTIFICATE REQUEST-----\n",
  "ttl": "720h",
  "template_id": "web-server",
  "metadata": {
    "owner": "platform-team",
    "environment": "production"
  }
}
```

</details>

<details>
<summary><strong>Response: 201 Created</strong></summary>

```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "serial": "12345678901234567890",
  "not_before": "2026-06-02T10:00:00Z",
  "not_after": "2026-07-02T10:00:00Z"
}
```

</details>

**Error codes:**

- `400 Bad Request` — missing CSR, invalid CSR, validation errors
- `401 Unauthorized` — authentication required or invalid credentials
- `403 Forbidden` — insufficient permissions or CA policy denial
- `405 Method Not Allowed` — non-POST request
- `500 Internal Server Error` — signing failure

---

### POST /api/v1/certificates/issue-with-token

Signs a CSR using a provisioner-issued single-use JWK token.

**Authentication:** Required — JWT or API key. Permission: `certificates:issue`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Request Body (JSON)</strong></summary>

```json
{
  "token": "eyJhbGciOiJFUzI1NiIs...",
  "csr": "-----BEGIN CERTIFICATE REQUEST-----\nMIIC...\n-----END CERTIFICATE REQUEST-----\n",
  "ttl": "24h",
  "template_id": "",
  "metadata": {
    "workload": "batch-job-42"
  }
}
```

</details>

<details>
<summary><strong>Response: 201 Created</strong></summary>

```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "serial": "98765432109876543210",
  "not_before": "2026-06-02T10:00:00Z",
  "not_after": "2026-06-03T10:00:00Z"
}
```

</details>

**Error codes:**

- `400 Bad Request` — missing `token` or `csr`, invalid CSR or token
- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-POST request
- `500 Internal Server Error` — signing failure

---

### POST /api/v1/certificates/auto

Generates a key pair and signs a certificate in one step. Used by `arx agent enroll`.

**Authentication:** Required — JWT or API key. Permission: `certificates:issue`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Request Body (JSON)</strong></summary>

```json
{
  "common_name": "www.example.com",
  "dns_sans": [
    "www.example.com",
    "example.com"
  ],
  "ip_sans": [
    "203.0.113.10"
  ],
  "ttl": "2160h",
  "template_id": "",
  "metadata": {
    "agent": "arx",
    "host": "web-01"
  }
}
```

</details>

<details>
<summary><strong>Response: 201 Created</strong></summary>

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

**Error codes:**

- `400 Bad Request` — missing `common_name`, invalid SANs
- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-POST request
- `500 Internal Server Error` — key generation or signing failure

---

### POST /api/v1/certificates/revoke

Revokes a certificate by serial number in the CA database.

**Authentication:** Required — JWT or API key. Permission: `certificates:revoke`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Request Body (JSON)</strong></summary>

```json
{
  "serial": "12345678901234567890",
  "reason": "keyCompromise",
  "reason_code": 1
}
```

</details>

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "serial": "12345678901234567890",
  "revoked_at": "2026-06-02T11:30:00Z"
}
```

</details>

**Error codes:**

- `400 Bad Request` — missing serial
- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `404 Not Found` — certificate not found
- `409 Conflict` — certificate already revoked
- `405 Method Not Allowed` — non-POST request
- `500 Internal Server Error` — revocation failure

---

### GET /api/v1/certificates

Lists all certificates in the CA database with full operator metadata.

**Authentication:** Required — JWT or API key. Permission: `certificates:read`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "certificates": [
    {
      "serial": "12345678901234567890",
      "subject": "CN=www.example.com",
      "dns_names": [
        "www.example.com"
      ],
      "ip_addresses": [],
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

**Error codes:**

- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-GET request
- `500 Internal Server Error` — list failure

---

### POST /api/v1/certificates/lint

Runs RFC 5280 / CA/Browser Forum lint checks on a PEM certificate. **Not available on Windows server builds.**

**Authentication:** Required — JWT or API key. Permission: `certificates:lint`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Request Body (JSON)</strong></summary>

```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n"
}
```

</details>

<details>
<summary><strong>Response: 200 OK</strong></summary>

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
  "summary": {
    "errors": 0,
    "warnings": 1,
    "notices": 0,
    "fatals": 0
  }
}
```

</details>

**Error codes:**

- `400 Bad Request` — missing or unparsable `certificate_pem`
- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-POST request
- `501 Not Implemented` — Windows build (linting disabled)
- `500 Internal Server Error` — lint engine failure

---

## Certificate Renewal & Rekey

### POST /api/v1/certificates/renew

Re-issues a certificate with the same public key (non-ACME renewal flow).

**Authentication:** Required — JWT or API key. Permission: `certificates:renew`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Request Body (JSON)</strong></summary>

```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "renew_token": ""
}
```

Either `certificate_pem` or `renew_token` must be provided (not both empty).

</details>

<details>
<summary><strong>Response: 201 Created</strong></summary>

```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "serial": "12345678901234567890",
  "not_before": "2026-06-02T12:00:00Z",
  "not_after": "2026-07-02T12:00:00Z"
}
```

</details>

**Error codes:**

- `400 Bad Request` — missing `certificate_pem` and `renew_token`, parse errors
- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions or renewal denied
- `405 Method Not Allowed` — non-POST request
- `500 Internal Server Error` — renewal failure

---

### POST /api/v1/certificates/rekey

Re-issues a certificate with a new key from the supplied CSR.

**Authentication:** Required — JWT or API key. Permission: `certificates:renew`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Request Body (JSON)</strong></summary>

```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "renew_token": "",
  "csr": "-----BEGIN CERTIFICATE REQUEST-----\nMIIC...\n-----END CERTIFICATE REQUEST-----\n"
}
```

</details>

<details>
<summary><strong>Response: 201 Created</strong></summary>

```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----\n",
  "serial": "12345678901234567890",
  "not_before": "2026-06-02T12:00:00Z",
  "not_after": "2026-07-02T12:00:00Z"
}
```

</details>

**Error codes:**

- `400 Bad Request` — missing `csr`, missing renewal credential, parse errors
- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-POST request
- `500 Internal Server Error` — rekey failure

---

## Provisioners & Enrollment Status

### POST /api/v1/provisioners/token

Mints a single-use JWK signing token for JWK/OIDC/K8s-style provisioners.

**Authentication:** Required — JWT or API key. Permission: `provisioners:token`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Request Body (JSON)</strong></summary>

```json
{
  "provisioner": "jwk",
  "common_name": "app.example.com",
  "dns_sans": [
    "app.example.com"
  ],
  "ip_sans": [],
  "token_ttl": "5m"
}
```

</details>

<details>
<summary><strong>Response: 201 Created</strong></summary>

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

**Error codes:**

- `400 Bad Request` — missing `common_name`, unknown provisioner, invalid TTL/SANs
- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-POST request
- `500 Internal Server Error` — token mint failure

---

### GET /api/v1/k8s/status

Reports Kubernetes Service Account provisioner configuration.

**Authentication:** Required — JWT or API key. Permission: `enrollment:status`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Response: 200 OK</strong></summary>

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

**Error codes:**

- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-GET request

---

### GET /api/v1/acme/status

Reports ACME directory URL, challenges, and EAB requirements.

**Authentication:** Required — JWT or API key. Permission: `enrollment:status`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "enabled": true,
  "directory_url": "https://ca.example.com/acme/directory",
  "provisioner": "acme",
  "challenges": [
    "http-01",
    "dns-01",
    "tls-alpn-01"
  ],
  "dns_name": "ca.example.com",
  "require_eab": false,
  "device_attest_enabled": false,
  "attestation_formats": []
}
```

</details>

**Error codes:**

- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-GET request

---

### POST /api/v1/acme/eab-keys

Creates an ACME External Account Binding (EAB) credential pair.

**Authentication:** Required — JWT or API key. Permission: `acme:eab`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Request Body (JSON)</strong></summary>

```json
{
  "provisioner": "acme",
  "reference": "customer-42"
}
```

</details>

<details>
<summary><strong>Response: 201 Created</strong></summary>

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

**Error codes:**

- `400 Bad Request` — invalid provisioner or reference
- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-POST request
- `500 Internal Server Error` — EAB creation failure

---

### GET /api/v1/scep/status

Reports SCEP endpoint availability (when SCEP provisioner is configured).

**Authentication:** Required — JWT or API key. Permission: `enrollment:status`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "enabled": true,
  "base_url": "http://localhost:8080/scep/scep",
  "provisioner": "scep",
  "challenge_hint": "configured"
}
```

</details>

**Error codes:**

- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-GET request

---

### GET /api/v1/ndes/status

Reports NDES connector endpoints for AD CS migration paths.

**Authentication:** Required — JWT or API key. Permission: `enrollment:status`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "enabled": true,
  "scep_endpoint": "http://localhost:8080/certsrv/mscep/mscep.dll",
  "admin_endpoint": "http://localhost:8080/certsrv/mscep_admin/mscep_admin.dll",
  "connectors": [
    "default"
  ],
  "adcs_compatible": true
}
```

</details>

**Error codes:**

- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-GET request

---

## Certificate Templates

### POST /api/v1/templates

Creates a certificate issuance template (Go `text/template` body rendering JSON SAN/extension data).

**Authentication:** Required — JWT or API key. Permission: `templates:manage`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Request Body (JSON)</strong></summary>

```json
{
  "name": "web-server",
  "description": "Standard TLS web server template",
  "body": "{\"dnsNames\": [\"{{ .CommonName }}\"], \"keyUsage\": [\"digitalSignature\", \"keyEncipherment\"]}"
}
```

</details>

<details>
<summary><strong>Response: 201 Created</strong></summary>

```json
{
  "id": "tpl_01HXYZ",
  "name": "web-server",
  "description": "Standard TLS web server template",
  "body": "{\"dnsNames\": [\"{{ .CommonName }}\"], \"keyUsage\": [\"digitalSignature\", \"keyEncipherment\"]}",
  "created_at": "2026-06-02T10:00:00Z",
  "updated_at": "2026-06-02T10:00:00Z"
}
```

</details>

**Error codes:**

- `400 Bad Request` — validation errors, duplicate name, invalid template body
- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-POST request
- `500 Internal Server Error` — persistence failure

---

### GET /api/v1/templates

Lists all certificate templates.

**Authentication:** Required — JWT or API key. Permission: `templates:read`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Response: 200 OK</strong></summary>

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

**Error codes:**

- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-GET request
- `500 Internal Server Error` — list failure

---

## SSH Certificate Authority

### POST /api/v1/ssh/sign-user

Issues a short-lived SSH **user** certificate. Authenticated callers without a body `token` receive an internal minted JWK token automatically.

**Authentication:** Optional — JWT or API key **unless** `token` is set in the body (OIDC/provisioner token path).

**Parameters / Query Strings:** None

<details>
<summary><strong>Request Body (JSON)</strong></summary>

```json
{
  "public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ...",
  "principal": "alice",
  "principals": [
    "alice",
    "admin"
  ],
  "ttl": "8h",
  "token": "",
  "provisioner": "ssh-oidc"
}
```

</details>

<details>
<summary><strong>Response: 201 Created</strong></summary>

```json
{
  "certificate": "ssh-rsa-cert-v01@openssh.com AAAAB3NzaC1yc2EAAAADAQABAAABAQ...",
  "certificate_type": "user",
  "key_id": "alice",
  "principals": [
    "alice",
    "admin"
  ],
  "serial": 42,
  "valid_after": "2026-06-02T10:00:00",
  "valid_before": "2026-06-02T18:00:00"
}
```

</details>

**Error codes:**

- `400 Bad Request` — missing `public_key` or principals
- `401 Unauthorized` — no credentials and no body `token`
- `405 Method Not Allowed` — non-POST request
- `500 Internal Server Error` — signing failure

---

### POST /api/v1/ssh/sign-host

Issues an SSH **host** certificate.

**Authentication:** Required — JWT or API key. Permission: `ssh:sign_host`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Request Body (JSON)</strong></summary>

```json
{
  "public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...",
  "hostname": "web-01.example.com",
  "principals": [
    "web-01.example.com",
    "localhost"
  ],
  "ttl": "8760h",
  "provisioner": "ssh-host"
}
```

</details>

<details>
<summary><strong>Response: 201 Created</strong></summary>

```json
{
  "certificate": "ssh-ed25519-cert-v01@openssh.com AAAAC3NzaC1lZDI1NTE5AAAAI...",
  "certificate_type": "host",
  "key_id": "web-01.example.com",
  "principals": [
    "web-01.example.com",
    "localhost"
  ],
  "serial": 7,
  "valid_after": "2026-06-02T10:00:00",
  "valid_before": "2027-06-02T10:00:00"
}
```

</details>

**Error codes:**

- `400 Bad Request` — missing `public_key` or hostname/principals
- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-POST request
- `500 Internal Server Error` — signing failure

---

### POST /api/v1/ssh/inspect

Decodes an SSH certificate and returns metadata.

**Authentication:** Required — JWT or API key. Permission: `ssh:inspect`.

**Parameters / Query Strings:** None

<details>
<summary><strong>Request Body (JSON)</strong></summary>

```json
{
  "certificate": "ssh-rsa-cert-v01@openssh.com AAAAB3NzaC1yc2EAAAADAQABAAABAQ..."
}
```

</details>

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "certificate_type": "user",
  "key_id": "alice",
  "principals": [
    "alice"
  ],
  "serial": 42,
  "valid_after": "2026-06-02T10:00:00",
  "valid_before": "2026-06-02T18:00:00",
  "public_key_type": "ssh-rsa",
  "critical_options": {},
  "extensions": {
    "permit-pty": ""
  },
  "signature_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ..."
}
```

</details>

**Error codes:**

- `400 Bad Request` — missing or invalid certificate
- `401 Unauthorized` — authentication required
- `403 Forbidden` — insufficient permissions
- `405 Method Not Allowed` — non-POST request
- `500 Internal Server Error` — inspection failure

---

### GET /api/v1/ssh/roots

Returns SSH CA public keys for `known_hosts` / `authorized_keys` trust configuration.

**Authentication:** No

**Parameters / Query Strings:** None

<details>
<summary><strong>Response: 200 OK</strong></summary>

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

**Error codes:**

- `404 Not Found` — SSH CA not configured
- `405 Method Not Allowed` — non-GET request
- `500 Internal Server Error` — retrieval failure

---

## OCSP Responder

RFC 6960 endpoints. Responses are **DER** `application/ocsp-response`, not the JSON envelope.

### POST /ocsp

Submits an OCSP request body.

**Authentication:** No

**Parameters / Query Strings:** None

**Request body:** `application/ocsp-request` (DER-encoded OCSP request)

**Response (success):** `200 OK`, `Content-Type: application/ocsp-response`, DER-encoded OCSP response

**Error codes:**

- `400 Bad Request` — unreadable request or invalid OCSP request
- `405 Method Not Allowed` — non-POST request

---

### GET /ocsp/{request}

Submits an OCSP request as a URL-safe Base64 (or standard Base64) path segment.

**Authentication:** No

**Parameters / Query Strings:**

| Name | Location | Description |
| ---- | -------- | ----------- |
| `request` | path | Base64url, Base64url with padding, or standard Base64 OCSP request DER |

**Response (success):** Same as `POST /ocsp`

**Error codes:**

- `400 Bad Request` — empty path, decode failure, invalid OCSP request
- `405 Method Not Allowed` — non-GET request

---

## ACMEv2 (RFC 8555)

Mounted at `/acme/` when ACME is enabled (`CA_API_ACME_DISABLED` is not `true` and an ACME provisioner exists). Public URLs use a **flat** layout (no provisioner name in paths). Requests and responses use **JWS** (`Content-Type: application/jose+json`) per RFC 8555, not the arx JSON envelope.

**Authentication:** ACME account key JWS on mutating endpoints; `GET` directory and certificate download use no admin JWT.

**Provisioner (internal name):** `acme`

### GET /acme/directory

Returns the ACME directory object listing all resource URLs.

**Authentication:** No

**Parameters / Query Strings:** None

<details>
<summary><strong>Response: 200 OK (application/json)</strong></summary>

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
    "caaIdentities": [
      "ca.example.com"
    ],
    "externalAccountRequired": false
  }
}
```

</details>

**Error codes:**

- `404 Not Found` — ACME disabled
- `500 Internal Server Error` — provisioner load failure

---

### HEAD /acme/new-nonce

### GET /acme/new-nonce

Issues a fresh anti-replay nonce.

**Authentication:** No

**Response (success):** `204 No Content` with header `Replay-Nonce: <nonce>`

**Error codes:**

- `404 Not Found` — ACME disabled

---

### POST /acme/new-account

Creates or updates an ACME account. Use JWS with an empty payload to look up an existing account (RFC 8555).

**Authentication:** JWS signed with account key (optional EAB when `externalAccountRequired` is true)

<details>
<summary><strong>Request (JWS payload, decoded JSON)</strong></summary>

```json
{
  "termsOfServiceAgreed": true,
  "contact": [
    "mailto:admin@example.com"
  ],
  "externalAccountBinding": {
    "protected": "eyJhbGciOiJIUzI1NiIs...",
    "payload": "eyJhbGciOiJIUzI1NiIs...",
    "signature": "abc..."
  }
}
```

</details>

<details>
<summary><strong>Response: 201 Created or 200 OK</strong></summary>

```json
{
  "status": "valid",
  "contact": [
    "mailto:admin@example.com"
  ],
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

**Error codes:**

- `400 Bad Request` — malformed JWS or payload
- `401 Unauthorized` — invalid EAB or account key
- `403 Forbidden` — policy violation
- `404 Not Found` — ACME disabled
- `409 Conflict` — account URL mismatch
- `500 Internal Server Error` — server error

---

### POST /acme/new-order

Creates a certificate order for one or more identifiers.

**Authentication:** JWS (account key, `kid` in protected header after registration)

<details>
<summary><strong>Request (JWS payload, decoded JSON)</strong></summary>

```json
{
  "identifiers": [
    {
      "type": "dns",
      "value": "www.example.com"
    }
  ],
  "notBefore": "2026-06-02T00:00:00Z",
  "notAfter": "2026-09-02T00:00:00Z"
}
```

</details>

<details>
<summary><strong>Response: 201 Created</strong></summary>

```json
{
  "status": "pending",
  "expires": "2026-06-03T00:00:00Z",
  "identifiers": [
    {
      "type": "dns",
      "value": "www.example.com"
    }
  ],
  "authorizations": [
    "https://ca.example.com/acme/authz/authz-01"
  ],
  "finalize": "https://ca.example.com/acme/order/ord-01/finalize",
  "certificate": null
}
```

</details>

**Error codes:**

- `400 Bad Request` — invalid identifiers
- `401 Unauthorized` — invalid JWS
- `403 Forbidden` — unauthorized identifier
- `404 Not Found` — ACME disabled
- `500 Internal Server Error` — order creation failure

---

### POST /acme/order/{orderID}/finalize

Submits a CSR to finalize a ready order.

**Authentication:** JWS (account key)

**Parameters / Query Strings:**

| Name | Location | Description |
| ---- | -------- | ----------- |
| `orderID` | path | Order identifier |

<details>
<summary><strong>Request (JWS payload, decoded JSON)</strong></summary>

```json
{
  "csr": "MIIB...base64url-encoded-CSR-DER..."
}
```

</details>

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "status": "valid",
  "expires": "2026-06-03T00:00:00Z",
  "identifiers": [
    {
      "type": "dns",
      "value": "www.example.com"
    }
  ],
  "authorizations": [
    "https://ca.example.com/acme/authz/authz-01"
  ],
  "finalize": "https://ca.example.com/acme/order/ord-01/finalize",
  "certificate": "https://ca.example.com/acme/certificate/cert-01"
}
```

</details>

**Error codes:**

- `400 Bad Request` — invalid CSR or order state
- `401 Unauthorized` — invalid JWS
- `403 Forbidden` — finalize not allowed
- `404 Not Found` — unknown order
- `500 Internal Server Error` — signing failure

---

### GET /acme/order/{orderID}

Polls order status.

**Authentication:** JWS (account key)

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "status": "valid",
  "expires": "2026-06-03T00:00:00Z",
  "identifiers": [
    {
      "type": "dns",
      "value": "www.example.com"
    }
  ],
  "authorizations": [
    "https://ca.example.com/acme/authz/authz-01"
  ],
  "finalize": "https://ca.example.com/acme/order/ord-01/finalize",
  "certificate": "https://ca.example.com/acme/certificate/cert-01"
}
```

</details>

**Error codes:**

- `401 Unauthorized` — invalid JWS
- `404 Not Found` — unknown order
- `500 Internal Server Error` — retrieval failure

---

### GET /acme/authz/{authzID}

Returns authorization status and associated challenges.

**Authentication:** JWS (account key)

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "identifier": {
    "type": "dns",
    "value": "www.example.com"
  },
  "status": "pending",
  "expires": "2026-06-03T00:00:00Z",
  "challenges": [
    {
      "type": "http-01",
      "url": "https://ca.example.com/acme/challenge/authz-01/ch-01",
      "status": "pending",
      "token": "token-abc123"
    },
    {
      "type": "dns-01",
      "url": "https://ca.example.com/acme/challenge/authz-01/ch-02",
      "status": "pending",
      "token": "token-abc123"
    },
    {
      "type": "tls-alpn-01",
      "url": "https://ca.example.com/acme/challenge/authz-01/ch-03",
      "status": "pending",
      "token": "token-abc123"
    }
  ],
  "wildcard": false
}
```

</details>

**Error codes:**

- `401 Unauthorized` — invalid JWS
- `404 Not Found` — unknown authorization
- `500 Internal Server Error` — retrieval failure

---

### POST /acme/challenge/{authzID}/{challengeID}

Triggers challenge validation. arx performs outbound **http-01**, **dns-01**, or **tls-alpn-01** checks.

**Authentication:** JWS (account key)

**Parameters / Query Strings:**

| Name | Location | Description |
| ---- | -------- | ----------- |
| `authzID` | path | Authorization ID |
| `challengeID` | path | Challenge ID |

<details>
<summary><strong>Request (JWS payload)</strong></summary>

Empty JSON object `{}` (payload may be empty string in JWS).

</details>

<details>
<summary><strong>Response: 200 OK</strong></summary>

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

**Error codes:**

- `400 Bad Request` — malformed JWS
- `401 Unauthorized` — invalid JWS
- `403 Forbidden` — challenge not acceptable
- `404 Not Found` — unknown challenge
- `500 Internal Server Error` — validation error

---

### GET /acme/certificate/{certID}

Downloads the issued certificate chain (DER), typically `application/pem-certificate-chain` or `application/pkix-cert`.

**Authentication:** JWS (account key) for POST-as-GET per RFC 8555; implementation may accept `Accept` negotiation.

**Response (success):** PEM certificate chain or DER per client `Accept` header

**Error codes:**

- `401 Unauthorized` — invalid JWS
- `404 Not Found` — unknown certificate
- `500 Internal Server Error` — download failure

---

### POST /acme/revoke-cert

Revokes a certificate.

**Authentication:** JWS with either account key (`kid`) or certificate key (`jwk` in protected header)

<details>
<summary><strong>Request (JWS payload, decoded JSON)</strong></summary>

```json
{
  "certificate": "MIIB...base64url-encoded-cert-DER...",
  "reason": 1
}
```

</details>

**Response (success):** `200 OK` with empty body

**Error codes:**

- `400 Bad Request` — malformed request
- `401 Unauthorized` — invalid JWS
- `403 Forbidden` — revocation not permitted
- `404 Not Found` — ACME disabled
- `500 Internal Server Error` — revocation failure

---

### POST /acme/key-change

Rotates the ACME account key (RFC 8555 key rollover).

**Authentication:** JWS signed by both old and new keys per RFC 8555

<details>
<summary><strong>Request (JWS payload, decoded JSON)</strong></summary>

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

**Response (success):** `200 OK` with updated account body

**Error codes:**

- `400 Bad Request` — invalid rollover proof
- `401 Unauthorized` — invalid JWS
- `403 Forbidden` — rollover denied
- `404 Not Found` — ACME disabled
- `500 Internal Server Error` — update failure

---

### POST /acme/account/{accountID}

Updates account contact information or deactivates the account.

**Authentication:** JWS (account key)

<details>
<summary><strong>Request (JWS payload, decoded JSON)</strong></summary>

```json
{
  "contact": [
    "mailto:ops@example.com"
  ],
  "status": "deactivated"
}
```

</details>

<details>
<summary><strong>Response: 200 OK</strong></summary>

```json
{
  "status": "deactivated",
  "contact": [
    "mailto:ops@example.com"
  ],
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

**Error codes:**

- `400 Bad Request` — invalid payload
- `401 Unauthorized` — invalid JWS
- `404 Not Found` — unknown account
- `500 Internal Server Error` — update failure

---

ACME errors use `application/problem+json` problem documents (`type`, `detail`, `status`) per RFC 8555, not the arx `error`/`data` envelope.

---

## SCEP (optional)

Registered at `/scep/` when a SCEP provisioner exists in `ca.json`. Uses the **smallstep SCEP** protocol (PKCS#7/CMS), not JSON.

**Base path:** `/scep/{provisioner}` (default provisioner name: `scep`)

Typical operations (HTTP methods vary by operation; see [smallstep SCEP API](https://github.com/smallstep/certificates)):

| Operation | Description |
| --------- | ----------- |
| GetCACaps | Query CA capabilities |
| GetCACert | Retrieve CA certificate(s) |
| PKIOperation | Enrollment and renewal messages |

**Authentication:** SCEP challenge password when `CA_API_SCEP_CHALLENGE` is configured.

**Status discovery (JSON API):** `GET /api/v1/scep/status`

**Error codes:** Protocol-specific HTTP statuses; `404` when SCEP is not enabled.

---

## NDES (optional)

Registered at `/certsrv/` for Microsoft AD CS–compatible paths when NDES connectors are configured.

### GET /certsrv/mscep/mscep.dll

Proxies to the configured SCEP connector (same protocol as `/scep/scep`).

**Authentication:** SCEP challenge password

**Response:** SCEP PKCS#7 payloads (not JSON)

**Error codes:**

- `503 Service Unavailable` — connector not configured
- `404 Not Found` — NDES disabled

---

### GET /certsrv/mscep_admin/mscep_admin.dll

Returns the SCEP challenge password for NDES enrollment workflows.

**Authentication:** Optional `X-NDES-Admin-Secret` header or `?secret=` query parameter when `NDES` admin secret is configured

**Response (success):** `200 OK`, `text/plain` challenge password body

**Error codes:**

- `401 Unauthorized` — invalid admin secret
- `405 Method Not Allowed` — non-GET request
- `503 Service Unavailable` — `CA_API_SCEP_CHALLENGE` not set

**Status discovery (JSON API):** `GET /api/v1/ndes/status`

---

## Related Documentation

- [acme.md](acme.md) — ACME challenge behavior, environment variables, reverse-proxy examples
- [cli_reference.md](cli_reference.md) — `arx` CLI and agent commands
- [architecture.md](architecture.md) — system layout and databases
