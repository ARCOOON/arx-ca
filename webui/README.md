# ARX CA WebUI

Vue 3 + Vite operator console for ARX CA.

## Stack

| Technology     | Version | Role                                 |
|----------------|---------|--------------------------------------|
| Vue 3          | ^3.5    | Reactive SPA framework               |
| Vite 6         | ^6.0    | Build tool and dev server            |
| Tailwind CSS 4 | ^4.0    | Utility-first styling (CSS-native)   |
| Radix Vue      | ^1.9    | Accessible headless UI primitives    |
| Pinia 2        | ^2.3    | Lightweight state management         |
| Vue Router 4   | ^4.5    | Client-side routing                  |
| Axios          | ^1.7    | HTTP client with auth interceptors   |

## Design System

The UI follows a **WinUI 3 × ARX** aesthetic:

- **Flat surfaces** with subtle background elevation shifts instead of heavy shadows
- **Smooth rounded corners** (`radius: 0.5rem` globally)
- **Structured spacing** with consistent 4-point grid
- **Clean borders** using `hsl(--border)` tokens
- **Three-mode theme switcher**: Light / Dark / Auto (OS-level)

CSS design tokens are defined via Tailwind v4 `@theme` directive in `src/assets/main.css`.

## Development

```bash
# Install dependencies
npm install

# Start dev server (proxies /api → http://127.0.0.1:8080)
npm run dev

# Type-check + production build
npm run build

# Preview production build
npm run preview
```

The dev server runs on `http://localhost:5173` and proxies all `/api/...` requests to `http://127.0.0.1:8080` (the ARX CA backend).

## Project Structure

```
src/
├── api/              # Axios API clients (one file per domain)
├── assets/           # Global CSS (main.css — design tokens)
├── components/
│   ├── layout/       # AppShell, SideNav, TopBar
│   ├── modals/       # PostUpdateChangelogModal
│   └── ui/           # Primitive components (Button, Input, Dialog, …)
├── composables/      # useTheme, useToast, useNotifications, useUpdater
├── router/           # Vue Router (auth guard)
├── store/            # Pinia stores (auth, notifications)
├── types/            # TypeScript API types
├── utils/            # cn(), format, errors
└── views/            # Route-level page components
```

## API Proxy

Set `VITE_API_BASE_URL` to override the default same-origin `/api/v1` base URL:

```bash
VITE_API_BASE_URL=https://ca.example.com/api/v1 npm run build
```

## Authentication

JWT tokens are stored in `localStorage` under `arx_auth_token`. All API requests attach the token as a `Bearer` header. A 401 response from any endpoint (except `/auth/login`) triggers automatic logout and redirects to `/login`.

## Theme Persistence

Theme preference (`light` | `dark` | `auto`) is stored in `localStorage` under `arx_theme`. `auto` tracks the OS `prefers-color-scheme` media query and reacts to live changes.
