# ACME (RFC 8555)

**arx** exposes an ACMEv2 endpoint compatible with standard clients and reverse proxies. The implementation builds on [smallstep/certificates](https://github.com/smallstep/certificates) for JWS, nonce, and order flow, with **arx-specific** flat URL layout, SQLite-backed ACME state (by default), and **multi-challenge validation** in `internal/acmeprotocol`.

## Enabling ACME

ACME is available when all of the following hold:

1. The step-ca configuration (`.pki/config/ca.json`) includes an **ACME provisioner** (default name: `acme`).
2. The application database is open (SQLite or PostgreSQL).
3. `CA_API_ACME_DISABLED` is not set to `true`.

On startup, when active, the server logs:

```text
ACME enabled; directory available at /acme/directory
```

Disable explicitly:

```bash
export CA_API_ACME_DISABLED=true
```

## Directory endpoint

| Item | Value |
| ---- | ----- |
| **URL path** | `/acme/directory` |
| **Mount** | Handlers registered at `/acme/`; public paths omit the internal provisioner segment |
| **Provisioner (internal)** | `acme` — used inside step-ca routing but omitted from public URLs |

### Discovering the directory

```bash
curl -s https://ca.example.com/acme/directory | jq .
```

Local development (port 8080):

```bash
curl -s http://localhost:8080/acme/directory | jq .
```

The directory JSON follows RFC 8555 (`newNonce`, `newAccount`, `newOrder`, `revokeCert`, `keyChange`, `meta`, etc.). Resource URLs use `/acme/account/...`, `/acme/order/...`, `/acme/authz/...`, and `/acme/challenge/...`.

## URL layout

Public clients see a **flat** tree (no `/acme/acme/...` provisioner prefix):

| Resource | Path pattern |
| -------- | ------------ |
| Directory | `/acme/directory` |
| New nonce | `/acme/new-nonce` |
| New account | `/acme/new-account` |
| New order | `/acme/new-order` |
| Account | `/acme/account/{id}` |
| Order | `/acme/order/{id}` |
| Authorization | `/acme/authz/{id}` |
| Challenge | `/acme/challenge/{authzID}/{challengeID}` |
| Certificate | `/acme/certificate/{id}` |

`internal/acmeprotocol/linker.go` (`FlatLinker`) generates these links. Incoming requests are adapted to step-ca’s internal provisioner-scoped paths in `pathAdapter` (`router.go`).

## Multi-challenge support

arx validates **all three** mainstream domain-validation challenge types before marking an authorization valid. Clients choose the challenge type offered in the authorization object; arx performs **outbound** proof checks from the CA process when the client POSTs to the challenge URL.

| Type | RFC | Validator | Proof |
| ---- | --- | --------- | ----- |
| **http-01** | RFC 8555 §8.3 | `VerifyHTTP01` (`http01.go`) | HTTP GET `http://<host>/.well-known/acme-challenge/<token>`; body equals key authorization |
| **dns-01** | RFC 8555 §8.4 | `VerifyDNS01` (`dns01.go`) | TXT at `_acme-challenge.<domain>` equals `DNS01Digest(keyAuthorization)` |
| **tls-alpn-01** | RFC 8737 | `VerifyTLSALPN01` (`tlsalpn01.go`) | TLS to identifier port with ALPN `acme-tls/1`; leaf cert has critical `id-pe-acmeIdentifier` with SHA-256(keyAuthorization) |

Orchestration: `internal/acmeprotocol/validate.go` transitions challenges `pending` → `processing` → `valid` | `invalid`. The HTTP handler `GetChallenge` (`challenge_handler.go`) invokes `Validator.ValidateChallenge` on each client-triggered validation.

### Challenge selection guide

| Challenge | Best for | Requirements |
| --------- | -------- | -------------- |
| **HTTP-01** | Single servers, reverse proxies, Traefik/Caddy HTTP routers | Port **80** reachable from the CA for the identifier (or `CA_API_ACME_HTTP_PORT` in lab setups); token at `/.well-known/acme-challenge/<token>` |
| **DNS-01** | Wildcards, internal names, multi-host, CDN frontends | `_acme-challenge.<name>` TXT record; no inbound HTTP to the origin required |
| **TLS-ALPN-01** | TLS on 443 without HTTP-01 path | Port **443** (or `CA_API_ACME_TLS_PORT`) serves temporary ALPN certificate during validation |

**Wildcards:** RFC 8555 requires **DNS-01** for wildcard identifiers; HTTP-01 and TLS-ALPN-01 apply to non-wildcard hostnames and IP identifiers as usual.

### HTTP-01 details

- Request URL: `http://<identifier>/.well-known/acme-challenge/<token>` (IPv6 addresses bracketed in URLs).
- Development: set `CA_API_ACME_HTTP_PORT` when the CA must connect to a non-80 port (for example API on 8080 behind a challenge proxy).

### DNS-01 details

- TXT name: `_acme-challenge.<apex>` (wildcard auth strips `*.` for the label).
- Expected value: `DNS01Digest(keyAuthorization)` from `internal/acmeprotocol/keyauth.go`.

### TLS-ALPN-01 details

- Dial `tcp` to identifier port 443 (default) or `CA_API_ACME_TLS_PORT`.
- ALPN protocol must be `acme-tls/1`.
- Leaf certificate must match the identifier (single DNS name or single IP) and include the critical ACME identifier extension.

## Challenge validation flow

```mermaid
sequenceDiagram
    participant Client as ACME Client
    participant CA as arx /acme
    participant Val as acmeprotocol.Validator
    participant Id as Identifier (HTTP/DNS/TLS)

    Client->>CA: POST challenge (JWS)
    CA->>Val: ValidateChallenge
    Val->>Id: HTTP GET / DNS TXT / TLS handshake
    Id-->>Val: proof
    Val->>CA: update status valid/invalid
    CA-->>Client: challenge JSON
```

Clients trigger validation by posting to the challenge URL (RFC 8555). arx does not use in-band hook callbacks from the applicant; the CA initiates outbound checks.

## Persistence

ACME accounts, orders, authorizations, challenges, and nonces are stored in the **application database** (SQLite `arx.db` by default), not in Badger. Schema is created in `internal/database/migrate.go` (`acme_*` tables). `internal/database/acme_store.go` implements the step-ca `acme.DB` interface.

## Environment variables

| Variable | Effect |
| -------- | ------ |
| `CA_API_ACME_DISABLED` | `true` disables ACME handler registration |
| `CA_API_ACME_DNS` | Hostname used in directory links when not inferable from request/`ca.json` |
| `CA_API_ACME_HTTP_PORT` | Port for outbound HTTP-01 checks (default 80) |
| `CA_API_ACME_TLS_PORT` | Port for outbound TLS-ALPN-01 checks (default 443) |
| `CA_API_ACME_STRICT_FQDN` | `true` enables strict FQDN checks (step-ca behavior) |
| `CA_API_ACME_REQUIRE_EAB` | `true` requires External Account Binding for new accounts |
| `CA_API_ACME_DEVICE_ATTEST` | `true` enables device-attest provisioner options |
| `CA_API_ACME_ATTESTATION_FORMATS` | Comma-separated attestation formats |
| `CA_API_ACME_ATTESTATION_ROOTS` | Path to attestation trust roots |

Admin API (JWT required):

| Method | Path | Description |
| ------ | ---- | ----------- |
| `GET` | `/api/v1/acme/status` | ACME enrollment status |
| `POST` | `/api/v1/acme/eab-keys` | Create EAB keys (when EAB is used) |

## Reverse-proxy integration

### Traefik

```yaml
certificatesResolvers:
  arx:
    acme:
      email: admin@example.com
      storage: /path/to/acme.json
      caServer: https://ca.example.com/acme/directory
      httpChallenge:
        entryPoint: web
```

For DNS-01, configure Traefik’s DNS provider; arx validates the TXT record at `_acme-challenge.<domain>`.

Ensure challenge traffic is reachable from the CA host:

- **HTTP-01:** Port 80 on the identifier serves the challenge token (or align `CA_API_ACME_HTTP_PORT` in dev).
- **TLS-ALPN-01:** Port 443 terminates TLS with ALPN support on the target.

### Caddy

Global CA:

```caddyfile
{
  acme_ca https://ca.example.com/acme/directory
}
```

Per-site:

```caddyfile
example.com {
  tls {
    issuer acme {
      dir https://ca.example.com/acme/directory
    }
  }
}
```

Caddy selects challenge types automatically; use a DNS challenge plugin for wildcards (DNS-01).

### Certbot

```bash
certbot certonly \
  --server https://ca.example.com/acme/directory \
  -d www.example.com \
  --standalone
```

For DNS-01, use a Certbot DNS plugin and ensure TXT records match arx validation timing.

### Private CA / TLS trust

Clients must trust your arx **root** (and often intermediate) CA:

```bash
curl -s https://ca.example.com/api/v1/ca/root
curl -s https://ca.example.com/api/v1/public/ca/intermediate
```

Or install locally:

```bash
arx agent trust install-root --url https://ca.example.com
arx agent trust install-intermediate --url https://ca.example.com
```

## Operational notes

1. **Reachability** — The CA validates challenges by connecting **to** the identifier from the server process. NAT, firewalls, and wrong DNS are the most common failure modes.
2. **Directory hostname** — Set `CA_API_ACME_DNS` to the public hostname clients use (not an internal bind address).
3. **Rate limits** — Follow step-ca / arx logs for authorization and validation errors; failed challenges return `pending` or `invalid` per error type.
4. **EAB** — When `CA_API_ACME_REQUIRE_EAB=true`, create binding keys via `POST /api/v1/acme/eab-keys` before `newAccount`.

## See also

- [architecture.md](architecture.md) — ACME component placement and databases
- [cli_reference.md](cli_reference.md) — trust installation and admin tools
- [README.md](../README.md) — quick start and health checks
