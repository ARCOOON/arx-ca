# API Reference

## Conventions

| Item | Value |
| ---- | ----- |
| Prefix | `/api/v1` |
| Default listen | `http://localhost:8080` |
| Content-Type | `application/json; charset=utf-8` |
| Envelope | `{ "error": null, "data": {…} }` on success |

## Authentication

| Method | Header |
| ------ | ------ |
| Admin JWT | `Authorization: Bearer <jwt>` |
| Service account | `X-API-Key: <key>` or `Authorization: Bearer <key>` |
| Session cookie | `arx_session` (HttpOnly, set by login) |
| mTLS | Client cert on TLS listener or `X-Forwarded-Client-Cert` via WebUI proxy |

### Auth endpoints

| Method | Path | Auth | Description |
| ------ | ---- | ---- | ----------- |
| `POST` | `/api/v1/auth/login` | None | Admin login → JWT |
| `POST` | `/api/v1/auth/logout` | JWT | Clear session |
| `POST` | `/api/v1/auth/service-accounts` | SuperAdmin | Create API key |

## Health & CA info

| Method | Path | Auth | Description |
| ------ | ---- | ---- | ----------- |
| `GET` | `/api/v1/health` | None | Uptime, memory, CA status |
| `GET` | `/api/v1/ca/root` | None | Root CA PEM |
| `GET` | `/api/v1/ca/chain` | None | Full chain PEM |
| `GET` | `/api/v1/ca/info` | Read | CA metadata |
| `GET` | `/api/v1/ca/provisioners` | Read | Provisioner list |
| `GET` | `/api/v1/ca/crl` | None | Certificate Revocation List |
| `GET` | `/api/v1/crl` | None | CRL alias |

## Public / agent read-only

| Method | Path | Auth | Description |
| ------ | ---- | ---- | ----------- |
| `GET` | `/api/v1/public/ca/intermediate` | None | Intermediate CA PEM |
| `GET` | `/api/v1/public/certificates` | None | Public cert inventory |
| `GET` | `/api/v1/public/certificates/{serial}` | None | Single cert by serial |

## Certificate management

| Method | Path | Auth | Description |
| ------ | ---- | ---- | ----------- |
| `POST` | `/api/v1/certificates/issue` | Issue | Sign CSR |
| `POST` | `/api/v1/certificates/generate` | Issue | Generate key + cert |
| `POST` | `/api/v1/certificates/issue-with-token` | Issue | Provisioner token flow |
| `POST` | `/api/v1/certificates/auto` | Issue (admin) | Automated issuance |
| `POST` | `/api/v1/certificates/revoke` | Revoke | Revoke by serial |
| `POST` | `/api/v1/certificates/lint` | Lint | Lint PEM/CSR |
| `GET` | `/api/v1/certificates` | Read | List inventory |
| `GET` | `/api/v1/certificates/stats` | Read | Inventory statistics |
| `GET` | `/api/v1/certificates/{serial}` | Read | Get by serial |
| `GET` | `/api/v1/certificates/{serial}/key` | Read (admin) | Escrowed private key |
| `GET` | `/api/v1/certificates/{serial}/bundle` | Read (admin) | Download ZIP bundle |

## Renewal & rekey

| Method | Path | Auth | Description |
| ------ | ---- | ---- | ----------- |
| `POST` | `/api/v1/certificates/renew` | Renew / mTLS | Renew existing cert |
| `POST` | `/api/v1/certificates/rekey` | Renew / mTLS | Rekey existing cert |

## Provisioners & enrollment

| Method | Path | Auth | Description |
| ------ | ---- | ---- | ----------- |
| `POST` | `/api/v1/provisioners/token` | Token | Mint provisioner token |
| `GET` | `/api/v1/k8s/status` | Enrollment | Kubernetes provisioner status |
| `GET` | `/api/v1/acme/status` | Enrollment | ACME provisioner status |
| `POST` | `/api/v1/acme/eab-keys` | EAB | Create EAB key |
| `GET` | `/api/v1/scep/status` | Enrollment | SCEP status |
| `GET` | `/api/v1/ndes/status` | Enrollment | NDES status |

## Templates

| Method | Path | Auth | Description |
| ------ | ---- | ---- | ----------- |
| `POST` | `/api/v1/templates` | Manage | Create template |
| `GET` | `/api/v1/templates` | Read | List templates |

## SSH CA

| Method | Path | Auth | Description |
| ------ | ---- | ---- | ----------- |
| `POST` | `/api/v1/ssh/generate/user` | SSH sign user | Generate user cert |
| `POST` | `/api/v1/ssh/generate/host` | SSH sign host | Generate host cert |
| `POST` | `/api/v1/ssh/sign-user` | Token | Sign user key |
| `POST` | `/api/v1/ssh/sign-host` | SSH sign host | Sign host key |
| `POST` | `/api/v1/ssh/inspect` | SSH inspect | Parse SSH cert |
| `GET` | `/api/v1/ssh/roots` | None | CA public keys |
| `GET` | `/api/v1/ssh/certificates` | SSH inspect | Cert inventory |
| `GET` | `/api/v1/ssh/stats` | SSH inspect | Issuance stats |

## Audit & notifications

| Method | Path | Auth | Description |
| ------ | ---- | ---- | ----------- |
| `GET` | `/api/v1/audit` | Audit read | Forensic log query |
| `GET` | `/api/v1/notifications/stream` | Audit read | SSE event stream |
| `GET` | `/api/v1/notifications` | Audit read | Notification history |
| `POST` | `/api/v1/notifications/{id}/read` | Audit read | Mark read |
| `POST` | `/api/v1/notifications/read-all` | Audit read | Mark all read |
| `POST` | `/api/v1/notifications/archive-all` | Audit read | Archive all |
| `DELETE` | `/api/v1/notifications/{id}` | Audit read | Delete notification |

### Audit query parameters

| Param | Type | Description |
| ----- | ---- | ----------- |
| `limit` | int | Page size (default 50, max 500) |
| `offset` | int | Pagination offset |
| `action` | string | Filter by action |
| `actor` | string | Filter by actor ID/type |
| `ip` | string | Filter by IP (substring) |
| `status` | int | Filter by HTTP status code |

## Webhooks

| Method | Path | Auth | Description |
| ------ | ---- | ---- | ----------- |
| `GET` | `/api/v1/webhooks/events` | Webhooks manage | Subscribable events |
| `GET` | `/api/v1/webhooks` | Webhooks manage | List webhooks |
| `POST` | `/api/v1/webhooks` | Webhooks manage | Create webhook |
| `PUT` | `/api/v1/webhooks/{id}` | Webhooks manage | Update webhook |
| `DELETE` | `/api/v1/webhooks/{id}` | Webhooks manage | Delete webhook |
| `POST` | `/api/v1/webhooks/{id}/test` | Webhooks manage | Connectivity test |

## Protocol endpoints (non-JSON)

| Method | Path | Format | Description |
| ------ | ---- | ------ | ----------- |
| `POST` | `/ocsp` | OCSP | OCSP responder |
| `GET` | `/ocsp/{request}` | OCSP | OCSP GET |
| `*` | `/acme/*` | ACME JWS | RFC 8555 directory + orders |
| `*` | `/scep/*` | SCEP | Simple Certificate Enrollment |
| `*` | `/certsrv/*` | NDES | Network Device Enrollment |

- ACME directory: `GET /acme/directory`
- Mounted only when the corresponding provisioner is active in `ca.json`

## RBAC permission labels

| Label | Roles |
| ----- | ----- |
| Read | SuperAdmin, CA-Admin, Revocation-Manager, Read-Only |
| Issue | SuperAdmin, CA-Admin |
| Revoke | SuperAdmin, Revocation-Manager |
| Lint | SuperAdmin, CA-Admin, Revocation-Manager |
| Renew | SuperAdmin, CA-Admin (+ mTLS self-renewal) |
| Token | SuperAdmin, CA-Admin |
| Manage | SuperAdmin, CA-Admin |
| EAB | SuperAdmin, CA-Admin |
| Enrollment | SuperAdmin, CA-Admin, Revocation-Manager, Read-Only |
| SSH sign user/host | SuperAdmin, CA-Admin |
| SSH inspect | SuperAdmin, CA-Admin |
| Audit read | SuperAdmin, CA-Admin, Revocation-Manager, Read-Only |
| Webhooks manage | SuperAdmin, CA-Admin |

## Error codes

| Status | Meaning |
| ------ | ------- |
| `401` | Missing or invalid credentials |
| `403` | Valid credentials, insufficient RBAC |
| `404` | Resource not found |
| `405` | Method not allowed |
| `500` | Internal server error |
