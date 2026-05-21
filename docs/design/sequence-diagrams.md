# Sequence Diagrams — ParkirPintar

Key interaction flows documented in Mermaid sequence diagram syntax.

---

## 1. Create Reservation Flow

The critical path for booking a parking spot. Uses distributed locking to prevent double-booking.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client (Mobile App)
    participant GW as API Gateway
    participant RS as Reservation Service
    participant RD as Redis (Lock)
    participant DB as PostgreSQL (reservation.*)
    participant BS as Billing Service
    participant NT as NATS JetStream
    participant SS as Search Service

    C->>GW: POST /api/v1/reservations<br/>{spot_id, start, end, idemp_key}
    GW->>GW: Validate JWT token
    GW->>GW: Rate limit check
    GW->>RS: gRPC CreateReservation(req)

    Note over RS: Idempotency check
    RS->>RD: GET idemp:{idemp_key}
    RD-->>RS: nil (not seen before)

    Note over RS: Acquire distributed lock
    RS->>RD: SET lock:spot:{spot_id}:{window} NX EX 10
    alt Lock acquired
        RD-->>RS: OK
    else Lock failed (spot taken)
        RD-->>RS: nil
        RS-->>GW: Error: SPOT_UNAVAILABLE
        GW-->>C: 409 Conflict
    end

    Note over RS: Verify no overlapping reservation
    RS->>DB: SELECT FROM reservations<br/>WHERE spot_id = ? AND time overlaps<br/>AND status NOT IN (cancelled, no_show)
    DB-->>RS: [] (no conflicts)

    Note over RS: Create reservation
    RS->>DB: INSERT INTO reservations<br/>(id, driver_id, spot_id, start, end, status='confirmed')
    DB-->>RS: OK

    Note over RS: Release lock
    RS->>RD: EVAL unlock_script (delete if owner matches)
    RD-->>RS: OK

    Note over RS: Cache idempotency result
    RS->>RD: SET idemp:{idemp_key} {response} EX 86400
    RD-->>RS: OK

    Note over RS: Publish event (async)
    RS--)NT: Publish parkir.reservation.created<br/>{reservation_id, spot_id, driver_id, start, end}

    RS-->>GW: ReservationResponse{id, status=confirmed}
    GW-->>C: 201 Created {reservation}

    Note over NT,SS: Async event processing
    NT--)BS: Deliver to billing consumer
    BS->>BS: Create billing_record<br/>(reservation_id, amount, status=pending)

    NT--)SS: Deliver to search consumer
    SS->>SS: Update spot_read_model<br/>SET is_available=false<br/>WHERE spot_id = ?
    SS->>RD: DEL cache:spot:{spot_id}
```

### Error Scenarios

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant GW as API Gateway
    participant RS as Reservation Service
    participant RD as Redis
    participant DB as PostgreSQL

    Note over C,DB: Scenario: Redis down (graceful degradation)
    C->>GW: POST /api/v1/reservations
    GW->>RS: gRPC CreateReservation(req)
    RS->>RD: SET lock:spot:... NX EX 10
    RD--xRS: Connection refused

    Note over RS: Fallback to DB advisory lock
    RS->>DB: SELECT pg_try_advisory_xact_lock(spot_hash)
    DB-->>RS: true (lock acquired)
    RS->>DB: Check overlap + INSERT
    DB-->>RS: OK
    RS-->>GW: ReservationResponse (degraded mode)
    GW-->>C: 201 Created

    Note over C,DB: Scenario: Duplicate request (idempotency)
    C->>GW: POST /api/v1/reservations (retry, same idemp_key)
    GW->>RS: gRPC CreateReservation(req)
    RS->>RD: GET idemp:{idemp_key}
    RD-->>RS: {cached_response}
    RS-->>GW: Cached ReservationResponse
    GW-->>C: 201 Created (same response as first call)
```

---

## 2. Check-in Flow

Driver arrives at parking spot and checks in via QR code scan.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client (Mobile App)
    participant GW as API Gateway
    participant RS as Reservation Service
    participant PS as Presence Service
    participant DB_P as PostgreSQL (presence.*)
    participant DB_R as PostgreSQL (reservation.*)
    participant BS as Billing Service
    participant NT as NATS JetStream
    participant SS as Search Service

    C->>GW: POST /api/v1/reservations/{id}/check-in<br/>{verification: "qr_code", qr_data: "..."}
    GW->>GW: Validate JWT + authorize (driver owns reservation)
    GW->>RS: gRPC CheckIn(reservation_id, verification)

    Note over RS: Validate reservation state
    RS->>DB_R: SELECT * FROM reservations WHERE id = ?
    DB_R-->>RS: Reservation{status=confirmed, start_time, end_time}

    RS->>RS: Validate: status == confirmed
    RS->>RS: Validate: current_time within check-in window<br/>(start_time - 15min to start_time + 30min)

    Note over RS: Update reservation status
    RS->>DB_R: UPDATE reservations SET status='checked_in' WHERE id = ?
    DB_R-->>RS: OK

    Note over RS: Record presence
    RS->>PS: gRPC RecordPresence(reservation_id, spot_id, CHECK_IN)
    PS->>DB_P: INSERT INTO presence_logs<br/>(reservation_id, spot_id, driver_id, event_type='check_in', event_time=now())
    DB_P-->>PS: OK
    PS-->>RS: OK

    Note over RS: Publish events
    RS--)NT: Publish parkir.reservation.checked_in<br/>{reservation_id, spot_id, driver_id, check_in_time}

    RS-->>GW: CheckInResponse{status=checked_in, check_in_time}
    GW-->>C: 200 OK {reservation updated}

    Note over NT,SS: Async event processing
    NT--)BS: Deliver to billing consumer
    BS->>BS: Update billing_record<br/>SET billing_period_start = check_in_time

    NT--)SS: Deliver to search consumer
    SS->>SS: Confirm spot occupied in read model
    SS->>RD: DEL cache:spot:{spot_id}
```

### Late Check-in / No-Show Detection

```mermaid
sequenceDiagram
    autonumber
    participant CJ as Cron Job (Scheduler)
    participant RS as Reservation Service
    participant DB as PostgreSQL (reservation.*)
    participant BS as Billing Service
    participant NT as NATS JetStream
    participant SS as Search Service

    Note over CJ: Runs every 5 minutes
    CJ->>RS: gRPC ProcessNoShows()

    RS->>DB: SELECT * FROM reservations<br/>WHERE status = 'confirmed'<br/>AND start_time + interval '30 min' < now()
    DB-->>RS: [reservation_1, reservation_2, ...]

    loop For each no-show reservation
        RS->>DB: UPDATE reservations<br/>SET status='no_show' WHERE id = ?
        DB-->>RS: OK

        RS--)NT: Publish parkir.reservation.no_show<br/>{reservation_id, spot_id, driver_id}
    end

    NT--)BS: Deliver no_show events
    BS->>BS: Mark billing as cancelled<br/>(refund booking fee — system failure only)

    NT--)SS: Deliver to search consumer
    SS->>SS: Mark spot as available again
```

---

## 3. Payment Flow

Driver pays for a completed reservation via Payment Gateway payment gateway.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client (Mobile App)
    participant GW as API Gateway
    participant PY as Payment Service
    participant CB as Circuit Breaker
    participant MG as Payment Gateway (stub)
    participant DB_PY as PostgreSQL (payment.*)
    participant BS as Billing Service
    participant DB_B as PostgreSQL (billing.*)
    participant NT as NATS JetStream
    participant RD as Redis

    C->>GW: POST /api/v1/payments<br/>{billing_record_id, method: "ewallet", idemp_key}
    GW->>GW: Validate JWT
    GW->>PY: gRPC ProcessPayment(req)

    Note over PY: Idempotency check
    PY->>RD: GET idemp:pay:{idemp_key}
    RD-->>PY: nil (first attempt)

    Note over PY: Validate billing record
    PY->>BS: gRPC GetBillingRecord(billing_record_id)
    BS->>DB_B: SELECT * FROM billing_records WHERE id = ?
    DB_B-->>BS: BillingRecord{amount, status=pending}
    BS-->>PY: BillingRecord

    PY->>PY: Validate: status == pending, amount > 0

    Note over PY: Create payment record
    PY->>DB_PY: INSERT INTO payments<br/>(id, billing_record_id, amount, status='processing')
    DB_PY-->>PY: OK

    Note over PY: Call external payment gateway (with circuit breaker)
    PY->>CB: Execute(chargeRequest)
    CB->>MG: POST /v2/charge<br/>{payment_type: "ewallet", amount, order_id}

    alt Payment successful
        MG-->>CB: 200 {status: "capture", transaction_id: "..."}
        CB-->>PY: Success

        PY->>DB_PY: UPDATE payments<br/>SET status='success', external_transaction_id=?, paid_at=now()
        DB_PY-->>PY: OK

        Note over PY: Update billing status
        PY->>BS: gRPC MarkBillingPaid(billing_record_id)
        BS->>DB_B: UPDATE billing_records SET status='paid'
        DB_B-->>BS: OK
        BS-->>PY: OK

        Note over PY: Cache idempotency
        PY->>RD: SET idemp:pay:{idemp_key} {response} EX 86400

        Note over PY: Publish event
        PY--)NT: Publish parkir.payment.completed<br/>{payment_id, billing_record_id, amount}

        PY-->>GW: PaymentResponse{status=success, transaction_id}
        GW-->>C: 200 OK {payment confirmed}

    else Payment failed
        MG-->>CB: 400 {status: "deny", message: "insufficient funds"}
        CB-->>PY: Error

        PY->>DB_PY: UPDATE payments SET status='failed'
        DB_PY-->>PY: OK

        PY-->>GW: Error: PAYMENT_FAILED
        GW-->>C: 402 Payment Required {error details}

    else Gateway timeout (circuit breaker opens)
        MG--xCB: Timeout
        CB->>CB: Increment failure count (5/5 threshold)
        CB->>CB: State: CLOSED → OPEN
        CB-->>PY: Error: ErrCircuitOpen

        PY->>DB_PY: UPDATE payments SET status='pending'
        PY--)NT: Publish parkir.payment.pending_retry<br/>{payment_id, retry_after: 30s}

        PY-->>GW: Error: GATEWAY_UNAVAILABLE
        GW-->>C: 503 Service Unavailable {retry_after: 30}
    end
```

### Payment Result (NATS Event)

```mermaid
sequenceDiagram
    autonumber
    participant MG as Payment Gateway (stub)
    participant GW as API Gateway
    participant PY as Payment Service
    participant DB_PY as PostgreSQL (payment.*)
    participant BS as Billing Service
    participant NT as NATS JetStream

    Note over MG: Async notification for pending payments
    MG->>GW: POST /webhooks/midtrans<br/>{order_id, transaction_status, signature}
    GW->>PY: gRPC HandleWebhook(notification)

    PY->>PY: Verify webhook signature (HMAC-SHA512)
    PY->>DB_PY: SELECT * FROM payments WHERE id = order_id
    DB_PY-->>PY: Payment{status=processing}

    alt transaction_status == "settlement"
        PY->>DB_PY: UPDATE payments SET status='success', paid_at=now()
        PY->>BS: gRPC MarkBillingPaid(billing_record_id)
        PY--)NT: Publish parkir.payment.completed
    else transaction_status == "expire" or "cancel"
        PY->>DB_PY: UPDATE payments SET status='failed'
        PY--)NT: Publish parkir.payment.failed
    end

    PY-->>GW: 200 OK
    GW-->>MG: 200 OK
```

---

## 4. Cache Invalidation Flow

Event-driven cache invalidation ensures the search read model stays fresh.

```mermaid
sequenceDiagram
    autonumber
    participant RS as Reservation Service
    participant NT as NATS JetStream
    participant SS as Search Service
    participant DB_S as PostgreSQL (search.*)
    participant RD as Redis Cache
    participant C as Client (next query)

    Note over RS: Any state change triggers event
    RS--)NT: Publish parkir.reservation.created<br/>{spot_id, start_time, end_time, status}

    Note over NT,SS: NATS delivers to search consumer group
    NT->>SS: Pull message (batch of 10)

    Note over SS: Update read model (database)
    SS->>DB_S: UPDATE spot_read_model<br/>SET is_available = false,<br/>next_available_at = end_time,<br/>active_reservations_count = active_reservations_count + 1,<br/>last_updated = now()<br/>WHERE spot_id = ?
    DB_S-->>SS: OK (1 row updated)

    Note over SS: Invalidate cache layers
    SS->>RD: DEL cache:spot:{spot_id}
    RD-->>SS: OK
    SS->>RD: DEL cache:lot:{lot_id}:available
    RD-->>SS: OK
    SS->>RD: DEL cache:search:{geo_hash}:*
    Note over SS: Pattern delete for geo-area caches
    RD-->>SS: OK

    Note over SS: Acknowledge message
    SS->>NT: ACK message

    Note over C,RD: Next client query
    C->>SS: SearchAvailableSpots(location, time)
    SS->>RD: GET cache:search:{geo_hash}:{time_window}
    RD-->>SS: nil (cache miss after invalidation)
    SS->>DB_S: SELECT * FROM spot_read_model<br/>WHERE ST_DWithin(...) AND is_available = true
    DB_S-->>SS: [updated results]
    SS->>RD: SET cache:search:{key} {results} EX 30
    SS-->>C: Fresh results
```

### Bulk Invalidation (Lot Status Change)

```mermaid
sequenceDiagram
    autonumber
    participant AD as Admin API
    participant RS as Reservation Service
    participant NT as NATS JetStream
    participant SS as Search Service
    participant DB_S as PostgreSQL (search.*)
    participant RD as Redis Cache

    Note over AD: Admin deactivates entire parking lot
    AD->>RS: gRPC DeactivateLot(lot_id)
    RS->>RS: Update all spots in lot to is_active=false
    RS--)NT: Publish parkir.lot.deactivated<br/>{lot_id, spot_ids: [...]}

    NT->>SS: Deliver lot.deactivated event

    Note over SS: Batch update read model
    SS->>DB_S: UPDATE spot_read_model<br/>SET is_available = false<br/>WHERE lot_id = ?
    DB_S-->>SS: OK (N rows updated)

    Note over SS: Bulk cache invalidation
    SS->>RD: EVAL scan_and_delete<br/>pattern: cache:spot:{lot_id}:*
    RD-->>SS: Deleted M keys

    SS->>RD: DEL cache:lot:{lot_id}:*
    RD-->>SS: OK

    SS->>NT: ACK
```

### Consistency Guarantee: Lag Monitoring

```mermaid
sequenceDiagram
    autonumber
    participant NT as NATS JetStream
    participant SS as Search Service
    participant MO as Monitoring (Prometheus)

    Note over NT,SS: Consumer lag tracking
    loop Every 10 seconds
        SS->>NT: ConsumerInfo(stream, consumer)
        NT-->>SS: {num_pending: 5, num_ack_pending: 2}
        SS->>MO: Gauge: search_consumer_lag = 5
        SS->>MO: Gauge: search_ack_pending = 2
    end

    Note over MO: Alert if lag > 1000 messages or > 30s
    alt Lag exceeds threshold
        MO->>MO: Fire alert: search_read_model_stale
    end
```

---

## 5. Check-out and Billing Finalization

```mermaid
sequenceDiagram
    autonumber
    participant C as Client (Mobile App)
    participant GW as API Gateway
    participant RS as Reservation Service
    participant PS as Presence Service
    participant BS as Billing Service
    participant DB_R as PostgreSQL (reservation.*)
    participant DB_P as PostgreSQL (presence.*)
    participant DB_B as PostgreSQL (billing.*)
    participant NT as NATS JetStream
    participant SS as Search Service

    C->>GW: POST /api/v1/reservations/{id}/check-out
    GW->>RS: gRPC CheckOut(reservation_id)

    RS->>DB_R: SELECT * FROM reservations WHERE id = ?
    DB_R-->>RS: Reservation{status=checked_in, end_time}

    Note over RS: Record check-out presence
    RS->>PS: gRPC RecordPresence(reservation_id, CHECK_OUT)
    PS->>DB_P: INSERT INTO presence_logs (event_type='check_out', event_time=now())
    DB_P-->>PS: OK
    PS-->>RS: OK

    Note over RS: Check for overstay
    RS->>RS: Calculate: actual_end = now(), planned_end = reservation.end_time
    RS->>RS: overstay_minutes = max(0, actual_end - planned_end)

    RS->>DB_R: UPDATE reservations SET status='completed'
    DB_R-->>RS: OK

    Note over RS: Finalize billing
    RS->>BS: gRPC FinalizeBilling(reservation_id, actual_end)
    BS->>DB_B: UPDATE billing_records<br/>SET billing_period_end = actual_end,<br/>amount = calculated_amount
    DB_B-->>BS: OK

        BS->>DB_B: UPDATE billing_records<br/>SET total_amount = base + overnight_fees
    BS-->>RS: BillingFinalized{total_amount}
    RS-->>GW: CheckOutResponse{status=completed, total_due}
    GW-->>C: 200 OK {checkout complete, billing summary}

    Note over RS: Publish events
    RS--)NT: Publish parkir.reservation.completed<br/>{reservation_id, spot_id}

    NT--)SS: Deliver to search consumer
    SS->>SS: UPDATE spot_read_model<br/>SET is_available = true,<br/>active_reservations_count -= 1
    SS->>RD: DEL cache:spot:{spot_id}
```
