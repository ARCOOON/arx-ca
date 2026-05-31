# Phase 28: Dual-Mode E2E Validation Report

**Started:** 2026-06-01 (Dev Container, Linux/WSL2)  
**Completed:** 2026-06-01  
**Iteration:** 1 (passed after one build fix)

---

## Environment

| Item | Value |
|------|-------|
| Workspace | `/workspaces/arx-ca` |
| PostgreSQL | `db:5432`, user `arx`, password `devpassword`, database `arx` |
| Bootstrap login | user `admin`, password `ArxRootCA-Bootstrap-Admin-2026!` |
| Docker-in-Docker | Active (`docker compose` builds `Dockerfile` → `arx-ca-server:latest`) |
| Health endpoint | `GET /api/v1/health` (no `/health` route; returns 404) |

**Note:** The `db` hostname did not resolve until the devcontainer PostgreSQL service was started (`docker compose -f .devcontainer/docker-compose.yml up -d db`) and `172.19.0.2 db` was added to `/etc/hosts` for this workspace container.

---

## 1. Bare-Metal Validation

### 1.1 Build (`make build`)

**Command:** `make build`  
**Expected:** Exit 0; `bin/arx-ca-server`, `bin/arx-ca-cli`, `bin/arx-cert-service` present  
**Result:** PASS (after Makefile fix; see FIX section)

### 1.2 Configuration (`bin/server.yaml`)

**Settings applied:**

```yaml
database:
  host: db
  user: arx
  password: devpassword
  dbname: arx
```

**Result:** PASS — server connected to PostgreSQL on first start (no DB retry warnings in log).

### 1.3 Server start

**Command:** `cd bin && ./arx-ca-server` (background, log: `/tmp/arx-bare-metal.log`)  
**PID:** `12501` (written to `bin/server.pid`)  
**Log excerpt:** `arx-ca-server listening on :8080`  
**Result:** PASS

### 1.4 CLI suite

| Step | Command | Exit | Result |
|------|---------|------|--------|
| Hash utility | `./bin/arx-ca-cli util hash testpassword` | 0 | bcrypt hash emitted |
| Login | `./bin/arx-ca-cli login --url http://localhost:8080 --username admin --password '…'` | 0 | JWT saved to `~/.arx/config.json`, role SuperAdmin |
| List info | `./bin/arx-ca-cli --help`, `util --help`, `login --help` | 0 | Command tree listed |

### 1.5 HTTP (curl)

| Endpoint | Auth | HTTP | Result |
|----------|------|------|--------|
| `GET /api/v1/health` | none | 200 | `api.status=healthy`, `ca_backend.status=healthy` |
| `GET /api/v1/ca/root` | none | 200 | PEM root certificate |
| `GET /api/v1/certificates` | Bearer JWT | 200 | Certificate list (`total`: 2) |

### 1.6 Teardown

**Command:** `kill $(cat bin/server.pid)`  
**Result:** PASS — port 8080 no longer accepts connections.

---

## 2. Containerized Validation

### 2.1 Compose / Dockerfile review

- **File:** `docker-compose.yml` — service `arx-ca` uses `build.context: .` and `dockerfile: Dockerfile`.
- **File:** `Dockerfile` — multi-stage build compiles `./cmd/server` → `/app/arx-ca-server`.

### 2.2 Deploy

**Command:** `cp .env.example .env` (JWT secret generated), `docker compose up -d --build`  
**Result:** PASS — image `arx-ca-server:latest` built and container `arx-ca` started.

### 2.3 Logs (after 5s wait)

**Command:** `docker compose logs`  
**Key lines:**

- PKI bootstrap completed under `/app/.pki/`
- Badger CA store opened
- `arx-ca-server listening on :8080`

**Note:** The production compose stack uses embedded BadgerDB/SQLite in `/app/data` volume, not the devcontainer PostgreSQL service. Application user DB for Docker is SQLite unless `CA_API_DB_*` is set.

**Result:** PASS — container started without crash.

### 2.4 Health check

**Command:** `curl http://localhost:8080/api/v1/health`  
**HTTP:** 200  
**Result:** PASS

**Command:** `curl http://localhost:8080/health`  
**HTTP:** 404 (expected; API health is under `/api/v1/health`)

### 2.5 Teardown

**Command:** `docker compose down -v`  
**Result:** PASS — container and `arx-ca_default` network removed.

---

## 3. Self-Healing

### ERROR

**Phase:** Bare-metal build  
**Command:** `make build`  
**Output:**

```text
error obtaining VCS status: exit status 128
	Use -buildvcs=false to disable VCS stamping.
make: *** [Makefile:27: build-server] Error 1
```

**Cause:** Go 1.22+ embeds VCS metadata by default; this workspace’s `.git` ownership triggers `fatal: detected dubious ownership`, which breaks `go build` VCS stamping.

### FIX

**File:** `Makefile`  
**Change:** Export `GOFLAGS=-buildvcs=false` for all `go build` targets.

```makefile
GOFLAGS := -buildvcs=false
export GOFLAGS
```

**Verification:** `make build` completed with exit 0; bare-metal and Docker builds succeeded on iteration 1 restart.

---

## 4. Summary

| Mode | Build | Start | CLI | Public API | Protected API | Teardown |
|------|-------|-------|-----|------------|---------------|----------|
| Bare-metal | PASS | PASS | PASS | PASS | PASS | PASS |
| Docker | PASS | PASS | N/A | PASS | N/A | PASS |

**Overall:** Phase 28 dual-mode E2E validation **PASSED**.
