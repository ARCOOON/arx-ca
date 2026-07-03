# Arx CA Console — WebUI

Operator console for the Arx CA control plane. Rebuilt greenfield with
**Vue 3 + Vite + TypeScript**, **Tailwind CSS v4**, and **shadcn-vue**. The
design merges the flat, minimalist arx-dns ecosystem standard with **WinUI 3**
cues: layered neutral surfaces for elevation (instead of heavy shadows), smooth
rounded corners (`--radius: 0.5rem`), and clean subtle borders.

## Stack

| Concern        | Choice                                             |
| -------------- | -------------------------------------------------- |
| Framework      | Vue 3 (Composition API, `<script setup>`)          |
| Build tool     | Vite                                               |
| Styling        | Tailwind CSS v4 (CSS-first, `@tailwindcss/vite`)   |
| Components     | shadcn-vue (reka-ui primitives)                    |
| Icons          | `@lucide/vue`                                      |
| Routing        | Vue Router 4                                        |
| State          | Pinia (`auth`, `theme`, `notifications`)           |
| HTTP transport | Native `fetch` (envelope-aware client)             |
| Toasts         | vue-sonner                                          |

## Project layout

```
src/
├── api/            # fetch-based API client + one module per backend domain
├── assets/         # index.css (design tokens + Tailwind + shadcn theme)
├── components/
│   ├── layout/     # AppShell, Sidebar (WinUI NavigationView), TopBar
│   ├── ui/         # shadcn-vue primitives
│   ├── ThemeSwitcher.vue
│   ├── NotificationBell.vue
│   ├── PostUpdateChangelogModal.vue
│   ├── StatCard.vue
│   └── StatusBadge.vue
├── composables/    # useUpdater (post-update changelog drift detection)
├── lib/            # utils (cn), format, errors, download helpers
├── router/         # routes + auth guard
├── stores/         # Pinia stores
├── types/          # api.ts — TypeScript contracts for /api/v1
└── views/          # Login, Dashboard, Certificates, Ssh, Settings, NotFound
```

## Theming

- Class-based dark mode: `.dark` is toggled on `<html>`.
- Theme switcher offers **Light / Dark / Auto (system)**, persisted to
  `localStorage` under `arx_theme`. `Auto` follows `prefers-color-scheme` and
  reacts live to OS changes.
- All colors are CSS variables (`--background`, `--primary`, `--sidebar`, …)
  defined in `src/assets/index.css` and mapped to Tailwind via `@theme inline`.

## API integration

The client targets the Go backend under `/api/v1` and unwraps the standard
`{ error, data }` envelope. Authentication uses a JWT stored in `localStorage`
(`arx_auth_token`) sent as `Authorization: Bearer`, plus `credentials: include`
for the `arx_session` cookie. A `401` on any non-login request clears the
session and redirects to `/login`.

Binary endpoints (CA chain zip, certificate bundle zip, PEM downloads) use the
`requestBlob` helper. Live notifications use `EventSource` against
`/api/v1/notifications/stream` with the token passed as `access_token`.

## Development

```bash
npm install
npm run dev        # Vite dev server on :5173, proxies /api -> http://127.0.0.1:8080
```

Run the Go backend separately (`arx-ca server start`). The dev proxy makes the
SPA use same-origin `/api/v1` just like production.

## Build

```bash
npm run build      # vue-tsc type-check + vite build -> dist/
npm run preview
```

The repository Makefile / GoReleaser package `dist/` into `webui-dist.tar.gz`,
which the `arx-ca` binary serves as a static SPA.

## Core views

- **Dashboard** — certificate stats, server & CA-backend health, CA chain
  download, provisioners.
- **Certificates** — search/filter, issue (CSR / provisioner token / auto),
  inspect details, revoke, download bundle.
- **SSH CA** — stats, issued certificate inventory, CA public keys, generate
  user/host certificates, inspect a certificate.
- **Settings** — session roles, **Auto-updater** configuration, CA PEM
  downloads, service-account API-key creation (SuperAdmin).
- **PostUpdateChangelogModal** — shown once after a version drift when
  `view_changelog_after_update` is enabled.
