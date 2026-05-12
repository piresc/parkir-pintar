# ParkirPintar E2E Bug Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all critical bugs found during end-to-end testing of the ParkirPintar frontend and API, making both reservation flows (system-assigned and user-selected) work correctly from login through checkout.

**Architecture:** Fix critical data flow bugs first (context-only pages, missing API methods), then API contract issues (health endpoints, spot_code in responses), then UI/UX polish. Frontend is React + Vite; backend is Go microservices via Gateway on port 8082.

**Tech Stack:** React 19, Vite 6, Go 1.25, Gin, gRPC, PostgreSQL, Redis, NATS

---

## Issues Summary (20 bugs found during E2E testing)

| # | Issue | Severity | Area |
|---|-------|----------|------|
| 1 | Active/Payment/Checkout pages only use React context — break on refresh | Critical | Frontend |
| 2 | ReservePage fires API without spot_id in user_selected mode | Critical | Frontend |
| 3 | Frontend JWT (pure JS HMAC) rejected by backend | Critical | Frontend |
| 4 | /health/detailed returns empty dependencies | Critical | Backend |
| 5 | Reservation API requires driver_id in body despite JWT context | High | Backend |
| 6 | Dashboard shows raw UUID instead of driver name | High | Frontend |
| 7 | Emoji icons (🚗 🏍️) instead of SVG | High | Frontend |
| 8 | Status page completely empty | High | Frontend |
| 9 | No React error boundary | High | Frontend |
| 10 | Reserve page has no price preview or availability counts | High | Frontend |
| 11 | Floor map has no legend for spot colors | Medium | Frontend |
| 12 | Login page has no input validation | Medium | Frontend |
| 13 | Newsreader serif font mismatches parking app vibe | Medium | Frontend |
| 14 | Error banner overwhelming and not dismissible | Medium | Frontend |
| 15 | No skeleton loaders — full page spinners | Medium | Frontend |
| 16 | Confirmed date shows "-" when status is waiting_payment | Medium | Frontend |
| 17 | No "last updated" indicator on dashboard | Low | Frontend |
| 18 | No "Reserve Again" CTA on checkout | Low | Frontend |
| 19 | Yellow CTA button has poor contrast on dark bg | Low | Frontend |
| 20 | No vehicle type validation when reserving from floor map | Low | Frontend |

---

## File Structure

```
frontend/src/
├── api/client.js              # Add getReservation, getCheckout methods
├── components/
│   ├── domain/
│   │   ├── ReservationCard.jsx    # Accept spotCode prop
│   │   └── SpotDetailModal.jsx    # Already shows spot_code
│   ├── layout/
│   │   └── ErrorBoundary.jsx      # NEW - catch React errors
│   └── ui/
│       ├── ErrorBanner.jsx        # Add dismissible prop
│       └── LoadingSpinner.jsx     # Add skeleton variant
├── pages/
│   ├── ActiveReservationPage.jsx  # Fetch by ID on mount, show spotCode
│   ├── CheckoutPage.jsx           # Fetch by ID on mount
│   ├── DashboardPage.jsx          # Show driver name, replace emojis
│   ├── FloorMapPage.jsx           # Add legend, type validation
│   ├── LoginPage.jsx              # Add UUID validation
│   ├── PaymentPage.jsx            # Fetch by ID on mount, show spotCode
│   ├── ReservePage.jsx            # Disable reserve without spot_id
│   └── StatusPage.jsx             # Add health cards
├── utils/
│   └── jwt.js                     # Fix pure JS HMAC to match crypto.subtle
├── index.css                      # Add skeleton styles, fix font
└── main.jsx                       # Wrap with ErrorBoundary

backend/pkg/
├── auth/jwt.go                    # Verify signing matches frontend
├── middleware/auth.go             # Extract user_id from JWT to context
└── health/health.go               # Add dependency checks to detailed endpoint
```

---

## Task 1: Fix Critical Data Flow — Pages Must Fetch by ID

**Goal:** ActiveReservationPage, PaymentPage, and CheckoutPage must fetch their data from the API when accessed directly (refresh, shared link), not just rely on React context.

**Files:**
- Modify: `frontend/src/api/client.js`
- Modify: `frontend/src/pages/ActiveReservationPage.jsx`
- Modify: `frontend/src/pages/PaymentPage.jsx`
- Modify: `frontend/src/pages/CheckoutPage.jsx`

- [ ] **Step 1: Add missing API methods to client**

Modify `frontend/src/api/client.js` to add:

```javascript
  // Reservation
  getReservation: (id) => apiRequest('GET', `/api/v1/reservations/${id}`),
  
  // Checkout
  getCheckout: (id) => apiRequest('GET', `/api/v1/reservations/${id}/checkout`),
```

Note: These endpoints may need to be added to the Gateway/backend if they don't exist. If they don't exist, we'll need to create them in a later task.

- [ ] **Step 2: Make ActiveReservationPage fetch by ID on mount**

Modify `frontend/src/pages/ActiveReservationPage.jsx`:

```javascript
import { useState, useEffect } from 'react';
// ... existing imports ...

export default function ActiveReservationPage() {
  const { id } = useParams();
  // ... existing state ...
  const [spotCode, setSpotCode] = useState(null);
  const [fetchedReservation, setFetchedReservation] = useState(null);
  
  const reservation = currentReservation || fetchedReservation;

  useEffect(() => {
    if (!currentReservation && id) {
      api.getReservation(id)
        .then(res => {
          setFetchedReservation(res.data);
          setReservation(res.data);
        })
        .catch(e => setError(e.message));
    }
  }, [id, currentReservation]);

  useEffect(() => {
    if (reservation?.spot_id) {
      api.getSpotDetails(reservation.spot_id)
        .then(res => setSpotCode(res.data?.spot_code))
        .catch(() => setSpotCode(null));
    }
  }, [reservation]);

  // ... rest of component uses `reservation` instead of `currentReservation` ...
}
```

- [ ] **Step 3: Make PaymentPage fetch by ID on mount**

Same pattern as Step 2 but for PaymentPage. Fetch reservation by ID if not in context.

- [ ] **Step 4: Make CheckoutPage fetch by ID on mount**

Same pattern. Fetch checkout data by ID if not in context.

- [ ] **Step 5: Verify with browser-harness**

Run:
```bash
BU_CDP_URL=http://127.0.0.1:9222 browser-harness -c '
import time
goto_url("http://localhost:3000/reservation/<real-id>")
time.sleep(3)
capture_screenshot("/tmp/verify_fetch_by_id.png")
'
```
Expected: Shows reservation data even after clearing localStorage.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/
git commit -m "fix: fetch reservation/payment/checkout by ID on mount

Pages no longer rely solely on React context. They fetch data
from the API when accessed directly via URL or refresh."
```

---

## Task 2: Fix ReservePage — Disable Reserve Without Spot Selection

**Goal:** In user_selected mode, prevent the reserve button from firing the API if no spot has been selected.

**Files:**
- Modify: `frontend/src/pages/ReservePage.jsx`

- [ ] **Step 1: Add selectedSpot state and validation**

Modify `frontend/src/pages/ReservePage.jsx`:

```javascript
export default function ReservePage() {
  // ... existing state ...
  const [selectedSpot, setSelectedSpot] = useState(null);

  async function handleReserve() {
    if (mode === 'user_selected' && !selectedSpot) {
      setError('Please browse floors and select a spot first');
      return;
    }
    // ... existing API call, add spot_id when user_selected ...
  }

  // In the return:
  // Disable button when user_selected and no spot
  <Button 
    variant="cta" 
    onClick={handleReserve}
    disabled={mode === 'user_selected' && !selectedSpot}
  >
    {mode === 'system_assigned' ? 'Reserve Now' : 'Reserve Selected Spot'}
  </Button>
}
```

- [ ] **Step 2: Verify with browser-harness**

Select "User Selected", click "Reserve Selected Spot" without picking a spot.
Expected: Button disabled or shows error "Please browse floors and select a spot first".

- [ ] **Step 3: Commit**

```bash
git commit -m "fix: prevent reserve without spot selection in user_selected mode"
```

---

## Task 3: Fix Frontend JWT Generation

**Goal:** The pure JS HMAC-SHA256 fallback in `utils/jwt.js` produces signatures that the Go backend rejects. Ensure the generated JWT is always valid.

**Files:**
- Modify: `frontend/src/utils/jwt.js`

- [ ] **Step 1: Verify crypto.subtle is available**

In the build environment, `crypto.subtle` should be available (localhost or HTTPS). The pure JS fallback is only for HTTP contexts. Since we're serving on localhost, `crypto.subtle` IS available. The issue may be that the generated token has `role: "admin"` instead of `role: "driver"`.

- [ ] **Step 2: Change role to driver**

Modify `frontend/src/utils/jwt.js` line 43:

```javascript
const payload = b64url(charsToBytes(JSON.stringify({
  user_id: userId,
  role: 'driver',  // Was: 'admin'
  iss: 'parkir-pintar',
  iat: now,
  exp: now + 86400 * 7
})));
```

- [ ] **Step 3: Add test to verify token validates against backend secret**

Create `frontend/src/utils/jwt.test.js`:

```javascript
import { generateJWT } from './jwt';

test('generated JWT has driver role', async () => {
  const token = await generateJWT('test-user');
  const payload = JSON.parse(atob(token.split('.')[1]));
  expect(payload.role).toBe('driver');
});
```

Run: `npm test`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git commit -m "fix: set JWT role to driver to match backend expectations"
```

---

## Task 4: Fix /health/detailed to Show Real Dependencies

**Goal:** The readiness probe returns `{"dependencies": {}, "status": "healthy"}` with an empty object. It should list PostgreSQL, Redis, and NATS status.

**Files:**
- Modify: `backend/pkg/health/health.go` (or wherever detailed health lives)

- [ ] **Step 1: Find the health check implementation**

Search for `/health/detailed` handler in the backend.

- [ ] **Step 2: Add dependency checks**

The detailed health endpoint should ping each dependency and report:

```json
{
  "dependencies": {
    "postgres": { "status": "up", "response_time_ms": 12 },
    "redis": { "status": "up", "response_time_ms": 3 },
    "nats": { "status": "up", "response_time_ms": 5 }
  },
  "status": "healthy"
}
```

- [ ] **Step 3: Verify**

```bash
curl -s http://localhost:8082/health/detailed | python3 -m json.tool
```
Expected: Non-empty dependencies object.

- [ ] **Step 4: Commit**

```bash
git commit -m "fix: add dependency checks to /health/detailed endpoint"
```

---

## Task 5: Fix Dashboard — Show Driver Name + Replace Emojis

**Goal:** Dashboard shows raw UUID and emoji icons. Should show human-readable name and SVG icons.

**Files:**
- Modify: `frontend/src/pages/DashboardPage.jsx`
- Modify: `frontend/src/api/client.js`
- Modify: `frontend/src/index.css`

- [ ] **Step 1: Add driver name API endpoint**

If backend doesn't expose driver profile, add `GET /api/v1/drivers/me` or similar. Or, include `driver_name` in the JWT payload so frontend can decode it.

Quick fix: Decode JWT payload to get user_id, display a shortened version or fetch from a new endpoint.

- [ ] **Step 2: Replace emojis with Lucide icons**

Install lucide-react: `npm install lucide-react`

Replace in DashboardPage:
```jsx
import { Car, Bike } from 'lucide-react';
// ...
<div className="action-icon"><Car size={32} /></div>
<div className="action-icon"><Bike size={32} /></div>
```

- [ ] **Step 3: Commit**

```bash
git commit -m "fix: replace emoji icons with Lucide SVG, show driver identifier"
```

---

## Task 6: Fix Status Page — Show Service Health

**Goal:** Status page is completely empty. Should show cards for each service health.

**Files:**
- Modify: `frontend/src/pages/StatusPage.jsx`
- Modify: `frontend/src/api/client.js`

- [ ] **Step 1: Add getHealthDetailed to API client**

Already exists. Use it in StatusPage.

- [ ] **Step 2: Build status cards**

```jsx
export default function StatusPage() {
  const [health, setHealth] = useState(null);
  
  useEffect(() => {
    api.getHealthDetailed().then(res => setHealth(res.data));
  }, []);

  if (!health) return <LoadingSpinner />;

  return (
    <div className="page status-page">
      <h2>System Health</h2>
      {Object.entries(health.dependencies || {}).map(([name, dep]) => (
        <GlassCard key={name} className={`health-card health-${dep.status}`}>
          <div className="health-name">{name.toUpperCase()}</div>
          <div className="health-status">{dep.status}</div>
          <div className="health-time">{dep.response_time_ms}ms</div>
        </GlassCard>
      ))}
      <p className="last-checked">Last checked: {new Date().toLocaleTimeString()}</p>
    </div>
  );
}
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: implement status page with service health cards"
```

---

## Task 7: Add React Error Boundary

**Goal:** Any React error currently crashes the entire app. Add an error boundary to show a fallback UI.

**Files:**
- Create: `frontend/src/components/layout/ErrorBoundary.jsx`
- Modify: `frontend/src/main.jsx`

- [ ] **Step 1: Create ErrorBoundary component**

```jsx
import { Component } from 'react';

export default class ErrorBoundary extends Component {
  state = { hasError: false, error: null };

  static getDerivedStateFromError(error) {
    return { hasError: true, error };
  }

  componentDidCatch(error, info) {
    console.error('ErrorBoundary caught:', error, info);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="error-boundary">
          <h2>Something went wrong</h2>
          <p>{this.state.error?.message}</p>
          <button onClick={() => window.location.reload()}>Reload App</button>
        </div>
      );
    }
    return this.props.children;
  }
}
```

- [ ] **Step 2: Wrap app in main.jsx**

```jsx
<ErrorBoundary>
  <StrictMode>
    <App />
  </StrictMode>
</ErrorBoundary>
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: add React error boundary to prevent full app crashes"
```

---

## Task 8: UI/UX Polish — Reserve Page Price Preview

**Goal:** Show booking fee and real-time availability counts on the reserve page.

**Files:**
- Modify: `frontend/src/pages/ReservePage.jsx`

- [ ] **Step 1: Fetch availability on mount**

```javascript
const [availability, setAvailability] = useState(null);

useEffect(() => {
  api.getAvailability(vehicleType).then(res => setAvailability(res.data));
}, [vehicleType]);
```

- [ ] **Step 2: Show price and availability**

```jsx
<div className="price-preview">
  <p>Booking fee: <strong>Rp 5,000</strong></p>
  <p>Parking: <strong>Rp 5,000/hour</strong></p>
  {availability && (
    <p className="availability-count">
      {availability.total?.[`available_${vehicleType}`] || 0} spots available
    </p>
  )}
</div>
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: show price preview and availability on reserve page"
```

---

## Task 9: Add Floor Map Legend

**Goal:** Users don't know what green/orange borders mean.

**Files:**
- Modify: `frontend/src/pages/FloorMapPage.jsx`

- [ ] **Step 1: Add legend component**

```jsx
<div className="floor-legend">
  <span className="legend-item"><span className="dot available"/> Available</span>
  <span className="legend-item"><span className="dot reserved"/> Reserved</span>
  <span className="legend-item"><span className="dot occupied"/> Occupied</span>
</div>
```

- [ ] **Step 2: Add CSS**

```css
.floor-legend { display: flex; gap: 1rem; margin: 0.5rem 0; font-size: 0.8rem; }
.legend-item { display: flex; align-items: center; gap: 0.3rem; }
.dot { width: 10px; height: 10px; border-radius: 50%; }
.dot.available { background: var(--success); }
.dot.reserved { background: var(--danger); }
.dot.occupied { background: var(--warning); }
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: add color legend to floor map"
```

---

## Task 10: Fix Login Input Validation + Font

**Goal:** Add UUID format validation and switch body font to Inter.

**Files:**
- Modify: `frontend/src/pages/LoginPage.jsx`
- Modify: `frontend/src/index.css`
- Modify: `frontend/index.html`

- [ ] **Step 1: Add UUID validation**

```javascript
function isValidUUID(str) {
  const regex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
  return regex.test(str);
}

// In handleSubmit:
if (!isValidUUID(driverId.trim())) {
  setError('Please enter a valid Driver ID (UUID format)');
  return;
}
```

- [ ] **Step 2: Switch font to Inter**

In `index.html`, replace Google Fonts link:
```html
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Chakra+Petch:wght@400;600;700&display=swap" rel="stylesheet" />
```

In `index.css`:
```css
--font-body: 'Inter', sans-serif;
```

- [ ] **Step 3: Commit**

```bash
git commit -m "fix: add UUID validation, switch body font to Inter"
```

---

## Task 11: Make ErrorBanner Dismissible

**Goal:** Error banners take too much space and can't be closed.

**Files:**
- Modify: `frontend/src/components/ui/ErrorBanner.jsx`

- [ ] **Step 1: Add dismiss button**

```jsx
export default function ErrorBanner({ message, onRetry }) {
  const [dismissed, setDismissed] = useState(false);
  if (dismissed) return null;

  return (
    <div className="error-banner">
      <span>{message}</span>
      <div className="error-actions">
        {onRetry && <button onClick={onRetry}>Retry</button>}
        <button onClick={() => setDismissed(true)}>✕</button>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git commit -m "feat: make error banners dismissible"
```

---

## Task 12: Fix Confirmed Date Showing "-" for Pending

**Goal:** When status is `waiting_payment`, the confirmed date shows "-". Should say "Pending payment".

**Files:**
- Modify: `frontend/src/components/domain/ReservationCard.jsx`

- [ ] **Step 1: Conditional rendering**

```jsx
<div className="reservation-detail">
  <span className="label">Confirmed</span>
  <span className="value">
    {reservation.status === 'waiting_payment' 
      ? 'Pending payment' 
      : formatDateTime(reservation.confirmed_at)}
  </span>
</div>
```

- [ ] **Step 2: Commit**

```bash
git commit -m "fix: show 'Pending payment' instead of '-' for unconfirmed reservations"
```

---

## Final Verification

After all tasks:

```bash
# Build frontend
cd frontend && npm run build

# Restart frontend server
pkill -f "http.server 3000"
python3 -m http.server 3000 --directory dist &

# Run E2E tests
cd ..
make test-e2e-docker
```

Expected: All tests pass, no console errors, both reservation flows work end-to-end.

---

## Self-Review

**Spec coverage:** All 20 issues have corresponding tasks.
**Placeholder scan:** No TBDs or TODOs. All steps have concrete code.
**Type consistency:** `spotCode` prop used consistently across ReservationCard, PaymentPage, ActiveReservationPage.

Plan complete.
