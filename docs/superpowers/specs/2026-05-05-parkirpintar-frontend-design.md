# ParkirPintar Driver Frontend — Design Specification

> **Date:** 2026-05-05  
> **Status:** Draft — Pending review  
> **Scope:** Driver-facing presentation web application using all available ParkirPintar REST APIs.

---

## 1. Goal

Build a distinctive, presentation-grade React + Vite single-page application that demonstrates the full ParkirPintar driver experience — from login through reservation, check-in, presence streaming, checkout, and payment. It must integrate with **every REST endpoint exposed by the Gateway service** and be visually striking enough to serve as a live demo / presentation asset.

---

## 2. Architecture & Tech Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| Framework | React 19 | Modern, component-based, excellent for interactive demos |
| Build Tool | Vite | Fast HMR, clean output, minimal config |
| Routing | React Router v7 | Client-side navigation between presentation screens |
| Styling | Vanilla CSS + CSS variables | Full creative control per the frontend-design skill mandate |
| HTTP Client | Fetch API | Native, zero deps, sufficient for this scope |
| State | React Context | Auth + Reservation state; no Redux needed |
| Fonts | Chakra Petch (display), Newsreader (body) | Google Fonts CDN |

---

## 3. Aesthetic Direction: "Premium Midnight Garage"

A dark, refined industrial-luxury aesthetic inspired by high-end parking facilities and digital LED signage. The app should feel like a luxury car interface — confident, precise, and unforgettable.

### 3.1 Color System

```css
:root {
  --bg-base: #070709;
  --bg-surface: #12121a;
  --bg-elevated: #1a1a25;
  --accent-cyan: #00e5ff;
  --accent-amber: #ffc800;
  --success: #00e676;
  --danger: #ff1744;
  --warning: #ff9100;
  --text-primary: #f0f0f5;
  --text-secondary: #8a8a9a;
  --border-subtle: rgba(255, 255, 255, 0.08);
  --border-glow: rgba(0, 229, 255, 0.25);
}
```

### 3.2 Typography

- **Display / Headers:** `Chakra Petch`, weight 600–700, letter-spacing `-0.02em`
- **Body / UI Text:** `Newsreader`, weight 400–500, line-height `1.6`
- **Scale:** H1 `2.5rem`, H2 `1.75rem`, H3 `1.25rem`, Body `1rem`, Caption `0.875rem`

### 3.3 Surfaces & Details

- **Cards:** `background: var(--bg-surface)`, `border: 1px solid var(--border-subtle)`, `border-radius: 16px`, `backdrop-filter: blur(12px)`
- **Parking Grid Overlay:** CSS `radial-gradient` dots at 5% opacity on dashboard background
- **Login Background:** Subtle radial gradient mesh (cyan → amber) at 8% opacity
- **Hover States:** `translateY(-4px)`, border glow intensifies to `var(--border-glow)`, `transition: all 0.3s cubic-bezier(0.22, 1, 0.36, 1)`

### 3.4 Motion

- **Page transitions:** Staggered fade + slide-up (`translateY(20px)` → `0`), duration `0.5s`, easing `cubic-bezier(0.22, 1, 0.36, 1)`
- **Spot grid entrance:** Staggered `opacity` + `scale(0.95)` → `scale(1)` with `animation-delay` increments of `30ms` per item
- **CTA hover:** Amber glow shadow `0 0 20px rgba(255, 200, 0, 0.3)`
- **Loading:** Pulsing cyan dot via `keyframes` with `scale` + `opacity` oscillation

---

## 4. Pages & User Flows

Every page maps to one or more Gateway REST endpoints. The flow is designed to tell a story during presentation.

### 4.1 Login Page (`/`)

**Purpose:** Obtain JWT token to authenticate all subsequent requests.

**UI:**
- Centered card on gradient-mesh background
- Inputs: `driver_id` (text), `JWT Token` (textarea / paste)
- Since the backend receives JWT from a super app, the demo accepts a raw token string
- CTA: "Enter Garage" (amber button)

**API:** None (client-side token storage only)

---

### 4.2 Dashboard (`/dashboard`)

**Purpose:** The driver's home screen. See real-time availability at a glance and start a reservation.

**UI:**
- Top nav with driver name + logout
- Large hero section with total available spots (`GET /api/v1/availability`)
- Two prominent CTAs: "Reserve Car" / "Reserve Motorcycle"
- Summary cards: Available Floors, Nearest Spot, Recent Reservation
- Bottom tab nav: Home | Map | My Spot

**API:**
- `GET /api/v1/availability` — loads on mount, auto-refreshes every 10s

---

### 4.3 Reserve Page (`/reserve`)

**Purpose:** Create a new reservation (system-assigned or user-selected).

**UI:**
- Mode toggle: "System Assigned" (fast) vs "User Selected" (browse)
- Vehicle type selector: Car / Motorcycle
- If **System Assigned**: One-tap "Reserve Now" button
- If **User Selected**: Floor picker → loads floor map → tap a spot → confirm
- Confirmation card shows assigned spot code, expiry countdown timer

**API:**
- `POST /api/v1/reservations` — creates reservation
  - `driver_id`, `vehicle_type`, `assignment_mode`, optional `spot_id`, generated `idempotency_key`

---

### 4.4 Floor Map Page (`/floors/:floor`)

**Purpose:** Browse spots by floor for user-selected mode.

**UI:**
- Floor selector tabs (1–5)
- Grid of spot cards showing `spot_code`, `vehicle_type` icon, `status`
- Color coding: green (available), red (reserved/taken), amber (your selection)
- Spot detail modal on tap

**API:**
- `GET /api/v1/floors/:floor` — loads all spots for selected floor
- `GET /api/v1/spots/:id` — loads detailed info for tapped spot

---

### 4.5 Active Reservation Page (`/reservation/:id`)

**Purpose:** Manage an active reservation — view status, check in, cancel, stream location.

**UI:**
- Large status badge: CONFIRMED | CHECKED_IN | EXPIRED | CANCELLED
- Spot code displayed prominently (like a parking ticket)
- Countdown timer until expiry (1h from `confirmed_at`)
- Action buttons:
  - "Check In" (enabled when CONFIRMED)
  - "Cancel Reservation" (enabled when CONFIRMED)
  - "Simulate Arrival" (sends presence stream + triggers check-in)
- Location stream simulator: map pin + "Send Location Update" button
- Real-time status polling every 5s

**API:**
- `POST /api/v1/reservations/:id/checkin` — manual check-in
- `DELETE /api/v1/reservations/:id` — cancel reservation
- `POST /api/v1/presence/stream` — simulate location update
  - Body: `reservation_id`, `latitude`, `longitude`, `accuracy`

---

### 4.6 Checkout Page (`/checkout/:id`)

**Purpose:** Complete the parking session, view billing, and process payment.

**UI:**
- Session summary: check-in time, duration, spot code
- "Check Out Now" CTA
- Post-checkout billing breakdown:
  - Booking Fee: 5,000 IDR
  - Parking Fee: calculated hours × 5,000 IDR
  - Overnight Fee: 20,000 IDR (if applicable)
  - Penalties: 200,000 IDR (if wrong-spot detected)
  - **Total**
- Payment status indicator
- "Payment Successful" celebration animation

**API:**
- `POST /api/v1/reservations/:id/checkout` — triggers checkout + billing + payment
- `GET /api/v1/payments/:id/status` — polls payment status

---

### 4.7 System Health / Demo Status Page (`/status`)

**Purpose:** Present the backend infrastructure health during the demo.

**UI:**
- Cards for each service: Gateway, Search, Reservation, Billing, Payment, Presence
- Status indicators (green = healthy, red = down)
- Response time graphs (simulated or real)

**API:**
- `GET /health` — build info
- `GET /health/live` — liveness
- `GET /health/ready` — readiness
- `GET /health/detailed` — per-dependency status

---

## 5. Component Design

### 5.1 Layout Components

| Component | Responsibility |
|-----------|---------------|
| `Navbar` | Top bar with logo, driver name, logout button |
| `BottomNav` | Tab bar for mobile-like navigation (Home, Map, My Spot) |
| `PageTransition` | Wrapper for entrance/exit animations on route change |
| `GlassCard` | Reusable card with glass-morphism styling |
| `StatusBadge` | Colored pill badge for reservation/payment/health status |

### 5.2 Domain Components

| Component | Responsibility |
|-----------|---------------|
| `AvailabilityBar` | Horizontal bar showing car vs motorcycle availability |
| `FloorGrid` | Grid of `SpotCard`s for a given floor |
| `SpotCard` | Individual parking spot tile with code, type, status |
| `SpotDetailModal` | Modal showing spot details + "Select This Spot" CTA |
| `ReservationCard` | Summary of active reservation with timer |
| `CountdownTimer` | Live countdown to expiry |
| `BillingBreakdown` | Itemized fee display |
| `LocationSimulator` | Map-like visual + coords + "Send Update" button |
| `HealthStatusCard` | Service name + status dot + response time |

### 5.3 Shared Components

| Component | Responsibility |
|-----------|---------------|
| `Button` | Primary (cyan), CTA (amber), Danger (red), Ghost variants |
| `Input` | Text input with dark theme styling |
| `Select` | Styled select dropdown |
| `LoadingSpinner` | Pulsing cyan dot animation |
| `ErrorBanner` | Inline error message with retry button |
| `Toast` | Success / error notification slide-in |

---

## 6. State Management

### 6.1 AuthContext

```javascript
{
  token: string | null,
  driverId: string | null,
  isAuthenticated: boolean,
  login: (token, driverId) => void,
  logout: () => void
}
```

- Token stored in `localStorage` for demo persistence
- All API calls include `Authorization: Bearer ${token}`
- Logout clears storage + state + navigates to `/`

### 6.2 ReservationContext

```javascript
{
  currentReservation: Reservation | null,
  setCurrentReservation: (r) => void,
  clearReservation: () => void,
  refreshReservation: () => Promise<void>
}
```

- Holds the active reservation object
- `refreshReservation` polls the latest status from the backend
- Cleared on checkout, expiry, or cancellation

---

## 7. API Integration

All endpoints target `http://localhost:8080` (configurable via `.env`).

### 7.1 Request Wrapper

```javascript
async function apiRequest(method, path, body = null) {
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  };
  // handle fetch, parse JSON, map errors
}
```

### 7.2 Endpoint Mapping

| Page | Method | Endpoint | Request Body | Response Used |
|------|--------|----------|-------------|---------------|
| Dashboard | GET | `/api/v1/availability` | — | `floors`, `total` |
| Reserve | POST | `/api/v1/reservations` | `driver_id`, `vehicle_type`, `assignment_mode`, `spot_id?`, `idempotency_key` | `reservation` object |
| Floor Map | GET | `/api/v1/floors/:floor` | — | `spots[]` |
| Floor Map | GET | `/api/v1/spots/:id` | — | `id`, `spot_code`, `floor_number`, `vehicle_type`, `status` |
| Active | POST | `/api/v1/reservations/:id/checkin` | — | updated `reservation` |
| Active | DELETE | `/api/v1/reservations/:id` | — | updated `reservation` |
| Active | POST | `/api/v1/presence/stream` | `reservation_id`, `latitude`, `longitude`, `accuracy` | `detected`, `is_geofenced` |
| Checkout | POST | `/api/v1/reservations/:id/checkout` | — | `reservation`, `total_amount`, `billing_id` |
| Checkout | GET | `/api/v1/payments/:id/status` | — | `status`, `amount`, `paid_at` |
| Status | GET | `/health` | — | `service`, `version` |
| Status | GET | `/health/ready` | — | `status` |
| Status | GET | `/health/detailed` | — | `dependencies[]` |

### 7.3 Error Handling

- **4xx errors:** Show inline `ErrorBanner` with backend message
- **5xx errors:** Show full-page error with retry CTA
- **Network errors:** Toast notification + auto-retry once
- **Auth errors (401):** Redirect to login with message

---

## 8. Responsive Design

- **Primary target:** Mobile viewport (375px–428px) — simulates a super-app mini-app
- **Secondary:** Tablet (768px) and desktop (1024px+) — centered phone frame on dark background
- **Desktop enhancement:** Side panel showing API request/response logs in real-time (great for presentations)

---

## 9. File Structure

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
    │   │   ├── Select.jsx
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
        ├── formatters.js       // IDR currency, date/time, spot code
        └── animations.js       // CSS animation helpers
```

---

## 10. Presentation Flow (Demo Script)

For live presentation, the ideal click-through path is:

1. **Login** → Paste demo JWT
2. **Dashboard** → Show real-time availability
3. **Reserve** → Choose "System Assigned" + Car → Tap "Reserve Now"
4. **Active Reservation** → Show spot code + countdown
5. **Simulate Arrival** → Send presence stream → Auto check-in
6. **Checkout** → Tap "Check Out" → Show billing breakdown
7. **Payment Success** → Celebration animation
8. **Status Page** → Show all services healthy

This path exercises **all 10 API endpoints** in a logical narrative.

---

## 11. Open Questions / Notes

- **Driver ID source:** The demo login will accept any `driver_id` string + JWT token. In production, this comes from the super app's auth system.
- **Geolocation:** The presence simulator will use hardcoded coordinates near the parking facility. Real geolocation API is optional.
- **QRIS:** The payment page will show a simulated QRIS code (placeholder image) since the backend payment gateway is stubbed.
- **Idempotency keys:** Generated client-side using `crypto.randomUUID()` on each reservation/create call.

---

*End of Design Specification*
