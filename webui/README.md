# Arx CA WebUI

Vue 3 management console for the Arx Certificate Authority API. Built with Vite, TypeScript, Pinia, Vue Router, Tailwind CSS v4, and Axios.

## Development

```bash
npm ci
npm run dev
```

Optional: set `VITE_API_BASE_URL` (see `.env.example`) when the API is not proxied on the same origin. By default the browser uses `{origin}/api/v1`.

## Production build

```bash
npm run build
```

Output is written to `dist/`. Package with the release workflow or copy into `webui.ui_dir` on the server.

## Routes

| Path | View | API usage |
| ---- | ---- | --------- |
| `/login` | Login | `POST /auth/login` |
| `/dashboard` | Dashboard | `GET /health`, `GET /certificates` (count) |
| `/certificates` | Certificates | `GET /certificates`, `POST /certificates/issue` |
| `/acme` | ACME | `GET /acme/status` |
| `/scep` | SCEP | `GET /scep/status` |
| `/settings` | Settings | `GET /health`, session metadata |

Authenticated routes render inside `src/components/layout/AppShell.vue` with a collapsible sidebar and top bar.

## Structure

- `src/api/` — Axios client and endpoint helpers
- `src/components/layout/` — App shell, navigation, top bar
- `src/components/ui/` — Shared flat UI primitives
- `src/router/` — Route table and auth guard
- `src/store/auth.ts` — JWT and roles in `localStorage`
- `src/views/` — Page components
