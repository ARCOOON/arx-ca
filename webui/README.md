# ARX CA WebUI

Vue 3 operator console for the ARX Certificate Authority platform. Rebuilt with **Vite**, **Tailwind CSS v4**, and **Shadcn-Vue** using a WinUI 3–inspired flat design system.

## Stack

| Layer | Technology |
| ----- | ---------- |
| Framework | Vue 3 (Composition API, `<script setup>`) |
| Build | Vite 8 |
| Styling | Tailwind CSS v4 + Shadcn-Vue (Reka UI primitives) |
| State | Pinia |
| Routing | Vue Router 5 |
| HTTP | Axios (`/api/v1` same-origin proxy) |
| Icons | Lucide (`@lucide/vue`) |

## Development

```bash
cd webui
npm install
npm run dev
```

The dev server proxies `/api` to `http://127.0.0.1:8080`. Override the API base with `VITE_API_BASE_URL` (see `.env.example`).

## Production build

```bash
npm run build
```

From the repository root, `make webui` installs dependencies, builds, and packages `build/webui-dist.tar.gz`.

## Views

| Route | Description |
| ----- | ----------- |
| `/dashboard` | Server health, CA metadata, CRL/chain downloads |
| `/certificates` | X.509 inventory, CSR signing, native issuance, revocation |
| `/ssh` | SSH CA operations (user/host certs, inspect, roots) |
| `/settings` | Session info, public API URL, **Auto-Updater** configuration |

## Theming

Theme preference (`light` / `dark` / `auto`) is stored in `localStorage` under `arx_theme_preference`. Auto mode follows the OS `prefers-color-scheme` media query.

## Authentication

JWT bearer tokens are persisted in `localStorage` (`arx_auth_token`, `arx_auth_roles`). Unauthenticated requests to protected routes redirect to `/login`.
