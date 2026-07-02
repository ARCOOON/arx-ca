# Arx CA WebUI

Vue 3 management console for the Arx Certificate Authority API. The styling stack is synchronized with the **ARX ecosystem** (`arx-dns`): Tailwind CSS v4 (CSS-first configuration), Shadcn Vue (`new-york` style), Radix Vue primitives, and shared typography (Source Sans 3 + Noto Sans).

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

Open **http://localhost:5173**. Vite binds `0.0.0.0:5173` and proxies `/api` to `http://127.0.0.1:8080` by default.

Optional: set `VITE_API_BASE_URL` (see `.env.example`) when the API is not proxied on the same origin. By default the browser uses `{origin}/api/v1`.

## Install dependencies

From `webui/`:

```bash
npm ci
```

Key packages (aligned with `arx-dns`):

| Package | Purpose |
| ------- | ------- |
| `tailwindcss` + `@tailwindcss/vite` | Tailwind CSS v4 via Vite plugin |
| `radix-vue` / `reka-ui` | Accessible primitives for Shadcn Vue |
| `class-variance-authority`, `clsx`, `tailwind-merge` | Component variants and class merging (`cn()`) |
| `tw-animate-css` | Dialog/popover enter/exit animations |
| `@vueuse/core` | Composables used by Shadcn inputs |
| `vue-sonner` | Toast notifications |
| `lucide-vue-next` | Icons |

## Styling architecture

Tailwind v4 does **not** use `tailwind.config.ts`. Theme tokens live in `src/style.css` (`@theme inline`, `:root`, `.dark`) and are referenced by Shadcn components via CSS variables (`--background`, `--foreground`, `--primary`, etc.).

| File | Role |
| ---- | ---- |
| `components.json` | Shadcn Vue schema (`new-york`, `neutral` base, `@/` aliases) |
| `src/style.css` | Tailwind import, design tokens, dark mode (`.dark` on `<html>`), layout utilities |
| `src/lib/utils.ts` | `cn()` helper (`clsx` + `tailwind-merge`) |
| `src/components/ui/` | Shadcn Vue primitives (Button, Input, Dialog, Table patterns, Switch, etc.) |
| `index.html` | Google Fonts: Source Sans 3 (UI), Noto Sans (headings) |

Dark/light mode toggles the `dark` class on `document.documentElement` (persisted in `localStorage` key `arx_theme`). See `src/composables/useTheme.ts`.

Design language: flat enterprise aesthetic — crisp borders, minimal shadows, semantic color tokens, non-monospace fonts for standard UI (monospace reserved for code/PEM fields).

## Production build

```bash
npm run build
```

Output is written to `dist/`. Package with the release workflow or copy into `webui.ui_dir` on the server.

### Build performance

`vite.config.ts` sets explicit `resolve.extensions` and an absolute `@` → `src` alias (mirrored in `tsconfig.app.json` `paths`) so Rolldown does not probe the filesystem for extension guesses.

Import Lucide icons from per-icon ESM paths (for example `lucide-vue-next/dist/esm/icons/shield-check.js`), not from the package root entry. Per-icon Lucide types are declared in `src/lucide-icons.d.ts`.

## Routes

| Path | View | API usage |
| ---- | ---- | --------- |
| `/login` | Login | `POST /auth/login` |
| `/dashboard` | Dashboard | `GET /health`, `GET /certificates` (count) |
| `/certificates` | Certificates | `GET /certificates`, `POST /certificates/issue`, `POST /certificates/generate`, `GET /crl` |
| `/acme` | ACME | `GET /acme/status` |
| `/scep` | SCEP | `GET /scep/status` |
| `/settings` | Settings | `GET /health`, session metadata |

Authenticated routes render inside `src/components/layout/AppShell.vue` with a collapsible sidebar (hamburger drawer on viewports below `md`), top bar (theme toggle via Shadcn `Switch`), and Shadcn/Tailwind theming. The shell uses a fixed viewport-height layout: the sidebar and top bar remain pinned while only the main content region scrolls.

**UI Preferences** (Settings → localStorage): notification drawer layout, default sidebar collapse, and **Show Developer API Hints** (`arx_ui_show_api_hints`, default off).

## Structure

- `src/api/` — Axios client and endpoint helpers
- `src/composables/` — Shared reactive state (theme, notifications, UI preferences, toasts)
- `src/components/layout/` — App shell, navigation, top bar
- `src/components/ui/` — Shadcn Vue primitives and thin wrappers (`DataTable`, `Modal`, `StatusBadge`, `TagInput`)
- `src/lib/utils.ts` — `cn()` class merge helper
- `src/router/` — Route table and auth guard
- `src/store/auth.ts` — JWT and roles in `localStorage`
- `src/views/` — Page components
- `src/lucide-icons.d.ts` — TypeScript module declarations for per-icon Lucide imports

## Add Shadcn components

With `components.json` in place, add new primitives the same way as `arx-dns`:

```bash
cd webui
npx shadcn-vue@latest add <component>
```

Components are generated under `src/components/ui/` using the shared tokens in `src/style.css`.
