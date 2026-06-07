# Rapid UI Development Loop

This workflow lets you iterate on the Vue WebUI with Vite Hot Module Replacement (HMR) without rebuilding the Go binary on every change. The Vite dev server serves the frontend; API traffic is proxied to the running `arx` backend.

## Prerequisites

- A built `arx` binary (`make build` or `make build-all`)
- Initialized server configuration (`./bin/arx server config init`)
- Node.js 20+ and npm in `webui/`

## Two-terminal workflow

### Step 1 — Start the Go backend (Terminal A)

Start the API once and leave it running:

```bash
./bin/arx server start
```

Or run it in the background:

```bash
./bin/arx server start &
```

The Vite dev server binds `0.0.0.0:5173` so forwarded ports work from both IPv4 and IPv6 clients (Devcontainers, WSL, remote hosts). The proxy forwards `/api` to `http://127.0.0.1:8080` by default, matching the stock HTTP API (`server.port: 8080`, TLS disabled).

For a TLS WebUI listener on `https://localhost:8443` with `webui.proxy_api: true`, change the proxy `target` in `webui/vite.config.ts` to `https://localhost:8443` and set `secure: false` for self-signed certificates.

The Axios client sends `withCredentials: true` so the API `arx_session` HttpOnly cookie set by `POST /api/v1/auth/login` is stored and replayed on proxied requests.

### Step 2 — Start the Vite dev server (Terminal B)

```bash
cd webui
npm ci          # first time only
npm run dev
```

### Step 3 — Open the browser

Navigate to the Vite dev server URL (typically **http://localhost:5173**).

- **UI changes** hot-reload instantly via HMR; no Go rebuild is required.
- **API calls** from the browser go to same-origin `/api/v1` on the Vite port; Vite proxies them to the Go backend.

The Axios client in `webui/src/api/client.ts` resolves the API base URL to `{origin}/api/v1` in the browser, so no `VITE_API_BASE_URL` override is needed when using this proxy.

## Production vs development

| Mode | Frontend | API |
| ---- | -------- | --- |
| **Development** | Vite dev server (`npm run dev`) | Proxied `/api` → backend |
| **Production** | Static assets in `webui.ui_dir` | Same-origin via WebUI listener or direct API URL |

Build production assets with `cd webui && npm run build`, or use `make webui` / `arx server ui download` for release tarballs.
