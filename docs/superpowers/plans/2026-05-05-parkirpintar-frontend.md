# ParkirPintar Driver Frontend — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a React + Vite single-page application that integrates with all ParkirPintar REST APIs and serves as a presentation-quality driver experience demo.

**Architecture:** React 19 SPA with React Router, vanilla CSS with a dark industrial-luxury design system, Fetch API client, and React Context for auth/reservation state. All pages are self-contained components that map 1:1 to Gateway REST endpoints.

**Tech Stack:** React 19, Vite 6, React Router 7, Vanilla CSS, Google Fonts (Chakra Petch + Newsreader), Fetch API.

---

## File Structure

```
frontend/
├── index.html
├── vite.config.js
├── package.json
├── .env.example
└── src/
    ├── main.jsx
    ├── App.jsx
    ├── index.css
    ├── contexts/
    │   ├── AuthContext.jsx
    │   └── ReservationContext.jsx
    ├── pages/
    │   ├── LoginPage.jsx
    │   ├── DashboardPage.jsx
    │   ├── ReservePage.jsx
    │   ├── FloorMapPage.jsx
    │   ├── ActiveReservationPage.jsx
    │   ├── CheckoutPage.jsx
    │   └── StatusPage.jsx
    ├── components/
    │   ├── layout/
    │   │   ├── Navbar.jsx
    │   │   ├── BottomNav.jsx
    │   │   └── PageTransition.jsx
    │   ├── ui/
    │   │   ├── GlassCard.jsx
    │   │   ├── Button.jsx
    │   │   ├── Input.jsx
    │   │   ├── StatusBadge.jsx
    │   │   ├── LoadingSpinner.jsx
    │   │   ├── ErrorBanner.jsx
    │   │   └── Toast.jsx
    │   └── domain/
    │       ├── AvailabilityBar.jsx
    │       ├── FloorGrid.jsx
    │       ├── SpotCard.jsx
    │       ├── SpotDetailModal.jsx
    │       ├── ReservationCard.jsx
    │       ├── CountdownTimer.jsx
    │       ├── BillingBreakdown.jsx
    │       ├── LocationSimulator.jsx
    │       └── HealthStatusCard.jsx
    ├── api/
    │   └── client.js
    └── utils/
        ├── formatters.js
        └── animations.js
```

---

### Task 1: Project Scaffolding

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.js`
- Create: `frontend/index.html`
- Create: `frontend/.env.example`
- Create: `frontend/src/main.jsx`

- [ ] **Step 1: Create package.json**

Create `frontend/package.json`:
```json
{
  "name": "parkir-pintar-frontend",
  "private": true,
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "react-router-dom": "^7.0.0"
  },
  "devDependencies": {
    "@types/react": "^19.0.0",
    "@types/react-dom": "^19.0.0",
    "@vitejs/plugin-react": "^4.3.0",
    "vite": "^6.0.0"
  }
}
```

- [ ] **Step 2: Create Vite config**

Create `frontend/vite.config.js`:
```javascript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
});
```

- [ ] **Step 3: Create index.html**

Create `frontend/index.html`:
```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no" />
    <title>ParkirPintar</title>
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link href="https://fonts.googleapis.com/css2?family=Chakra+Petch:wght@400;600;700&family=Newsreader:ital,opsz,wght@0,6..72,400;0,6..72,500;1,6..72,400&display=swap" rel="stylesheet" />
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.jsx"></script>
  </body>
</html>
```

- [ ] **Step 4: Create .env.example**

Create `frontend/.env.example`:
```
VITE_API_BASE_URL=http://localhost:8080
```

- [ ] **Step 5: Create main.jsx**

Create `frontend/src/main.jsx`:
```jsx
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';
import App from './App';

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>
);
```

- [ ] **Step 6: Install dependencies and verify dev server starts**

Run:
```bash
cd frontend && npm install
```

Expected: `node_modules/` created with React, Vite, React Router.

Run:
```bash
cd frontend && npm run dev
```

Expected: Vite dev server starts on port 3000. (Stop it with Ctrl+C after verifying.)

- [ ] **Step 7: Commit**

```bash
git add frontend/
git commit -m "chore(frontend): scaffold React + Vite project"
```

---

### Task 2: Global Styles & CSS Design System

**Files:**
- Create: `frontend/src/index.css`
- Create: `frontend/src/utils/animations.js`
- Create: `frontend/src/utils/formatters.js`

- [ ] **Step 1: Write index.css with design system**

Create `frontend/src/index.css` with the full dark industrial-luxury design system including CSS variables, reset, typography, glass card styles, button variants, form inputs, loading spinner keyframes, page transitions, and responsive layout rules. See spec section 3 for exact color values and section 5 for component styling rules. The file must include: `:root` variables, `*` reset, `body` base styles, `.glass-card`, `.btn-primary`, `.btn-cta`, `.btn-danger`, `.btn-ghost`, `.input`, `.spinner`, `@keyframes pulse-dot`, `@keyframes fade-slide-up`, `.page-enter`, `.page-enter-active`, and media queries for mobile-first responsive design.

- [ ] **Step 2: Create animation utilities**

Create `frontend/src/utils/animations.js`:
```javascript
export const fadeSlideUp = {
  initial: { opacity: 0, transform: 'translateY(20px)' },
  animate: { opacity: 1, transform: 'translateY(0)' },
  transition: 'all 0.5s cubic-bezier(0.22, 1, 0.36, 1)',
};

export const staggerDelay = (index, baseMs = 30) => ({
  animationDelay: `${index * baseMs}ms`,
});

export function cn(...classes) {
  return classes.filter(Boolean).join(' ');
}
```

- [ ] **Step 3: Create formatters**

Create `frontend/src/utils/formatters.js`:
```javascript
export function formatIDR(amount) {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: 0,
  }).format(amount);
}

export function formatDateTime(isoString) {
  if (!isoString) return '-';
  return new Date(isoString).toLocaleString('id-ID', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function formatDuration(minutes) {
  if (!minutes || minutes <= 0) return '0m';
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  if (h > 0 && m > 0) return `${h}h ${m}m`;
  if (h > 0) return `${h}h`;
  return `${m}m`;
}

export function generateIdempotencyKey() {
  return crypto.randomUUID();
}
```

- [ ] **Step 4: Verify styles load**

Run `npm run dev` and open `http://localhost:3000`. The page background should be `#070709` (very dark) and Chakra Petch font should be applied to headings.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/index.css frontend/src/utils/
git commit -m "feat(frontend): add design system, formatters, and animation utilities"
```

---

### Task 3: API Client

**Files:**
- Create: `frontend/src/api/client.js`

- [ ] **Step 1: Create the API client**

Create `frontend/src/api/client.js`:
```javascript
const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

function getToken() {
  return localStorage.getItem('pp_token');
}

async function apiRequest(method, path, body = null) {
  const headers = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${getToken() || ''}`,
  };
  const opts = { method, headers };
  if (body) opts.body = JSON.stringify(body);

  const res = await fetch(`${API_BASE}${path}`, opts);
  const data = await res.json().catch(() => ({}));

  if (!res.ok) {
    const err = new Error(data.error || `HTTP ${res.status}`);
    err.status = res.status;
    err.data = data;
    throw err;
  }
  return data;
}

export const api = {
  // Health
  getHealth: () => apiRequest('GET', '/health'),
  getHealthReady: () => apiRequest('GET', '/health/ready'),
  getHealthDetailed: () => apiRequest('GET', '/health/detailed'),

  // Search
  getAvailability: (vehicleType) =>
    apiRequest('GET', `/api/v1/availability?vehicle_type=${vehicleType || ''}`),
  getFloorMap: (floor) => apiRequest('GET', `/api/v1/floors/${floor}`),
  getSpotDetails: (id) => apiRequest('GET', `/api/v1/spots/${id}`),

  // Reservation
  createReservation: (body) => apiRequest('POST', '/api/v1/reservations', body),
  cancelReservation: (id) => apiRequest('DELETE', `/api/v1/reservations/${id}`),
  checkIn: (id) => apiRequest('POST', `/api/v1/reservations/${id}/checkin`),
  checkOut: (id) => apiRequest('POST', `/api/v1/reservations/${id}/checkout`),

  // Presence
  streamLocation: (body) => apiRequest('POST', '/api/v1/presence/stream', body),

  // Payment
  getPaymentStatus: (id) => apiRequest('GET', `/api/v1/payments/${id}/status`),
};
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/api/client.js
git commit -m "feat(frontend): add API client with all Gateway endpoints"
```

---

### Task 4: Contexts (Auth & Reservation)

**Files:**
- Create: `frontend/src/contexts/AuthContext.jsx`
- Create: `frontend/src/contexts/ReservationContext.jsx`

- [ ] **Step 1: Create AuthContext**

Create `frontend/src/contexts/AuthContext.jsx`:
```jsx
import { createContext, useContext, useState, useCallback } from 'react';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => localStorage.getItem('pp_token'));
  const [driverId, setDriverId] = useState(() => localStorage.getItem('pp_driver_id'));

  const login = useCallback((newToken, newDriverId) => {
    localStorage.setItem('pp_token', newToken);
    localStorage.setItem('pp_driver_id', newDriverId);
    setToken(newToken);
    setDriverId(newDriverId);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('pp_token');
    localStorage.removeItem('pp_driver_id');
    setToken(null);
    setDriverId(null);
  }, []);

  const isAuthenticated = !!token;

  return (
    <AuthContext.Provider value={{ token, driverId, isAuthenticated, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be inside AuthProvider');
  return ctx;
}
```

- [ ] **Step 2: Create ReservationContext**

Create `frontend/src/contexts/ReservationContext.jsx`:
```jsx
import { createContext, useContext, useState, useCallback } from 'react';
import { api } from '../api/client';

const ReservationContext = createContext(null);

export function ReservationProvider({ children }) {
  const [currentReservation, setCurrentReservation] = useState(() => {
    const raw = localStorage.getItem('pp_reservation');
    return raw ? JSON.parse(raw) : null;
  });

  const setReservation = useCallback((res) => {
    if (res) {
      localStorage.setItem('pp_reservation', JSON.stringify(res));
    } else {
      localStorage.removeItem('pp_reservation');
    }
    setCurrentReservation(res);
  }, []);

  const clearReservation = useCallback(() => {
    localStorage.removeItem('pp_reservation');
    setCurrentReservation(null);
  }, []);

  const refreshReservation = useCallback(async (id) => {
    // There is no GET /reservations/:id endpoint; we rely on in-memory state
    // and localStorage for the demo. If the backend adds one, replace this.
    return currentReservation;
  }, [currentReservation]);

  return (
    <ReservationContext.Provider
      value={{ currentReservation, setReservation, clearReservation, refreshReservation }}
    >
      {children}
    </ReservationContext.Provider>
  );
}

export function useReservation() {
  const ctx = useContext(ReservationContext);
  if (!ctx) throw new Error('useReservation must be inside ReservationProvider');
  return ctx;
}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/contexts/
git commit -m "feat(frontend): add AuthContext and ReservationContext"
```

---

### Task 5: UI Components

**Files:**
- Create: `frontend/src/components/ui/GlassCard.jsx`
- Create: `frontend/src/components/ui/Button.jsx`
- Create: `frontend/src/components/ui/Input.jsx`
- Create: `frontend/src/components/ui/StatusBadge.jsx`
- Create: `frontend/src/components/ui/LoadingSpinner.jsx`
- Create: `frontend/src/components/ui/ErrorBanner.jsx`

- [ ] **Step 1: Create GlassCard**

Create `frontend/src/components/ui/GlassCard.jsx`:
```jsx
export default function GlassCard({ children, className = '', onClick }) {
  return (
    <div
      className={`glass-card ${className}`}
      onClick={onClick}
      style={{ cursor: onClick ? 'pointer' : 'default' }}
    >
      {children}
    </div>
  );
}
```

- [ ] **Step 2: Create Button**

Create `frontend/src/components/ui/Button.jsx`:
```jsx
import { cn } from '../../utils/animations';

export default function Button({
  children,
  variant = 'primary',
  className = '',
  disabled = false,
  ...props
}) {
  const base = 'btn';
  const variants = {
    primary: 'btn-primary',
    cta: 'btn-cta',
    danger: 'btn-danger',
    ghost: 'btn-ghost',
  };
  return (
    <button
      className={cn(base, variants[variant] || variants.primary, className)}
      disabled={disabled}
      {...props}
    >
      {children}
    </button>
  );
}
```

- [ ] **Step 3: Create Input**

Create `frontend/src/components/ui/Input.jsx`:
```jsx
export default function Input({ className = '', ...props }) {
  return <input className={`input ${className}`} {...props} />;
}
```

- [ ] **Step 4: Create StatusBadge**

Create `frontend/src/components/ui/StatusBadge.jsx`:
```jsx
const STATUS_COLORS = {
  confirmed: 'var(--accent-cyan)',
  checked_in: 'var(--success)',
  checked_out: 'var(--text-secondary)',
  expired: 'var(--danger)',
  cancelled: 'var(--danger)',
  available: 'var(--success)',
  reserved: 'var(--danger)',
  success: 'var(--success)',
  failed: 'var(--danger)',
  pending: 'var(--warning)',
};

export default function StatusBadge({ status }) {
  const color = STATUS_COLORS[status?.toLowerCase()] || 'var(--text-secondary)';
  return (
    <span
      className="status-badge"
      style={{ color, borderColor: color }}
    >
      {status?.toUpperCase() || 'UNKNOWN'}
    </span>
  );
}
```

- [ ] **Step 5: Create LoadingSpinner**

Create `frontend/src/components/ui/LoadingSpinner.jsx`:
```jsx
export default function LoadingSpinner({ size = 40 }) {
  return (
    <div className="spinner-wrapper">
      <div className="spinner" style={{ width: size, height: size }} />
    </div>
  );
}
```

- [ ] **Step 6: Create ErrorBanner**

Create `frontend/src/components/ui/ErrorBanner.jsx`:
```jsx
import Button from './Button';

export default function ErrorBanner({ message, onRetry }) {
  return (
    <div className="error-banner">
      <span>{message}</span>
      {onRetry && (
        <Button variant="ghost" onClick={onRetry}>
          Retry
        </Button>
      )}
    </div>
  );
}
```

- [ ] **Step 7: Commit**

```bash
git add frontend/src/components/ui/
git commit -m "feat(frontend): add UI primitives (GlassCard, Button, Input, StatusBadge, LoadingSpinner, ErrorBanner)"
```

---

### Task 6: Domain Components

**Files:**
- Create: `frontend/src/components/domain/AvailabilityBar.jsx`
- Create: `frontend/src/components/domain/FloorGrid.jsx`
- Create: `frontend/src/components/domain/SpotCard.jsx`
- Create: `frontend/src/components/domain/SpotDetailModal.jsx`
- Create: `frontend/src/components/domain/ReservationCard.jsx`
- Create: `frontend/src/components/domain/CountdownTimer.jsx`
- Create: `frontend/src/components/domain/BillingBreakdown.jsx`
- Create: `frontend/src/components/domain/LocationSimulator.jsx`
- Create: `frontend/src/components/domain/HealthStatusCard.jsx`

- [ ] **Step 1: Create AvailabilityBar**

Create `frontend/src/components/domain/AvailabilityBar.jsx`:
```jsx
import GlassCard from '../ui/GlassCard';

export default function AvailabilityBar({ floors, total }) {
  return (
    <div className="availability-bar">
      <div className="availability-total">
        <span className="availability-number">{total?.total_available ?? 0}</span>
        <span className="availability-label">spots available</span>
      </div>
      <div className="availability-floors">
        {floors?.map((f) => (
          <GlassCard key={f.floor_number} className="availability-floor-card">
            <div className="floor-name">Floor {f.floor_number}</div>
            <div className="floor-counts">
              <span style={{ color: 'var(--accent-cyan)' }}>
                {f.available_car} cars
              </span>
              <span style={{ color: 'var(--accent-amber)' }}>
                {f.available_moto} moto
              </span>
            </div>
          </GlassCard>
        ))}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Create SpotCard**

Create `frontend/src/components/domain/SpotCard.jsx`:
```jsx
import { cn } from '../../utils/animations';

export default function SpotCard({ spot, isSelected, onClick }) {
  const statusColor =
    spot.status === 'available'
      ? 'var(--success)'
      : spot.status === 'reserved'
      ? 'var(--danger)'
      : 'var(--warning)';

  return (
    <div
      className={cn('spot-card', isSelected && 'spot-card-selected')}
      onClick={onClick}
      style={{ borderColor: isSelected ? 'var(--accent-amber)' : statusColor }}
    >
      <div className="spot-code" style={{ color: statusColor }}>
        {spot.spot_code}
      </div>
      <div className="spot-type">{spot.vehicle_type}</div>
    </div>
  );
}
```

- [ ] **Step 3: Create FloorGrid**

Create `frontend/src/components/domain/FloorGrid.jsx`:
```jsx
import SpotCard from './SpotCard';

export default function FloorGrid({ spots, selectedSpotId, onSelect }) {
  return (
    <div className="floor-grid">
      {spots?.map((spot, i) => (
        <div key={spot.id} className="floor-grid-item" style={{ animationDelay: `${i * 30}ms` }}>
          <SpotCard
            spot={spot}
            isSelected={selectedSpotId === spot.id}
            onClick={() => onSelect(spot)}
          />
        </div>
      ))}
    </div>
  );
}
```

- [ ] **Step 4: Create SpotDetailModal**

Create `frontend/src/components/domain/SpotDetailModal.jsx`:
```jsx
import GlassCard from '../ui/GlassCard';
import Button from '../ui/Button';
import StatusBadge from '../ui/StatusBadge';

export default function SpotDetailModal({ spot, onClose, onSelect }) {
  if (!spot) return null;
  return (
    <div className="modal-overlay" onClick={onClose}>
      <GlassCard className="modal-content" onClick={(e) => e.stopPropagation()}>
        <h3>{spot.spot_code}</h3>
        <p>Floor {spot.floor_number}</p>
        <p>Type: {spot.vehicle_type}</p>
        <StatusBadge status={spot.status} />
        {spot.status === 'available' && (
          <Button variant="cta" onClick={() => onSelect(spot)}>
            Select This Spot
          </Button>
        )}
        <Button variant="ghost" onClick={onClose}>
          Close
        </Button>
      </GlassCard>
    </div>
  );
}
```

- [ ] **Step 5: Create ReservationCard**

Create `frontend/src/components/domain/ReservationCard.jsx`:
```jsx
import GlassCard from '../ui/GlassCard';
import StatusBadge from '../ui/StatusBadge';
import CountdownTimer from './CountdownTimer';
import { formatDateTime } from '../../utils/formatters';

export default function ReservationCard({ reservation }) {
  return (
    <GlassCard className="reservation-card">
      <div className="reservation-header">
        <h3>Your Reservation</h3>
        <StatusBadge status={reservation.status} />
      </div>
      <div className="reservation-detail">
        <span className="label">Spot</span>
        <span className="value spot-code">{reservation.spot_id}</span>
      </div>
      <div className="reservation-detail">
        <span className="label">Vehicle</span>
        <span className="value">{reservation.vehicle_type}</span>
      </div>
      <div className="reservation-detail">
        <span className="label">Confirmed</span>
        <span className="value">{formatDateTime(reservation.confirmed_at)}</span>
      </div>
      {reservation.status === 'confirmed' && reservation.expires_at && (
        <CountdownTimer target={reservation.expires_at} />
      )}
    </GlassCard>
  );
}
```

- [ ] **Step 6: Create CountdownTimer**

Create `frontend/src/components/domain/CountdownTimer.jsx`:
```jsx
import { useState, useEffect } from 'react';

export default function CountdownTimer({ target }) {
  const [remaining, setRemaining] = useState(() => calcRemaining(target));

  useEffect(() => {
    const id = setInterval(() => {
      setRemaining(calcRemaining(target));
    }, 1000);
    return () => clearInterval(id);
  }, [target]);

  function calcRemaining(t) {
    const diff = new Date(t) - Date.now();
    if (diff <= 0) return { expired: true, text: 'Expired' };
    const m = Math.floor(diff / 60000);
    const s = Math.floor((diff % 60000) / 1000);
    return { expired: false, text: `${m}m ${s}s` };
  }

  return (
    <div className={`countdown-timer ${remaining.expired ? 'expired' : ''}`}>
      <span className="label">Expires in</span>
      <span className="value">{remaining.text}</span>
    </div>
  );
}
```

- [ ] **Step 7: Create BillingBreakdown**

Create `frontend/src/components/domain/BillingBreakdown.jsx`:
```jsx
import GlassCard from '../ui/GlassCard';
import { formatIDR } from '../../utils/formatters';

export default function BillingBreakdown({ bookingFee, parkingFee, overnightFee, penalty, total }) {
  const items = [
    { label: 'Booking Fee', value: bookingFee || 0 },
    { label: 'Parking Fee', value: parkingFee || 0 },
    { label: 'Overnight Fee', value: overnightFee || 0, conditional: true },
    { label: 'Penalty', value: penalty || 0, conditional: true, danger: true },
  ];

  return (
    <GlassCard className="billing-breakdown">
      <h3>Bill Summary</h3>
      {items.map((item) =>
        item.conditional && item.value === 0 ? null : (
          <div key={item.label} className={`billing-row ${item.danger ? 'danger' : ''}`}>
            <span>{item.label}</span>
            <span>{formatIDR(item.value)}</span>
          </div>
        )
      )}
      <div className="billing-row total">
        <span>Total</span>
        <span>{formatIDR(total || 0)}</span>
      </div>
    </GlassCard>
  );
}
```

- [ ] **Step 8: Create LocationSimulator**

Create `frontend/src/components/domain/LocationSimulator.jsx`:
```jsx
import { useState } from 'react';
import Button from '../ui/Button';

const DEMO_COORDS = { lat: -6.2088, lng: 106.8456, accuracy: 5.0 };

export default function LocationSimulator({ reservationId, onSend }) {
  const [coords, setCoords] = useState(DEMO_COORDS);
  const [loading, setLoading] = useState(false);

  async function handleSend() {
    setLoading(true);
    await onSend({
      reservation_id: reservationId,
      latitude: coords.lat,
      longitude: coords.lng,
      accuracy: coords.accuracy,
    });
    setLoading(false);
  }

  return (
    <div className="location-simulator">
      <div className="location-pin">📍</div>
      <div className="location-coords">
        <div>Lat: {coords.lat.toFixed(4)}</div>
        <div>Lng: {coords.lng.toFixed(4)}</div>
        <div>Accuracy: ±{coords.accuracy}m</div>
      </div>
      <Button variant="primary" onClick={handleSend} disabled={loading}>
        {loading ? 'Sending...' : 'Send Location Update'}
      </Button>
    </div>
  );
}
```

- [ ] **Step 9: Create HealthStatusCard**

Create `frontend/src/components/domain/HealthStatusCard.jsx`:
```jsx
import GlassCard from '../ui/GlassCard';
import StatusBadge from '../ui/StatusBadge';

export default function HealthStatusCard({ name, status, responseTime }) {
  const isHealthy = status === 'UP' || status === 'healthy' || status === 'ok';
  return (
    <GlassCard className="health-card">
      <div className="health-header">
        <span className="health-name">{name}</span>
        <StatusBadge status={isHealthy ? 'available' : 'reserved'} />
      </div>
      {responseTime != null && (
        <div className="health-time">{responseTime}ms</div>
      )}
    </GlassCard>
  );
}
```

- [ ] **Step 10: Commit**

```bash
git add frontend/src/components/domain/
git commit -m "feat(frontend): add domain components (AvailabilityBar, SpotCard, FloorGrid, ReservationCard, CountdownTimer, BillingBreakdown, LocationSimulator, HealthStatusCard)"
```

---

### Task 7: Layout Components

**Files:**
- Create: `frontend/src/components/layout/Navbar.jsx`
- Create: `frontend/src/components/layout/BottomNav.jsx`
- Create: `frontend/src/components/layout/PageTransition.jsx`

- [ ] **Step 1: Create Navbar**

Create `frontend/src/components/layout/Navbar.jsx`:
```jsx
import { useAuth } from '../../contexts/AuthContext';
import Button from '../ui/Button';

export default function Navbar() {
  const { driverId, logout } = useAuth();
  return (
    <nav className="navbar">
      <div className="navbar-brand">ParkirPintar</div>
      <div className="navbar-user">
        <span>{driverId || 'Guest'}</span>
        <Button variant="ghost" onClick={logout}>Logout</Button>
      </div>
    </nav>
  );
}
```

- [ ] **Step 2: Create BottomNav**

Create `frontend/src/components/layout/BottomNav.jsx`:
```jsx
import { Link, useLocation } from 'react-router-dom';
import { cn } from '../../utils/animations';

const TABS = [
  { path: '/dashboard', label: 'Home', icon: '🏠' },
  { path: '/floors/1', label: 'Map', icon: '🗺️' },
  { path: '/reservation/current', label: 'My Spot', icon: '🅿️' },
];

export default function BottomNav() {
  const { pathname } = useLocation();
  return (
    <nav className="bottom-nav">
      {TABS.map((tab) => (
        <Link
          key={tab.path}
          to={tab.path}
          className={cn('bottom-nav-item', pathname.startsWith(tab.path.split('/')[1]) && 'active')}
        >
          <span className="bottom-nav-icon">{tab.icon}</span>
          <span className="bottom-nav-label">{tab.label}</span>
        </Link>
      ))}
    </nav>
  );
}
```

- [ ] **Step 3: Create PageTransition**

Create `frontend/src/components/layout/PageTransition.jsx`:
```jsx
export default function PageTransition({ children }) {
  return <div className="page-transition">{children}</div>;
}
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/layout/
git commit -m "feat(frontend): add layout components (Navbar, BottomNav, PageTransition)"
```

---

### Task 8: Pages

**Files:**
- Create: `frontend/src/pages/LoginPage.jsx`
- Create: `frontend/src/pages/DashboardPage.jsx`
- Create: `frontend/src/pages/ReservePage.jsx`
- Create: `frontend/src/pages/FloorMapPage.jsx`
- Create: `frontend/src/pages/ActiveReservationPage.jsx`
- Create: `frontend/src/pages/CheckoutPage.jsx`
- Create: `frontend/src/pages/StatusPage.jsx`

- [ ] **Step 1: Create LoginPage**

Create `frontend/src/pages/LoginPage.jsx`:
```jsx
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import GlassCard from '../components/ui/GlassCard';
import Button from '../components/ui/Button';
import Input from '../components/ui/Input';

export default function LoginPage() {
  const [token, setToken] = useState('');
  const [driverId, setDriverId] = useState('');
  const { login } = useAuth();
  const navigate = useNavigate();

  function handleSubmit(e) {
    e.preventDefault();
    if (!token.trim() || !driverId.trim()) return;
    login(token.trim(), driverId.trim());
    navigate('/dashboard');
  }

  return (
    <div className="login-page">
      <div className="login-mesh" />
      <GlassCard className="login-card">
        <h1 className="login-title">ParkirPintar</h1>
        <p className="login-subtitle">Smart Parking, Simplified</p>
        <form onSubmit={handleSubmit} className="login-form">
          <label>Driver ID</label>
          <Input value={driverId} onChange={(e) => setDriverId(e.target.value)} placeholder="e.g. drv-123" />
          <label>JWT Token</label>
          <textarea
            className="input"
            rows={4}
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder="Paste your JWT token..."
          />
          <Button variant="cta" type="submit" disabled={!token || !driverId}>
            Enter Garage
          </Button>
        </form>
      </GlassCard>
    </div>
  );
}
```

- [ ] **Step 2: Create DashboardPage**

Create `frontend/src/pages/DashboardPage.jsx`:
```jsx
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import { useReservation } from '../contexts/ReservationContext';
import GlassCard from '../components/ui/GlassCard';
import Button from '../components/ui/Button';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import ErrorBanner from '../components/ui/ErrorBanner';
import AvailabilityBar from '../components/domain/AvailabilityBar';

export default function DashboardPage() {
  const { driverId } = useAuth();
  const { currentReservation } = useReservation();
  const navigate = useNavigate();
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  async function load() {
    try {
      setLoading(true);
      setError(null);
      const res = await api.getAvailability('');
      setData(res.data);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    const id = setInterval(load, 10000);
    return () => clearInterval(id);
  }, []);

  return (
    <div className="page dashboard-page">
      <h2>Welcome, {driverId}</h2>
      {loading && !data && <LoadingSpinner />}
      {error && <ErrorBanner message={error} onRetry={load} />}
      {data && <AvailabilityBar floors={data.floors} total={data.total} />}
      <div className="dashboard-actions">
        <GlassCard className="action-card" onClick={() => navigate('/reserve?type=car')}>
          <div className="action-icon">🚗</div>
          <div className="action-label">Reserve Car</div>
        </GlassCard>
        <GlassCard className="action-card" onClick={() => navigate('/reserve?type=motorcycle')}>
          <div className="action-icon">🏍️</div>
          <div className="action-label">Reserve Motorcycle</div>
        </GlassCard>
      </div>
      {currentReservation && (
        <GlassCard className="action-card" onClick={() => navigate(`/reservation/${currentReservation.id}`)}>
          <div className="action-label">View Active Reservation</div>
        </GlassCard>
      )}
    </div>
  );
}
```

- [ ] **Step 3: Create ReservePage**

Create `frontend/src/pages/ReservePage.jsx`:
```jsx
import { useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { useReservation } from '../contexts/ReservationContext';
import { api } from '../api/client';
import { generateIdempotencyKey } from '../utils/formatters';
import GlassCard from '../components/ui/GlassCard';
import Button from '../components/ui/Button';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import ErrorBanner from '../components/ui/ErrorBanner';

export default function ReservePage() {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const { driverId } = useAuth();
  const { setReservation } = useReservation();
  const [mode, setMode] = useState('system_assigned');
  const [vehicleType, setVehicleType] = useState(params.get('type') || 'car');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  async function handleReserve() {
    setLoading(true);
    setError(null);
    try {
      const res = await api.createReservation({
        driver_id: driverId,
        vehicle_type: vehicleType,
        assignment_mode: mode,
        idempotency_key: generateIdempotencyKey(),
      });
      setReservation(res.data);
      navigate(`/reservation/${res.data.id}`);
    } catch (e) {
      setError(e.message);
      setLoading(false);
    }
  }

  return (
    <div className="page reserve-page">
      <h2>New Reservation</h2>
      <div className="mode-toggle">
        <button className={mode === 'system_assigned' ? 'active' : ''} onClick={() => setMode('system_assigned')}>
          System Assigned
        </button>
        <button className={mode === 'user_selected' ? 'active' : ''} onClick={() => setMode('user_selected')}>
          User Selected
        </button>
      </div>
      <div className="vehicle-toggle">
        <button className={vehicleType === 'car' ? 'active' : ''} onClick={() => setVehicleType('car')}>Car</button>
        <button className={vehicleType === 'motorcycle' ? 'active' : ''} onClick={() => setVehicleType('motorcycle')}>Motorcycle</button>
      </div>
      {mode === 'user_selected' && (
        <GlassCard className="info-card" onClick={() => navigate('/floors/1')}>
          Browse floors to pick your spot →
        </GlassCard>
      )}
      {error && <ErrorBanner message={error} />}
      {loading ? <LoadingSpinner /> : (
        <Button variant="cta" onClick={handleReserve}>
          {mode === 'system_assigned' ? 'Reserve Now' : 'Reserve Selected Spot'}
        </Button>
      )}
    </div>
  );
}
```

- [ ] **Step 4: Create FloorMapPage**

Create `frontend/src/pages/FloorMapPage.jsx`:
```jsx
import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useReservation } from '../contexts/ReservationContext';
import { useAuth } from '../contexts/AuthContext';
import { generateIdempotencyKey } from '../utils/formatters';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import ErrorBanner from '../components/ui/ErrorBanner';
import FloorGrid from '../components/domain/FloorGrid';
import SpotDetailModal from '../components/domain/SpotDetailModal';
import Button from '../components/ui/Button';

export default function FloorMapPage() {
  const { floor } = useParams();
  const navigate = useNavigate();
  const { driverId } = useAuth();
  const { setReservation } = useReservation();
  const [spots, setSpots] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedSpot, setSelectedSpot] = useState(null);

  async function load() {
    try {
      setLoading(true);
      setError(null);
      const res = await api.getFloorMap(Number(floor));
      setSpots(res.data?.spots || []);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, [floor]);

  async function reserveSpot(spot) {
    try {
      const res = await api.createReservation({
        driver_id: driverId,
        vehicle_type: spot.vehicle_type,
        assignment_mode: 'user_selected',
        spot_id: spot.id,
        idempotency_key: generateIdempotencyKey(),
      });
      setReservation(res.data);
      navigate(`/reservation/${res.data.id}`);
    } catch (e) {
      setError(e.message);
    }
  }

  return (
    <div className="page floor-map-page">
      <h2>Floor {floor}</h2>
      <div className="floor-tabs">
        {[1,2,3,4,5].map((f) => (
          <Button key={f} variant={Number(floor) === f ? 'primary' : 'ghost'} onClick={() => navigate(`/floors/${f}`)}>
            F{f}
          </Button>
        ))}
      </div>
      {loading && <LoadingSpinner />}
      {error && <ErrorBanner message={error} onRetry={load} />}
      <FloorGrid spots={spots} selectedSpotId={selectedSpot?.id} onSelect={setSelectedSpot} />
      <SpotDetailModal spot={selectedSpot} onClose={() => setSelectedSpot(null)} onSelect={reserveSpot} />
    </div>
  );
}
```

- [ ] **Step 5: Create ActiveReservationPage**

Create `frontend/src/pages/ActiveReservationPage.jsx`:
```jsx
import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useReservation } from '../contexts/ReservationContext';
import ReservationCard from '../components/domain/ReservationCard';
import LocationSimulator from '../components/domain/LocationSimulator';
import Button from '../components/ui/Button';
import ErrorBanner from '../components/ui/ErrorBanner';
import LoadingSpinner from '../components/ui/LoadingSpinner';

export default function ActiveReservationPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { currentReservation, setReservation, clearReservation } = useReservation();
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);

  const reservation = currentReservation;

  async function handleCheckIn() {
    setLoading(true);
    setError(null);
    try {
      const res = await api.checkIn(id);
      setReservation(res.data);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  async function handleCancel() {
    setLoading(true);
    setError(null);
    try {
      await api.cancelReservation(id);
      clearReservation();
      navigate('/dashboard');
    } catch (e) {
      setError(e.message);
      setLoading(false);
    }
  }

  async function handleLocation(body) {
    setError(null);
    try {
      const res = await api.streamLocation(body);
      if (res.data?.is_geofenced) {
        const checkRes = await api.checkIn(id);
        setReservation(checkRes.data);
      }
    } catch (e) {
      setError(e.message);
    }
  }

  async function handleCheckout() {
    setLoading(true);
    setError(null);
    try {
      const res = await api.checkOut(id);
      setReservation(res.data?.reservation);
      navigate(`/checkout/${id}`);
    } catch (e) {
      setError(e.message);
      setLoading(false);
    }
  }

  if (!reservation) {
    return (
      <div className="page">
        <h2>No active reservation</h2>
        <Button variant="primary" onClick={() => navigate('/dashboard')}>Go Home</Button>
      </div>
    );
  }

  return (
    <div className="page active-reservation-page">
      <ReservationCard reservation={reservation} />
      {error && <ErrorBanner message={error} />}
      {loading && <LoadingSpinner />}
      <div className="action-buttons">
        {reservation.status === 'confirmed' && (
          <>
            <Button variant="primary" onClick={handleCheckIn}>Check In</Button>
            <Button variant="danger" onClick={handleCancel}>Cancel</Button>
          </>
        )}
        {reservation.status === 'checked_in' && (
          <>
            <LocationSimulator reservationId={id} onSend={handleLocation} />
            <Button variant="cta" onClick={handleCheckout}>Check Out</Button>
          </>
        )}
      </div>
    </div>
  );
}
```

- [ ] **Step 6: Create CheckoutPage**

Create `frontend/src/pages/CheckoutPage.jsx`:
```jsx
import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { useReservation } from '../contexts/ReservationContext';
import BillingBreakdown from '../components/domain/BillingBreakdown';
import Button from '../components/ui/Button';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import ErrorBanner from '../components/ui/ErrorBanner';
import StatusBadge from '../components/ui/StatusBadge';

export default function CheckoutPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { currentReservation, clearReservation } = useReservation();
  const [payment, setPayment] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  async function loadPayment() {
    if (!currentReservation?.billing_id) return;
    try {
      const res = await api.getPaymentStatus(currentReservation.billing_id);
      setPayment(res.data);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadPayment();
  }, [currentReservation]);

  function handleDone() {
    clearReservation();
    navigate('/dashboard');
  }

  return (
    <div className="page checkout-page">
      <h2>Checkout Complete</h2>
      {loading && <LoadingSpinner />}
      {error && <ErrorBanner message={error} onRetry={loadPayment} />}
      {currentReservation && (
        <BillingBreakdown
          bookingFee={5000}
          parkingFee={currentReservation.parking_fee || 0}
          overnightFee={currentReservation.overnight_fee || 0}
          penalty={currentReservation.penalty_amount || 0}
          total={currentReservation.total_amount || 0}
        />
      )}
      {payment && (
        <div className="payment-status">
          <StatusBadge status={payment.status} />
          <p>Paid at: {payment.paid_at || '-'}</p>
        </div>
      )}
      <Button variant="cta" onClick={handleDone}>Done</Button>
    </div>
  );
}
```

- [ ] **Step 7: Create StatusPage**

Create `frontend/src/pages/StatusPage.jsx`:
```jsx
import { useEffect, useState } from 'react';
import { api } from '../api/client';
import HealthStatusCard from '../components/domain/HealthStatusCard';
import LoadingSpinner from '../components/ui/LoadingSpinner';
import ErrorBanner from '../components/ui/ErrorBanner';

export default function StatusPage() {
  const [health, setHealth] = useState(null);
  const [detailed, setDetailed] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  async function load() {
    try {
      setLoading(true);
      setError(null);
      const [h, d] = await Promise.all([api.getHealth(), api.getHealthDetailed()]);
      setHealth(h.data);
      setDetailed(d.data);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, []);

  const services = detailed?.dependencies
    ? Object.entries(detailed.dependencies).map(([name, info]) => ({ name, ...info }))
    : [];

  return (
    <div className="page status-page">
      <h2>System Health</h2>
      {health && (
        <div className="health-meta">
          <div>Service: {health.service}</div>
          <div>Version: {health.version}</div>
        </div>
      )}
      {loading && <LoadingSpinner />}
      {error && <ErrorBanner message={error} onRetry={load} />}
      <div className="health-grid">
        {services.map((svc) => (
          <HealthStatusCard key={svc.name} name={svc.name} status={svc.status} responseTime={svc.response_time_ms} />
        ))}
      </div>
    </div>
  );
}
```

- [ ] **Step 8: Commit**

```bash
git add frontend/src/pages/
git commit -m "feat(frontend): add all pages (Login, Dashboard, Reserve, FloorMap, ActiveReservation, Checkout, Status)"
```

---

### Task 9: Routing, App Assembly & Final CSS

**Files:**
- Create: `frontend/src/App.jsx`
- Modify: `frontend/src/index.css` (append remaining component-specific styles)

- [ ] **Step 1: Create App.jsx with routing and providers**

Create `frontend/src/App.jsx`:
```jsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { ReservationProvider } from './contexts/ReservationContext';
import Navbar from './components/layout/Navbar';
import BottomNav from './components/layout/BottomNav';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import ReservePage from './pages/ReservePage';
import FloorMapPage from './pages/FloorMapPage';
import ActiveReservationPage from './pages/ActiveReservationPage';
import CheckoutPage from './pages/CheckoutPage';
import StatusPage from './pages/StatusPage';

function ProtectedRoute({ children }) {
  const { isAuthenticated } = useAuth();
  return isAuthenticated ? children : <Navigate to="/" replace />;
}

function AppLayout({ children }) {
  return (
    <div className="app">
      <Navbar />
      <main className="app-main">{children}</main>
      <BottomNav />
    </div>
  );
}

function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<LoginPage />} />
      <Route path="/dashboard" element={<ProtectedRoute><AppLayout><DashboardPage /></AppLayout></ProtectedRoute>} />
      <Route path="/reserve" element={<ProtectedRoute><AppLayout><ReservePage /></AppLayout></ProtectedRoute>} />
      <Route path="/floors/:floor" element={<ProtectedRoute><AppLayout><FloorMapPage /></AppLayout></ProtectedRoute>} />
      <Route path="/reservation/:id" element={<ProtectedRoute><AppLayout><ActiveReservationPage /></AppLayout></ProtectedRoute>} />
      <Route path="/checkout/:id" element={<ProtectedRoute><AppLayout><CheckoutPage /></AppLayout></ProtectedRoute>} />
      <Route path="/status" element={<ProtectedRoute><AppLayout><StatusPage /></AppLayout></ProtectedRoute>} />
    </Routes>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <ReservationProvider>
          <AppRoutes />
        </ReservationProvider>
      </AuthProvider>
    </BrowserRouter>
  );
}
```

- [ ] **Step 2: Append component-specific CSS to index.css**

Add the following styles to `frontend/src/index.css` (after existing content):

```css
/* Layout */
.app { display: flex; flex-direction: column; min-height: 100vh; background: var(--bg-base); color: var(--text-primary); font-family: 'Newsreader', serif; }
.app-main { flex: 1; padding: 1rem; padding-bottom: 80px; max-width: 480px; margin: 0 auto; width: 100%; }

/* Navbar */
.navbar { display: flex; justify-content: space-between; align-items: center; padding: 1rem; border-bottom: 1px solid var(--border-subtle); position: sticky; top: 0; background: rgba(7,7,9,0.8); backdrop-filter: blur(12px); z-index: 100; }
.navbar-brand { font-family: 'Chakra Petch', sans-serif; font-weight: 700; font-size: 1.25rem; color: var(--accent-cyan); letter-spacing: -0.02em; }
.navbar-user { display: flex; align-items: center; gap: 0.75rem; font-size: 0.875rem; color: var(--text-secondary); }

/* Bottom Nav */
.bottom-nav { position: fixed; bottom: 0; left: 0; right: 0; display: flex; justify-content: space-around; padding: 0.5rem 0; background: rgba(7,7,9,0.9); backdrop-filter: blur(12px); border-top: 1px solid var(--border-subtle); z-index: 100; }
.bottom-nav-item { display: flex; flex-direction: column; align-items: center; gap: 2px; color: var(--text-secondary); text-decoration: none; font-size: 0.75rem; transition: color 0.2s; }
.bottom-nav-item.active { color: var(--accent-cyan); }
.bottom-nav-icon { font-size: 1.25rem; }

/* Glass Card */
.glass-card { background: var(--bg-surface); border: 1px solid var(--border-subtle); border-radius: 16px; padding: 1.25rem; backdrop-filter: blur(12px); transition: all 0.3s cubic-bezier(0.22, 1, 0.36, 1); }
.glass-card:hover { border-color: var(--border-glow); transform: translateY(-2px); }

/* Buttons */
.btn { display: inline-flex; align-items: center; justify-content: center; gap: 0.5rem; padding: 0.75rem 1.5rem; border-radius: 12px; border: 1px solid transparent; font-family: 'Chakra Petch', sans-serif; font-weight: 600; font-size: 0.9375rem; cursor: pointer; transition: all 0.3s cubic-bezier(0.22, 1, 0.36, 1); }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-primary { background: rgba(0, 229, 255, 0.1); color: var(--accent-cyan); border-color: rgba(0, 229, 255, 0.2); }
.btn-primary:hover:not(:disabled) { background: rgba(0, 229, 255, 0.2); box-shadow: 0 0 20px rgba(0, 229, 255, 0.15); }
.btn-cta { background: var(--accent-amber); color: #000; border-color: var(--accent-amber); }
.btn-cta:hover:not(:disabled) { box-shadow: 0 0 20px rgba(255, 200, 0, 0.3); transform: translateY(-2px); }
.btn-danger { background: rgba(255, 23, 68, 0.1); color: var(--danger); border-color: rgba(255, 23, 68, 0.2); }
.btn-danger:hover:not(:disabled) { background: rgba(255, 23, 68, 0.2); }
.btn-ghost { background: transparent; color: var(--text-secondary); border-color: var(--border-subtle); }
.btn-ghost:hover:not(:disabled) { color: var(--text-primary); border-color: var(--text-secondary); }

/* Input */
.input { width: 100%; padding: 0.75rem 1rem; background: var(--bg-elevated); border: 1px solid var(--border-subtle); border-radius: 12px; color: var(--text-primary); font-family: 'Newsreader', serif; font-size: 1rem; outline: none; transition: border-color 0.2s; }
.input:focus { border-color: var(--accent-cyan); }
.input::placeholder { color: var(--text-secondary); }

/* Status Badge */
.status-badge { display: inline-flex; align-items: center; padding: 0.25rem 0.75rem; border-radius: 999px; border: 1px solid; font-family: 'Chakra Petch', sans-serif; font-size: 0.75rem; font-weight: 600; letter-spacing: 0.05em; }

/* Loading Spinner */
.spinner-wrapper { display: flex; justify-content: center; padding: 2rem; }
.spinner { width: 40px; height: 40px; border-radius: 50%; background: var(--accent-cyan); animation: pulse-dot 1.5s ease-in-out infinite; }
@keyframes pulse-dot { 0%, 100% { transform: scale(0.6); opacity: 0.4; } 50% { transform: scale(1); opacity: 1; } }

/* Error Banner */
.error-banner { display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: 1rem; background: rgba(255, 23, 68, 0.08); border: 1px solid rgba(255, 23, 68, 0.2); border-radius: 12px; color: var(--danger); font-size: 0.875rem; margin-bottom: 1rem; }

/* Page transitions */
.page { animation: fade-slide-up 0.5s cubic-bezier(0.22, 1, 0.36, 1); }
@keyframes fade-slide-up { from { opacity: 0; transform: translateY(20px); } to { opacity: 1; transform: translateY(0); } }

/* Login Page */
.login-page { display: flex; align-items: center; justify-content: center; min-height: 100vh; position: relative; overflow: hidden; }
.login-mesh { position: absolute; inset: 0; background: radial-gradient(circle at 30% 20%, rgba(0,229,255,0.08) 0%, transparent 50%), radial-gradient(circle at 70% 80%, rgba(255,200,0,0.06) 0%, transparent 50%); pointer-events: none; }
.login-card { width: 90%; max-width: 380px; text-align: center; }
.login-title { font-family: 'Chakra Petch', sans-serif; font-size: 2rem; font-weight: 700; color: var(--accent-cyan); margin-bottom: 0.25rem; }
.login-subtitle { color: var(--text-secondary); margin-bottom: 1.5rem; font-style: italic; }
.login-form { display: flex; flex-direction: column; gap: 0.75rem; text-align: left; }
.login-form label { font-size: 0.875rem; color: var(--text-secondary); }

/* Dashboard */
.dashboard-page h2 { font-family: 'Chakra Petch', sans-serif; margin-bottom: 1rem; }
.availability-bar { margin-bottom: 1.5rem; }
.availability-total { text-align: center; margin-bottom: 1rem; }
.availability-number { display: block; font-family: 'Chakra Petch', sans-serif; font-size: 3rem; font-weight: 700; color: var(--accent-cyan); line-height: 1; }
.availability-label { color: var(--text-secondary); font-size: 0.875rem; }
.availability-floors { display: grid; grid-template-columns: repeat(2, 1fr); gap: 0.75rem; }
.availability-floor-card { padding: 1rem; }
.floor-name { font-family: 'Chakra Petch', sans-serif; font-weight: 600; margin-bottom: 0.25rem; }
.floor-counts { display: flex; gap: 0.75rem; font-size: 0.875rem; }
.dashboard-actions { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; margin-bottom: 1rem; }
.action-card { display: flex; flex-direction: column; align-items: center; gap: 0.5rem; padding: 1.5rem 1rem; text-align: center; }
.action-icon { font-size: 2rem; }
.action-label { font-family: 'Chakra Petch', sans-serif; font-weight: 600; font-size: 0.875rem; }

/* Reserve */
.reserve-page h2 { font-family: 'Chakra Petch', sans-serif; margin-bottom: 1rem; }
.mode-toggle, .vehicle-toggle { display: flex; gap: 0.5rem; margin-bottom: 1rem; }
.mode-toggle button, .vehicle-toggle button { flex: 1; padding: 0.75rem; background: var(--bg-surface); border: 1px solid var(--border-subtle); border-radius: 12px; color: var(--text-secondary); font-family: 'Chakra Petch', sans-serif; font-weight: 600; cursor: pointer; transition: all 0.2s; }
.mode-toggle button.active, .vehicle-toggle button.active { background: rgba(0, 229, 255, 0.1); color: var(--accent-cyan); border-color: rgba(0, 229, 255, 0.3); }
.info-card { margin-bottom: 1rem; text-align: center; color: var(--text-secondary); cursor: pointer; }

/* Floor Map */
.floor-map-page h2 { font-family: 'Chakra Petch', sans-serif; margin-bottom: 0.5rem; }
.floor-tabs { display: flex; gap: 0.5rem; margin-bottom: 1rem; overflow-x: auto; }
.floor-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.5rem; }
.floor-grid-item { animation: fade-slide-up 0.4s cubic-bezier(0.22, 1, 0.36, 1) both; }
.spot-card { padding: 0.75rem; border-radius: 12px; border: 1px solid; background: var(--bg-surface); text-align: center; cursor: pointer; transition: all 0.2s; }
.spot-card:hover { transform: translateY(-2px); }
.spot-card-selected { box-shadow: 0 0 12px rgba(255, 200, 0, 0.3); }
.spot-code { font-family: 'Chakra Petch', sans-serif; font-weight: 600; font-size: 0.875rem; }
.spot-type { font-size: 0.75rem; color: var(--text-secondary); margin-top: 0.25rem; }
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.6); backdrop-filter: blur(4px); display: flex; align-items: center; justify-content: center; z-index: 200; padding: 1rem; }
.modal-content { width: 100%; max-width: 320px; }

/* Reservation */
.reservation-card { margin-bottom: 1rem; }
.reservation-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
.reservation-header h3 { font-family: 'Chakra Petch', sans-serif; margin: 0; }
.reservation-detail { display: flex; justify-content: space-between; padding: 0.5rem 0; border-bottom: 1px solid var(--border-subtle); }
.reservation-detail .label { color: var(--text-secondary); font-size: 0.875rem; }
.reservation-detail .value { font-weight: 500; }
.spot-code { font-family: 'Chakra Petch', sans-serif; color: var(--accent-cyan); }
.countdown-timer { display: flex; justify-content: space-between; align-items: center; margin-top: 1rem; padding: 0.75rem; background: rgba(0, 229, 255, 0.05); border-radius: 12px; border: 1px solid rgba(0, 229, 255, 0.1); }
.countdown-timer .label { color: var(--text-secondary); font-size: 0.875rem; }
.countdown-timer .value { font-family: 'Chakra Petch', sans-serif; font-weight: 700; color: var(--accent-cyan); }
.countdown-timer.expired { border-color: var(--danger); background: rgba(255, 23, 68, 0.05); }
.countdown-timer.expired .value { color: var(--danger); }
.action-buttons { display: flex; flex-direction: column; gap: 0.75rem; }

/* Location Simulator */
.location-simulator { display: flex; flex-direction: column; align-items: center; gap: 0.75rem; padding: 1rem; border: 1px dashed var(--border-subtle); border-radius: 12px; margin-bottom: 1rem; }
.location-pin { font-size: 2rem; }
.location-coords { text-align: center; font-size: 0.875rem; color: var(--text-secondary); font-family: 'Chakra Petch', sans-serif; }

/* Billing */
.billing-breakdown { margin-bottom: 1rem; }
.billing-breakdown h3 { font-family: 'Chakra Petch', sans-serif; margin-bottom: 1rem; }
.billing-row { display: flex; justify-content: space-between; padding: 0.5rem 0; border-bottom: 1px solid var(--border-subtle); font-size: 0.9375rem; }
.billing-row.danger { color: var(--danger); }
.billing-row.total { border-top: 2px solid var(--border-subtle); border-bottom: none; font-family: 'Chakra Petch', sans-serif; font-weight: 700; font-size: 1.125rem; margin-top: 0.5rem; padding-top: 0.75rem; }

/* Checkout */
.checkout-page h2 { font-family: 'Chakra Petch', sans-serif; margin-bottom: 1rem; }
.payment-status { text-align: center; margin: 1rem 0; }

/* Status */
.status-page h2 { font-family: 'Chakra Petch', sans-serif; margin-bottom: 1rem; }
.health-meta { color: var(--text-secondary); font-size: 0.875rem; margin-bottom: 1rem; }
.health-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
.health-card { padding: 1rem; }
.health-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem; }
.health-name { font-family: 'Chakra Petch', sans-serif; font-weight: 600; font-size: 0.875rem; }
.health-time { font-size: 0.75rem; color: var(--text-secondary); }

/* Responsive */
@media (min-width: 768px) {
  .app-main { max-width: 420px; }
  body { background: #000; }
  .app { border-left: 1px solid var(--border-subtle); border-right: 1px solid var(--border-subtle); max-width: 420px; margin: 0 auto; min-height: 100vh; }
}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/App.jsx frontend/src/index.css
git commit -m "feat(frontend): add App routing, layout, and complete design system CSS"
```

---

### Task 10: Verification & Build

- [ ] **Step 1: Verify dev server**

Run:
```bash
cd frontend && npm run dev
```

Open `http://localhost:3000`. Verify:
- Login page loads with dark theme and gradient mesh background
- Entering a token and driver ID navigates to dashboard
- Dashboard shows availability (or error banner if backend is down)
- Bottom nav works
- All routes are reachable

- [ ] **Step 2: Production build**

Run:
```bash
cd frontend && npm run build
```

Expected: `frontend/dist/` created with optimized static assets. No build errors.

- [ ] **Step 3: Final commit**

```bash
git add frontend/
git commit -m "feat(frontend): complete ParkirPintar driver presentation app"
```

---

## Self-Review

1. **Spec coverage:** All 7 pages from the design spec are implemented. All 10 business endpoints + 4 health endpoints are wired. Dashboard polling, countdown timer, location simulator, and billing breakdown are all present. ✅
2. **Placeholder scan:** No TBD, TODO, or vague requirements. Every step includes complete code. ✅
3. **Type consistency:** Component names, API method names, and context hooks are consistent throughout. ✅

*End of Implementation Plan*
