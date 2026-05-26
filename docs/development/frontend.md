# Frontend Development

## Architecture Overview

The ParkirPintar frontend is a single-page application built with:

- **React 19** — UI library with functional components and hooks
- **Vite 6** — Build tool and dev server with HMR
- **React Router DOM 7** — Client-side routing with protected routes
- **Redux Toolkit + React Redux** — Global state management (available for complex state)
- **React Context** — Auth and reservation state (lightweight, co-located with providers)

State management uses a hybrid approach: React Context handles auth and active reservation state, while Redux Toolkit is available for more complex cross-cutting concerns.

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `react` / `react-dom` | UI rendering (v19) |
| `react-router-dom` | SPA routing with nested layouts |
| `@reduxjs/toolkit` | Redux state management with simplified API |
| `react-redux` | React bindings for Redux |
| `jose` | JWT handling (token generation/validation) |

Dev dependencies: `vite`, `@vitejs/plugin-react`, TypeScript type definitions for React.

## Component Structure

```
frontend/src/
├── api/
│   └── client.js          # HTTP client with auth interceptor
├── components/
│   ├── domain/            # Business-specific components
│   │   ├── AvailabilityBar.jsx
│   │   ├── BillingBreakdown.jsx
│   │   ├── CountdownTimer.jsx
│   │   ├── FloorGrid.jsx
│   │   ├── HealthStatusCard.jsx
│   │   ├── LocationSimulator.jsx
│   │   ├── ReservationCard.jsx
│   │   ├── SpotCard.jsx
│   │   └── SpotDetailModal.jsx
│   ├── layout/            # App shell components
│   │   ├── BottomNav.jsx
│   │   ├── ErrorBoundary.jsx
│   │   └── Navbar.jsx
│   └── ui/                # Reusable UI primitives
│       ├── Button.jsx
│       ├── ErrorBanner.jsx
│       ├── GlassCard.jsx
│       ├── Input.jsx
│       ├── LoadingSpinner.jsx
│       └── StatusBadge.jsx
├── contexts/
│   ├── AuthContext.jsx    # JWT auth state, login/logout, expiry check
│   └── ReservationContext.jsx  # Active reservation tracking
├── pages/
│   ├── LoginPage.jsx
│   ├── DashboardPage.jsx
│   ├── ReservePage.jsx
│   ├── FloorMapPage.jsx
│   ├── ActiveReservationPage.jsx
│   ├── CheckoutPage.jsx
│   ├── PaymentPage.jsx
│   └── StatusPage.jsx
├── App.jsx                # Router + providers + route definitions
└── main.jsx               # Entry point with ErrorBoundary + StrictMode
```

## Page Routing

| Path | Page | Auth Required |
|------|------|:---:|
| `/` | LoginPage | No |
| `/dashboard` | DashboardPage | Yes |
| `/reserve` | ReservePage | Yes |
| `/floors/:floor` | FloorMapPage | Yes |
| `/my-spot` | ActiveReservationPage | Yes |
| `/reservation/:id` | ActiveReservationPage | Yes |
| `/payment/:id` | PaymentPage | Yes |
| `/checkout/:id` | CheckoutPage | Yes |
| `/status` | StatusPage | Yes |

All authenticated routes are wrapped in `ProtectedRoute`, which redirects to `/` if the user has no valid token. The `AppLayout` component provides the shared Navbar and BottomNav shell.

## API Client Layer

`src/api/client.js` provides a centralized HTTP client:

- Reads `VITE_API_BASE_URL` for the backend target
- Automatically attaches JWT from `localStorage` (`pp_token`) to all requests
- On 401 responses, clears all `pp_*` localStorage keys and redirects to login
- Exposes named methods grouped by domain: health, search, reservation, presence, payment

Usage:

```javascript
import { api } from '../api/client';

const spots = await api.getFloorMap(1);
const reservation = await api.createReservation({ spot_id, driver_id, vehicle_type });
```

## Build & Deploy

### Docker production build

The frontend uses a two-step deployment:

1. **Build** — Run `npm run build` locally or in CI to produce static assets in `dist/`
2. **Containerize** — The Dockerfile copies `dist/` into an nginx:alpine image

```dockerfile
FROM nginx:alpine
COPY dist/ /usr/share/nginx/html/
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

### Nginx configuration

- SPA fallback: all routes serve `index.html` (no-cache headers for immediate deploy propagation)
- `/api/` and `/health` proxied to the gateway service on the Docker network
- `/assets/` served with 1-year immutable cache headers (Vite hashes filenames)

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_BASE_URL` | `""` (empty, uses relative paths) | Backend API base URL |
| `VITE_API_TARGET` | `https://staging-parkir-pintar.piresc.dev` | Vite dev server proxy target |
| `VITE_JWT_SECRET` | — | JWT signing secret (demo auth) |
| `VITE_JWT_ISSUER` | `parkir-pintar` | JWT issuer claim |
| `VITE_JWT_EXPIRATION_MINUTES` | `60` | Token lifetime |

Copy the example file to get started:

```bash
cp frontend/.env.example frontend/.env
```

In production, `VITE_API_BASE_URL` is left empty — nginx handles API proxying.

## Development Workflow

### Start dev server

```bash
cd frontend
npm install
npm run dev
```

Vite serves on `http://localhost:5173` with HMR. API requests to `/api/*` and `/health` are proxied to the backend (configurable via `VITE_API_TARGET`).

### Production build

```bash
npm run build
```

Outputs optimized static files to `frontend/dist/`.

### Preview production build locally

```bash
npm run preview
```

Serves the built `dist/` folder locally to verify the production build.

### Full Docker build

```bash
cd frontend
npm run build
docker build -t parkir-pintar-frontend .
docker run -p 3000:80 parkir-pintar-frontend
```

---

*Owner: ParkirPintar Engineering*
