# API-to-UI Parity Matrix

This document maps every HTTP endpoint registered by the arx CA server
(`internal/cmd/arx/server_start.go`) to its WebUI representation. It is the
authoritative parity checklist for Phase 91 and subsequent UI work.

**Legend**

| Status | Meaning |
| ------ | ------- |
| **IMPLEMENTED** | Full UI control with API wiring, error handling, and state |
| **INTERNAL** | Background or protocol route; no dedicated operator control required |
| **NEW/PENDING IMPLEMENTATION** | Gap identified in Phase 91 audit (resolved in same phase unless noted) |

---

## Health

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| GET | `/api/v1/health` | `Dashboard.vue` → stat cards; `Settings.vue` → Server section | IMPLEMENTED |

---

## CA (Root / Chain / Info / Provisioners)

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| GET | `/api/v1/ca/root` | `Settings.vue` → Download Root CA (.pem) button | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |
| GET | `/api/v1/ca/chain` | `Dashboard.vue` → Download CA Bundle (.zip) | IMPLEMENTED |
| GET | `/api/v1/ca/info` | `Dashboard.vue` → Certificate Authorities section | IMPLEMENTED |
| GET | `/api/v1/ca/provisioners` | `Dashboard.vue` → Active Provisioners grid | IMPLEMENTED |

---

## Revocation (CRL)

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| GET | `/api/v1/ca/crl` | `Certificates.vue` + `Dashboard.vue` → CRL status badge + download buttons (alias) | IMPLEMENTED |
| GET | `/api/v1/crl` | `Certificates.vue` + `Dashboard.vue` → Download CRL (PEM/DER) | IMPLEMENTED |

---

## Public (Unauthenticated)

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| GET | `/api/v1/public/ca/intermediate` | `Settings.vue` → Download Intermediate CA (.pem) | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |
| GET | `/api/v1/public/certificates` | `Settings.vue` → Public API reference + copy link | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |
| GET | `/api/v1/public/certificates/{serial}` | `Settings.vue` → Public API reference (documented URL pattern) | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |

---

## Authentication & Service Accounts

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| POST | `/api/v1/auth/login` | `Login.vue` → Sign in form | IMPLEMENTED |
| POST | `/api/v1/auth/service-accounts` | `Settings.vue` → Create Service Account modal + one-time API key alert | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |

---

## Certificates (Lifecycle)

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| POST | `/api/v1/certificates/issue` | `Certificates.vue` → Issue modal → Paste CSR tab | IMPLEMENTED |
| POST | `/api/v1/certificates/generate` | `Certificates.vue` → Issue modal → Native Generation tab (ZIP) | IMPLEMENTED |
| POST | `/api/v1/certificates/issue-with-token` | `Certificates.vue` → Issue modal → Provisioner Token tab | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |
| POST | `/api/v1/certificates/auto` | `Certificates.vue` → Issue modal → Auto Issue tab (SuperAdmin) | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |
| POST | `/api/v1/certificates/revoke` | `Certificates.vue` → red Revoke button (table + details modal) with serial-prefix / `REVOKE` confirmation | IMPLEMENTED |
| POST | `/api/v1/certificates/lint` | `Certificates.vue` → Details modal → Lint Certificate button | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |
| GET | `/api/v1/certificates` | `Dashboard.vue` count; `Certificates.vue` → DataTable | IMPLEMENTED |
| GET | `/api/v1/certificates/{serial}` | `Certificates.vue` → View Details modal | IMPLEMENTED |
| GET | `/api/v1/certificates/{serial}/key` | `Certificates.vue` → Details modal → Reveal / Download Key | IMPLEMENTED |
| GET | `/api/v1/certificates/{serial}/bundle` | `Certificates.vue` → Details modal → Download Bundle | IMPLEMENTED |
| POST | `/api/v1/certificates/renew` | `Certificates.vue` → Details modal → Renew Certificate button | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |
| POST | `/api/v1/certificates/rekey` | `Certificates.vue` → Details modal → Rekey modal (CSR input) | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |

---

## Provisioners & Kubernetes

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| POST | `/api/v1/provisioners/token` | `Provisioners.vue` → Mint Provisioner Token form + one-time token alert | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |
| GET | `/api/v1/k8s/status` | `Provisioners.vue` → Kubernetes provisioner status section | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |

---

## ACME / SCEP / NDES (Admin Status & EAB)

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| GET | `/api/v1/acme/status` | `Acme.vue` → status badges, endpoints, policy toggles | IMPLEMENTED |
| POST | `/api/v1/acme/eab-keys` | `Acme.vue` → Generate EAB Key form + one-time HMAC alert | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |
| GET | `/api/v1/scep/status` | `Scep.vue` → status badges, endpoints, challenge hint | IMPLEMENTED |
| GET | `/api/v1/ndes/status` | `Ndes.vue` → NDES connector status (SideNav entry) | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |

---

## Certificate Templates

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| POST | `/api/v1/templates` | `Templates.vue` → Create Template modal | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |
| GET | `/api/v1/templates` | `Templates.vue` → template list DataTable | NEW/PENDING IMPLEMENTATION → IMPLEMENTED |

---

## SSH CA

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| POST | `/api/v1/ssh/generate/user` | `Ssh.vue` → User Certificates tab | IMPLEMENTED |
| POST | `/api/v1/ssh/generate/host` | `Ssh.vue` → Host Certificates tab | IMPLEMENTED |
| POST | `/api/v1/ssh/sign-user` | Legacy API — OIDC/provisioner token flow | IMPLEMENTED |
| POST | `/api/v1/ssh/sign-host` | Legacy API — host signing | IMPLEMENTED |
| POST | `/api/v1/ssh/inspect` | Not exposed in WebUI (API-only) | IMPLEMENTED |
| GET | `/api/v1/ssh/roots` | `Ssh.vue` → SSH CA roots list + download buttons | IMPLEMENTED |

---

## OCSP (Non-`/api/v1`)

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| POST | `/ocsp` | INTERNAL — consumed by relying parties; documented in `Settings.vue` Public endpoints | INTERNAL |
| GET | `/ocsp/{request}` | INTERNAL — same as above | INTERNAL |

---

## ACME Protocol (RFC 8555)

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| * | `/acme/*` | `Acme.vue` → Directory URL display; clients use protocol directly | INTERNAL (discovery only) |

---

## SCEP Protocol

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| GET/POST | `/scep/{provisioner}` | `Scep.vue` → Base URL display; MDM clients enroll directly | INTERNAL (discovery only) |

---

## NDES / AD CS Compatible

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| * | `/certsrv/.../mscep/mscep.dll` | `Ndes.vue` → SCEP endpoint reference | INTERNAL (discovery only) |
| GET | `/certsrv/.../mscep_admin/mscep_admin.dll` | `Ndes.vue` → Admin endpoint reference | INTERNAL (discovery only) |

---

## Forensic Audit Log

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| GET | `/api/v1/audit` | `Audit.vue` → `DataTable` with inline `#row-expanded` detail rows; `Pagination.vue` numbered pages (`total`, `limit`, `offset` from API) | IMPLEMENTED |

State-changing handlers populate `action`, `provisioner`, and `fingerprint` via injected `AuditContext` before the audit middleware persists each row.

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| GET | `/api/v1/notifications/stream` | `AppShell.vue` + `NotificationToaster.vue` → SSE `EventSource` with priority toasts | IMPLEMENTED |
| GET | `/api/v1/notifications` | `TopBar.vue` + `NotificationDrawer.vue` → paginated history on drawer open | IMPLEMENTED |
| POST | `/api/v1/notifications/{id}/read` | `NotificationDrawer.vue` → per-item Mark as read | IMPLEMENTED |
| POST | `/api/v1/notifications/read-all` | `NotificationDrawer.vue` → Mark all as read | IMPLEMENTED |
| DELETE | `/api/v1/notifications/{id}` | `NotificationDrawer.vue` → trash icon delete | IMPLEMENTED |
| POST | `/api/v1/notifications/archive-all` | `NotificationDrawer.vue` → header trash icon clear all (soft-delete) | IMPLEMENTED |

**Client-only UI preferences (localStorage, no API):**

| Key | Values | View / Component | Behavior |
| --- | ------ | ---------------- | -------- |
| `arx_ui_notification_style` | `drawer` \| `overlay` | `Settings.vue` → UI Preferences; `NotificationDrawer.vue` | Drawer: full-height right slide-out with solid opaque panel and dimmed backdrop (`bg-black/40`). Overlay: floating card below the top bar with uniform 1rem inset from the right edge and below the chrome bar (`top-20`, `right-4`, `max-h-[70vh]`, `rounded-[var(--radius-surface)]`); no dimmed backdrop (transparent click-catcher only). Both use shadow/border depth instead of backdrop blur. Updates globally in real time via `useNotificationLayout.ts`. |
| `arx_sidebar_collapsed` | `true` \| `false` | `Settings.vue` → UI Preferences; `AppShell.vue` | Default sidebar collapsed state; requires page reload after save. |

---

## Webhook Notifications

| Method | Path | UI Mapping | Status |
| ------ | ---- | ---------- | ------ |
| GET | `/api/v1/webhooks` | `Webhooks.vue` → DataTable list | IMPLEMENTED |
| GET | `/api/v1/webhooks/events` | `Webhooks.vue` → subscribed event multi-select | IMPLEMENTED |
| POST | `/api/v1/webhooks` | `Webhooks.vue` → Add Webhook modal | IMPLEMENTED |
| PUT | `/api/v1/webhooks/{id}` | `Webhooks.vue` → Edit Webhook modal | IMPLEMENTED |
| DELETE | `/api/v1/webhooks/{id}` | `Webhooks.vue` → delete confirmation | IMPLEMENTED |
| POST | `/api/v1/webhooks/{id}/test` | `Webhooks.vue` → Test Connection (toast feedback) | IMPLEMENTED |

---

## SideNav Routes (Phase 91)

| Label | Path | View |
| ----- | ---- | ---- |
| Dashboard | `/dashboard` | `Dashboard.vue` |
| Certificates | `/certificates` | `Certificates.vue` |
| ACME | `/acme` | `Acme.vue` |
| SCEP | `/scep` | `Scep.vue` |
| NDES | `/ndes` | `Ndes.vue` |
| Provisioners | `/provisioners` | `Provisioners.vue` |
| Templates | `/templates` | `Templates.vue` |
| SSH CA | `/ssh` | `Ssh.vue` |
| Audit Log | `/audit` | `Audit.vue` |
| Webhooks | `/webhooks` | `Webhooks.vue` |
| Settings | `/settings` | `Settings.vue` |

---

## One-Time Secret Display Pattern

Endpoints that return secrets shown only once use a high-visibility warning alert:

| Endpoint | View | Secret Field |
| -------- | ---- | ------------ |
| POST `/api/v1/acme/eab-keys` | `Acme.vue` | `hmac_key` |
| POST `/api/v1/auth/service-accounts` | `Settings.vue` | `api_key` |
| POST `/api/v1/provisioners/token` | `Provisioners.vue` | `token` |

All three use the `ui-alert-warning` design token with explicit copy-to-clipboard controls.
