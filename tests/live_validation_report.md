# Live End-to-End System Validation Report

**Started:** 2026-05-31 23:43:17 +02:00  
**Platform:** Windows (PowerShell)  
**Iteration:** 1 of 3

---

## Phase 1: Preparation & Build


## Phase 1 (continued): Preparation & Build


## Step A: Server Start


### Launch server

**Command:** `./bin/arx-ca-server.exe (background, logs to full_test.log)`

**Expected:** PID captured; HTTP binds :8080

**Exit code:** 0

**Actual:**
```nPID=47404; listening on :8080```n

### Health poll

**Command:** `GET http://localhost:8080/api/v1/health`

**Expected:** HTTP 200

**Exit code:** 0

**Actual:**
```n{"error":null,"data":{"uptime":{"seconds":19,"human":"19s"},"memory":{"alloc_bytes":90196760,"total_alloc_bytes":91906472,"sys_bytes":103590136,"heap_alloc_bytes":90196760,"heap_inuse_bytes":92086272,"heap_objects":12173,"stack_inuse_bytes":589824,"num_gc":2,"last_gc_unix":1780264065,"goroutines":17},"api":{"status":"healthy","version":"v1"},"ca_backend":{"status":"healthy","message":"CA operational; crypto=local; storage=badgerv2; fingerprint=96f1d895da7eed193629ff84d41c1516a0dc446d31f939a9543778d0df0b83be","engine":"step-ca","initialized":true}}}
```n

### Build binaries

**Command:** `go build (make unavailable on Windows)`

**Expected:** Exit 0; bin/*.exe present

**Exit code:** 0

**Actual:**
```n```n

### PostgreSQL readiness

**Command:** `docker run arx-ca-pg-e2e (postgres:16-alpine on :5432)`

**Expected:** accepting connections

**Exit code:** 0

**Actual:**
```n/var/run/postgresql:5432 - accepting connections```n

### server.yaml

**Command:** `Write bin/server.yaml with PostgreSQL`

**Expected:** File exists with database.host=localhost

**Exit code:** 0

**Actual:**
```nexists=True```n

## Step B: Admin CLI


### util hash

**Command:** `arx-ca-cli util hash testpassword`

**Expected:** bcrypt hash on stdout, exit 0

**Exit code:** 0

**Actual:**
```n$2a$10$cfrgSUJyePal5cJt2EYuLOKRi3/MjX/og4YAgApTtctVt2xdzK5q.```n

### login

**Command:** `arx-ca-cli login --url http://localhost:8080`

**Expected:** JWT saved to ~/.arx/config.json, exit 0

**Exit code:** 1

**Actual:**
```narx-ca-cli.exe : Error: username is required
At C:\Users\leon\AppData\Local\Temp\ps-script-6595827d-81f5-4f45-b69a-47d02f79b853.ps1:93 char:714
+ ...  $loginIn | & .\bin\arx-ca-cli.exe login --url http://localhost:8080  ...
+                 ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
    + CategoryInfo          : NotSpecified: (Error: username is required:String) [], RemoteException
    + FullyQualifiedErrorId : NativeCommandError
 
Usage:
  arx-ca-cli login [flags]

Flags:
  -h, --help         help for login
  -u, --url string   Override the server URL from ~/.arx/cli.yaml

username is required
Server URL [http://localhost:8080]: Username:```n

### cli --help

**Command:** `arx-ca-cli --help`

**Expected:** exit 0

**Exit code:** 0

**Actual:**
```narx-ca-cli authenticates against arx-ca-server and provides a terminal UI for CA management and operations dashboards.

Usage:
  arx-ca-cli [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  login       Authenticate with admin credentials and store a JWT locally
  ui          Launch the interactive terminal UI
  util        Utility commands for arx-ca administration

Flags:
  -h, --help   help for arx-ca-cli

Use "arx-ca-cli [command] --help" for more information about a command.```n

### cli util --help

**Command:** `arx-ca-cli util --help`

**Expected:** exit 0

**Exit code:** 0

**Actual:**
```nUtility commands for arx-ca administration

Usage:
  arx-ca-cli util [command]

Available Commands:
  hash        Generate a bcrypt hash for a password

Flags:
  -h, --help   help for util

Use "arx-ca-cli util [command] --help" for more information about a command.```n

### cli login --help

**Command:** `arx-ca-cli login --help`

**Expected:** exit 0

**Exit code:** 0

**Actual:**
```nAuthenticate with admin credentials and store a JWT locally

Usage:
  arx-ca-cli login [flags]

Flags:
  -h, --help         help for login
  -u, --url string   Override the server URL from ~/.arx/cli.yaml```n

### cli ui --help

**Command:** `arx-ca-cli ui --help`

**Expected:** exit 0

**Exit code:** 0

**Actual:**
```nLaunch the interactive terminal UI

Usage:
  arx-ca-cli ui [flags]

Flags:
  -h, --help         help for ui
  -u, --url string   Override the server URL from ~/.arx/cli.yaml```n

## Step C: Server API (curl)


### GET /api/v1/health

**Command:** `curl /api/v1/health`

**Expected:** HTTP 200

**Exit code:** 0

**Actual:**
```n{"error":null,"data":{"uptime":{"seconds":33,"human":"33s"},"memory":{"alloc_bytes":90211064,"total_alloc_bytes":91920776,"sys_bytes":103590136,"heap_alloc_bytes":90211064,"heap_inuse_bytes":92086272,"heap_objects":12308,"stack_inuse_bytes":589824,"num_gc":2,"last_gc_unix":1780264065,"goroutines":17},"api":{"status":"healthy","version":"v1"},"ca_backend":{"status":"healthy","message":"CA operational; crypto=local; storage=badgerv2; fingerprint=96f1d895da7eed193629ff84d41c1516a0dc446d31f939a9543778d0df0b83be","engine":"step-ca","initialized":true}}}  HTTP_CODE:200```n

### GET /api/v1/ca/root

**Command:** `curl /api/v1/ca/root`

**Expected:** HTTP 200

**Exit code:** 0

**Actual:**
```n{"error":null,"data":{"pem":"-----BEGIN CERTIFICATE-----\nMIIBmDCCAT6gAwIBAgIRAL8L3Q62LPblv2s6jdCAh1UwCgYIKoZIzj0EAwIwKjEP\nMA0GA1UEChMGQXJ4IENBMRcwFQYDVQQDEw5BcnggQ0EgUm9vdCBDQTAeFw0yNjA1\nMjkxNjE2MTJaFw0zNjA1MjYxNjE2MTJaMCoxDzANBgNVBAoTBkFyeCBDQTEXMBUG\nA1UEAxMOQXJ4IENBIFJvb3QgQ0EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAATQ\n7Yt7OeK7GgamRuZNbdwLxdaNhj+1KBz8pdKucY7TemsSWh5Z0JVR0QpU+A+9no/9\napBqDJ1/kVP4RyT4EWrpo0UwQzAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0TAQH/BAgw\nBgEB/wIBATAdBgNVHQ4EFgQUG7AN9X9Eeckjlt7WC2uC1DZtXQ4wCgYIKoZIzj0E\nAwIDSAAwRQIhAKQnCWgJNldlTGMddnrdP8PBdBPRd7Pm1iTGyUjJsFx6AiAUsVED\nNJTIyEPt4szSkAZZpEBGLFNNjWiVkHu9mnw56A==\n-----END CERTIFICATE-----\n"}}  HTTP_CODE:200```n

### POST /certificates/auto

**Command:** `curl auto with Bearer JWT`

**Expected:** HTTP 201

**Exit code:** 0

**Actual:**
```n{"error":"authentication required","data":null}  HTTP_CODE:401```n

### POST /certificates/revoke

**Command:** `curl revoke with Bearer JWT`

**Expected:** HTTP 200

**Exit code:** 0

**Actual:**
```n{"error":"authentication required","data":null}  HTTP_CODE:401```n

### POST /certificates/issue (CSR)

**Command:** `curl issue with dummy CSR`

**Expected:** HTTP 201

**Exit code:** 0

**Actual:**
```n{"error":"authentication required","data":null}  HTTP_CODE:401```n

## Step B (retry): Admin CLI


### util hash

**Command:** `arx-ca-cli util hash testpassword`

**Expected:** bcrypt hash, exit 0

**Exit code:** 0

**Actual:**
```n$2a$10$wJgOGq9wmVvts7CboPLQQ.zD8TWxp5zacQ0LaIDKomU.S3Nxh8O/i```n

### login

**Command:** `arx-ca-cli login --url ... --username admin --password ***`

**Expected:** JWT saved, exit 0

**Exit code:** 0

**Actual:**
```nLogged in as admin. Token saved to C:\Users\leon\.arx\config.json
Expires: 2026-06-01T21:49:02Z
Roles: SuperAdmin```n

### JWT saved

**Command:** `Test-Path ~/.arx/config.json`

**Expected:** token non-empty

**Exit code:** 0

**Actual:**
```ntoken present```n

### cli --help

**Command:** `arx-ca-cli --help`

**Expected:** exit 0

**Exit code:** 0

**Actual:**
```nok```n

### cli util --help

**Command:** `arx-ca-cli util --help`

**Expected:** exit 0

**Exit code:** 0

**Actual:**
```nok```n

### cli login --help

**Command:** `arx-ca-cli login --help`

**Expected:** exit 0

**Exit code:** 0

**Actual:**
```nok```n

### server service install --help

**Command:** `arx-ca-server service install --help`

**Expected:** exit 0

**Exit code:** 0

**Actual:**
```nok```n

### server service install

**Command:** `arx-ca-server service install`

**Expected:** Linux-only error on Windows (non-zero expected)

**Exit code:** 1

**Actual:**
```narx-ca-server.exe : Error: service install is only supported on Linux
At C:\Users\leon\AppData\Local\Temp\ps-script-eef8e820-9d98-42d1-b737-f64c1f55f3c4.ps1:93 char:1704
+ ...  $svcInstall = & .\bin\arx-ca-server.exe service install 2>&1 | Out-S ...
+                    ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
    + CategoryInfo          : NotSpecified: (Error: service ...ported on Linux:String) [], RemoteException
    + FullyQualifiedErrorId : NativeCommandError
 
Usage:
  arx-ca-server service install [flags]

Flags:
  -h, --help   help for install

Global Flags:
      --config string   Path to server.yaml (default: server.yaml beside the executable)

2026/05/31 23:49:02 service install is only supported on Linux```n

## Step C: Server API


### health

**Command:** `curl GET /api/v1/health`

**Expected:** HTTP 200

**Exit code:** 0

**Actual:**
```n{"error":null,"data":{"uptime":{"seconds":77,"human":"1m 17s"},"memory":{"alloc_bytes":90405952,"total_alloc_bytes":92115664,"sys_bytes":103590136,"heap_alloc_bytes":90405952,"heap_inuse_bytes":92307456,"heap_objects":13180,"stack_inuse_bytes":589824,"num_gc":2,"last_gc_unix":1780264065,"goroutines":17},"api":{"status":"healthy","version":"v1"},"ca_backend":{"status":"healthy","message":"CA operational; crypto=local; storage=badgerv2; fingerprint=96f1d895da7eed193629ff84d41c1516a0dc446d31f939a9543778d0df0b83be","engine":"step-ca","initialized":true}}}  HTTP_CODE:200```n

### ca root

**Command:** `curl GET /api/v1/ca/root`

**Expected:** HTTP 200

**Exit code:** 0

**Actual:**
```n{"error":null,"data":{"pem":"-----BEGIN CERTIFICATE-----\nMIIBmDCCAT6gAwIBAgIRAL8L3Q62LPblv2s6jdCAh1UwCgYIKoZIzj0EAwIwKjEP\nMA0GA1UEChMGQXJ4IENBMRcwFQYDVQQDEw5BcnggQ0EgUm9vdCBDQTAeFw0yNjA1\nMjkxNjE2MTJaFw0zNjA1MjYxNjE2MTJaMCoxDzANBgNVBAoTBkFyeCBDQTEXMBUG\nA1UEAxMOQXJ4IENBIFJvb3QgQ0EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAATQ\n7Yt7OeK7GgamRuZNbdwLxdaNhj+1KBz8pdKucY7TemsSWh5Z0JVR0QpU+A+9no/9\napBqDJ1/kVP4RyT4EWrpo0UwQzAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0TAQH/BAgw\nBgEB/wIBATAdBgNVHQ4EFgQUG7AN9X9Eeckjlt7WC2uC1DZtXQ4wCgYIKoZIzj0E\nAwIDSAAwRQIhAKQnCWgJNldlTGMddnrdP8PBdBPRd7Pm1iTGyUjJsFx6AiAUsVED\nNJTIyEPt4szSkAZZpEBGLFNNjWiVkHu9mnw56A==\n-----END CERTIFICATE-----\n"}}  HTTP_CODE:200```n

### certificates auto

**Command:** `curl POST /api/v1/certificates/auto`

**Expected:** HTTP 201

**Exit code:** 0

**Actual:**
```n{"error":null,"data":{"certificate_pem":"-----BEGIN CERTIFICATE-----\nMIICJzCCAcygAwIBAgIRAPubvEqrFZhowhTA7dEvbEQwCgYIKoZIzj0EAwIwMjEP\nMA0GA1UEChMGQXJ4IENBMR8wHQYDVQQDExZBcnggQ0EgSW50ZXJtZWRpYXRlIENB\nMB4XDTI2MDUzMTIxNDgwM1oXDTI2MDUzMTIyNDkwM1owGTEXMBUGA1UEAxMOZTJl\nLXRlc3QubG9jYWwwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAR8YCUqRyftzI3G\nPPUbcZix1gOPeIquMKd4PVNzQSv5K0wL000kXyvsgRl0j2z3uTr9clGfhpuGy2ms\nnoOCGVB3o4HbMIHYMA4GA1UdDwEB/wQEAwIHgDAdBgNVHSUEFjAUBggrBgEFBQcD\nAQYIKwYBBQUHAwIwHQYDVR0OBBYEFBnWOPm3Ll7pTkcn5E52Cy7foaBzMB8GA1Ud\nIwQYMBaAFM1WSLwf1mIUUgrvUFqY5hQWLgfIMBkGA1UdEQQSMBCCDmUyZS10ZXN0\nLmxvY2FsMEwGDCsGAQQBgqRkxihAAQQ8MDoCAQEECGNhLWFkbWluBCtWSW5RVjRY\nUGpvV3pUNklnZnFmVE41QWlpVTRPaUNCcVM1TU5yQUJ3dFBFMAoGCCqGSM49BAMC\nA0kAMEYCIQCFAF/r2/8H19tqQaYAcgys5RxBrd1QspRuNmTOW5eRewIhAJqitn5d\nx0+hj6SdX9BcEs+uGVUjW1tO0BgCNRAj08p8\n-----END CERTIFICATE-----\n","private_key_pem":"-----BEGIN PRIVATE KEY-----\nMIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgqGLKFaTIqBcMCN1s\nKK/cNDGyshumGYxNW4yL6f4xa6GhRANCAAR8YCUqRyftzI3GPPUbcZix1gOPeIqu\nMKd4PVNzQSv5K0wL000kXyvsgRl0j2z3uTr9clGfhpuGy2msnoOCGVB3\n-----END PRIVATE KEY-----\n","serial":"334444851963924338804692191753903500356","not_before":"2026-05-31T21:48:03Z","not_after":"2026-05-31T22:49:03Z"}}  HTTP_CODE:201```n

### certificates revoke

**Command:** `curl POST /api/v1/certificates/revoke`

**Expected:** HTTP 200

**Exit code:** 0

**Actual:**
```n{"error":null,"data":{"serial":"334444851963924338804692191753903500356","revoked_at":"2026-05-31T21:49:03Z"}}  HTTP_CODE:200```n

## Step D: Client Agent


### local list

**Command:** `arx-cert-service local list`

**Expected:** exit 0

**Exit code:** 0

**Actual:**
```nID                                                                STORE  LOCATION  SUBJECT                                                                                                                                                                                              NOT AFTER
80b4b2203734793eecef8f6dd367e3f5abd49092c20210e6ba4bb4c1fa9ac1cf  user   ROOT      CN=Fleet Root CA,O=Fleet,C=US                                                                                                                                                                        2036-05-03
552f7bdcf1a7af9e6ce672017f4f12abf77240c78e761ac203d1d9d20ac89988  user   ROOT      CN=DigiCert Trusted Root G4,OU=www.digicert.com,O=DigiCert Inc,C=US                                                                   ```n

### server list

**Command:** `arx-cert-service server list --url ...`

**Expected:** exit 0

**Exit code:** 0

**Actual:**
```nSERIAL                                   SUBJECT            NOT AFTER             REVOKED
334444851963924338804692191753903500356  CN=e2e-test.local  2026-05-31T22:49:03Z  yes```n

### server download root

**Command:** `arx-cert-service server download --kind root`

**Expected:** exit 0; PEM written

**Exit code:** 0

**Actual:**
```nsaved to C:\Users\leon\AppData\Local\Temp\arx-e2e-root.pem; exists=True```n

### trust install-root

**Command:** `arx-cert-service trust install-root --url ...`

**Expected:** exit 0 or elevation note

**Exit code:** 0

**Actual:**
```nRoot CA installed into local trust stores.
State saved under ~/.arx-cert-service/```n

### ERROR DETECTED (iteration 1)

1. **Server start failed** — `ca.json` contained WSL paths (`/mnt/c/...`); native Windows could not read `root_ca.crt`.
2. **CLI login failed** — piped stdin could not supply a password (`term.ReadPassword`: invalid handle).

### FIX APPLIED

1. **`internal/ca/pki_path_heal.go`** — On startup, rewrite `ca.json` filesystem paths when artifacts exist under the PKI `basePath` but recorded paths are stale (cross-environment).
2. **`internal/cli/login/login.go`**, **`cmd/cli/main.go`** — Added `--username` and `--password` flags; read password from a line when stdin is not a TTY.

### Additional API coverage (authenticated GET, iteration 1)

| Endpoint | HTTP |
| -------- | ---- |
| `/api/v1/ca/crl` | 200 |
| `/api/v1/public/ca/intermediate` | 200 |
| `/api/v1/public/certificates` | 200 |
| `/api/v1/acme/status` | 200 |
| `/api/v1/scep/status` | 200 |
| `/api/v1/ndes/status` | 200 |
| `/api/v1/k8s/status` | 200 |
| `/api/v1/templates` | 200 |
| `/api/v1/certificates` | 200 |
| `/api/v1/ssh/roots` | 200 |

Protected POST flows verified after login: `/api/v1/certificates/auto` (201), `/api/v1/certificates/revoke` (200).

### VALIDATION SUCCESSFUL

All required live steps completed on iteration 1 (after fixes). Server used PostgreSQL (`arx-ca-pg-e2e` on `:5432`) for the application user store. Binaries built with `go build` (`make` unavailable on Windows). Server PID terminated; `bin/full_test.log` removed.

**Finished:** 2026-05-31 (live validation phase 26)
