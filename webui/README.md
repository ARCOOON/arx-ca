# Arx CA WebUI

Vue 3 management console for the Arx Certificate Authority API. Built with Vite, TypeScript, Pinia, Vue Router, Tailwind CSS v4, Shadcn Vue (radix-vue), and Axios.

Styling is aligned with the **ARX ecosystem** (`arx-dns`): flat enterprise UI, crisp borders, no drop shadows, and native dark/light mode via the `dark` class on `<html>`.

## Development

Run the Go backend once, then start Vite in a second terminal for HMR without rebuilding `arx`:

```bash
# Terminal A (repository root)
./bin/arx server start

# Terminal B
cd webui
npm ci
npm run dev
```

Open **http://localhost:5173**. Vite binds `0.0.0.0:5173` and proxies `/api` to `http://127.0.0.1:8080` by default. See the [Wiki → Development Workflow](https://github.com/ARCOOON/arx-ca/wiki/Development-Workflow) for TLS WebUI proxy targets and the full rapid UI loop.

Optional: set `VITE_API_BASE_URL` (see `.env.example`) when the API is not proxied on the same origin. By default the browser uses `{origin}/api/v1`.

### Install dependencies (fresh clone)

```bash
cd webui
npm ci
```

Core styling packages (mirrored from `arx-dns`):

| Package | Role |
| ------- | ---- |
| `tailwindcss` + `@tailwindcss/vite` | Tailwind CSS v4 (CSS-first config in `src/style.css`) |
| `tw-animate-css` | Shadcn animation utilities |
| `radix-vue` / `reka-ui` | Accessible primitives for Shadcn Vue |
| `class-variance-authority`, `clsx`, `tailwind-merge` | Variant styling and `cn()` helper |
| `@vueuse/core` | Composables used by Shadcn inputs |

Initialize or add Shadcn components with the CLI (optional):

```bash
npx shadcn-vue@latest init   # uses components.json
npx shadcn-vue@latest add button input dialog table
```

There is **no** `tailwind.config.ts` — Tailwind v4 theme tokens live in `src/style.css` (`@theme inline` block), matching `arx-dns`.

## Production build

```bash
npm run build
```

Output is written to `dist/`. Package with the release workflow or copy into `webui.ui_dir` on the server.

### Build performance

`vite.config.ts` sets explicit `resolve.extensions` and an absolute `@` → `src` alias (mirrored in `tsconfig.app.json` `paths`) so Rolldown does not probe the filesystem for extension guesses.

Import Lucide icons from per-icon ESM paths (for example `lucide-vue-next/dist/esm/icons/shield-check.js`) or from the `lucide-vue-next` package root in layout/views that tree-shake well. Per-icon Lucide types are declared in `src/lucide-icons.d.ts`.

There are no application-level barrel files under `src/`; use direct module paths (or `@/…` aliases) for local code.

## Theming & design tokens

| File | Purpose |
| ---- | ------- |
| `components.json` | Shadcn Vue configuration (`new-york` style, CSS variables) |
| `src/style.css` | Tailwind v4 import, OKLCH palette, sidebar tokens, legacy `ui-*` compatibility classes |
| `src/composables/useTheme.ts` | Toggles `dark` on `<html>`; persists `arx_theme` in `localStorage` |

Fonts (Google Fonts, loaded in `index.html`):

- **Source Sans 3** — body / UI (`font-sans`)
- **Noto Sans** — headings (`font-heading`)

Monospace is reserved for `code` blocks and certificate text areas only.

## Routes

| Path | View | API usage |
| ---- | ---- | --------- |
| `/login` | Login | `POST /auth/login` |
| `/dashboard` | Dashboard | `GET /health`, `GET /certificates` (count) |
| `/certificates` | Certificates | `GET /certificates`, `POST /certificates/issue`, `POST /certificates/generate`, `GET /crl` |
| `/acme` | ACME | `GET /acme/status` |
| `/scep` | SCEP | `GET /scep/status` |
| `/settings` | Settings | `GET /health`, session metadata |

Authenticated routes render inside `src/components/layout/AppShell.vue` with a collapsible sidebar (hamburger drawer on viewports below `md`), top bar (theme toggle via Shadcn `Switch`), and flat bordered surfaces. The shell uses a fixed viewport-height layout: the sidebar and top bar remain pinned while only the main content region scrolls.

**UI Preferences** (Settings → localStorage): notification drawer layout, default sidebar collapse, and **Show Developer API Hints** (`arx_ui_show_api_hints`, default off). Audit and Certificates search panels are collapsed by default.

The Certificates view exposes CRL download (`GET /crl`), CSR signing, native generation with a server-built ZIP bundle (`certificate.crt`, `certificate.pem`, `private.key`) via `POST /certificates/generate?format=zip`, plus standard/extended key usage toggles, and SuperAdmin-only escrowed key retrieval in the certificate details modal. The Dashboard exposes **Download CA Bundle (.zip)** (`GET /ca/chain`, `ca-bundle.zip`).

## Structure

- `src/api/` — Axios client and endpoint helpers
- `src/composables/` — Shared reactive state (theme, notifications, UI preferences)
- `src/components/layout/` — App shell, navigation, top bar
- `src/components/ui/` — Shadcn Vue primitives (button, input, dialog, popover, …) plus app wrappers (`DataTable`, `Modal`, `StatusBadge`, `TagInput`)
- `src/lib/utils.ts` — `cn()` class merge helper
- `src/router/` — Route table and auth guard
- `src/store/auth.ts` — JWT and roles in `localStorage`
- `src/views/` — Page components
- `src/lucide-icons.d.ts` — TypeScript module declarations for per-icon Lucide imports
