CREATE TABLE IF NOT EXISTS reservation_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id UUID NOT NULL,
    driver_id UUID NOT NULL,
    spot_id UUID NOT NULL,
    vehicle_type VARCHAR(50) NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reservation_events_reservation_id ON reservation_events(reservation_id);
CREATE INDEX idx_reservation_events_timestamp ON reservation_events(timestamp);
CREATE INDEX idx_reservation_events_status ON reservation_events(status);
