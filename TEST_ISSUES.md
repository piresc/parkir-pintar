# ParkirPintar Frontend — Test Results

## Summary

All core user flows tested and working end-to-end.

---

## Bugs Fixed

### BUG-001: Check Out button overlap with Bottom Nav (FIXED ✅)
- **Root cause:** Bottom nav (`position: fixed; z-index: 100`) overlapped the "Check Out" button area. Clicks on the button were intercepted by the Map nav link underneath.
- **Fix:** Added `position: relative; z-index: 200` to `.action-buttons` in `index.css`.

### BUG-002: Dashboard spot count showing 150 instead of 400 (FIXED ✅)
- **Root cause:** `getAvailability()` API defaults to `vehicle_type=car`, returning only 150 car spots. `AvailabilityBar` displayed `total.total_available` directly.
- **Fix:** Compute combined total from floor data (`available_car + available_moto` per floor) in `AvailabilityBar.jsx`.

### BUG-003: Inter-service auth forwarding (FIXED ✅)
- **Root cause:** Gateway forwarded JWT to first service, but inter-service gRPC calls (reservation→billing→payment) didn't propagate the auth metadata.
- **Fix:** Added `clientAuthForwardingInterceptor()` and `clientAuthForwardingStreamInterceptor()` to `pkg/grpcclient/client.go`.

### BUG-004: DB search_path not set for background workers (FIXED ✅)
- **Root cause:** Tables in different schemas (reservation, billing, payment, search) weren't found by services.
- **Fix:** `ALTER ROLE parkir_user SET search_path TO reservation, billing, payment, search, public;`

### BUG-005: ActiveReservationPage loses state on refresh (FIXED ✅)
- **Root cause:** Page relied on in-memory context only. Refreshing or navigating directly to `/reservation/:id` showed "No active reservation".
- **Fix:** Added `useEffect` with `api.getReservation(id)` fallback + loading state. Also added `GetReservation` RPC endpoint to backend.

### BUG-006: "My Spot" nav link broken without context (FIXED ✅)
- **Root cause:** BottomNav checked `currentReservation?.id` which was null after page refresh.
- **Fix:** Added localStorage fallback in `getMySpotPath()` — checks `pp_reservation` for active reservation ID.

---

## Flows Verified Working

| Flow | Status |
|------|--------|
| Login (driver UUID) | ✅ |
| Dashboard (400 spots, floor breakdown) | ✅ |
| Reserve Car — System Assigned | ✅ |
| Reserve Motorcycle — System Assigned | ✅ |
| Reserve Car — User Selected (floor map → spot modal → reserve) | ✅ |
| Pay Booking Fee (QRIS simulation) | ✅ |
| Active Reservation page (countdown timer, status badge) | ✅ |
| Check In | ✅ |
| Location Simulator (lat/lng display) | ✅ |
| Check Out → Checkout page (bill summary) | ✅ |
| Checkout Payment (QRIS) → Done → Dashboard | ✅ |
| Cancel Reservation → Dashboard | ✅ |
| Floor Map — all 5 floors, 80 spots each | ✅ |
| Floor switching (F1-F5 tabs) | ✅ |
| Spot Detail Modal (click spot → modal with details) | ✅ |
| "My Spot" nav — with active reservation | ✅ |
| "My Spot" nav — without reservation (falls back to dashboard) | ✅ |
| "View Active Reservation" card on dashboard | ✅ |
| Logout | ✅ |

---

## Files Modified

### Backend
- `pkg/grpcclient/client.go` — auth forwarding interceptors
- `proto/reservation/v1/reservation_grpc.pb.go` — GetReservation RPC
- `proto/reservation/v1/reservation.pb.go` — GetReservationRequest type alias
- `internal/reservation/handler/handler.go` — GetReservation handler
- `internal/reservation/usecase/usecase.go` — GetReservation interface + impl
- `internal/gateway/handler/handler.go` — GET route + handler

### Frontend
- `frontend/src/index.css` — padding-bottom fix, action-buttons z-index
- `frontend/src/api/client.js` — added `getReservation(id)`
- `frontend/src/pages/ActiveReservationPage.jsx` — API fetch fallback
- `frontend/src/components/layout/BottomNav.jsx` — localStorage fallback for My Spot
- `frontend/src/components/domain/AvailabilityBar.jsx` — combined spot count

---

## Known Limitations (not bugs)

1. **Spot availability not real-time synced** — The `spot_read_model` in search schema always shows all spots as available. The floor map doesn't reflect reserved/occupied status visually. Would need NATS event sync to update read model.
2. **No "active reservations" API** — Dashboard shows "View Active Reservation" only when context is populated. After fresh login, user must know their reservation ID or use "My Spot" (which relies on localStorage).
3. **Reservation expiry** — Timer counts down but no auto-cancel on frontend. Backend may handle expiry via background worker.
