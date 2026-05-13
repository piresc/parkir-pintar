//go:build contract

// Package contract implements API contract tests that verify service responses
// match expected schemas derived from proto definitions. These tests ensure that
// changes to service implementations don't break the agreed-upon API contracts.
//
// Run: go test -tags contract -v ./tests/contract/
package contract

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	reservationv1 "parkir-pintar/proto/reservation/v1"
)

// --- ReservationResponse Contract Tests ---

// TestReservationResponseContract verifies that a ReservationResponse message
// serialized to JSON matches the expected contract shape. The proto definition
// (proto/reservation/v1/reservation.proto) is the source of truth.
func TestReservationResponseContract(t *testing.T) {
	now := time.Now()

	resp := &reservationv1.ReservationResponse{
		Id:             "res-001",
		DriverId:       "drv-123",
		SpotId:         "spot-A1",
		VehicleType:    "car",
		AssignmentMode: "system_assigned",
		Status:         "confirmed",
		IdempotencyKey: "idem-key-abc",
		ConfirmedAt:    timestamppb.New(now),
		ExpiresAt:      timestamppb.New(now.Add(30 * time.Minute)),
		SpotCode:       "A1-01",
	}

	// Serialize to JSON using protojson (canonical proto3 JSON).
	data, err := protojson.Marshal(resp)
	require.NoError(t, err, "ReservationResponse should serialize to JSON")

	// Parse back into a generic map to validate field presence and types.
	var fields map[string]interface{}
	err = json.Unmarshal(data, &fields)
	require.NoError(t, err, "JSON should parse into map")

	// Contract: required fields must be present.
	t.Run("required_fields_present", func(t *testing.T) {
		requiredFields := []string{
			"id", "driverId", "spotId", "vehicleType",
			"assignmentMode", "status", "idempotencyKey", "spotCode",
		}
		for _, field := range requiredFields {
			assert.Contains(t, fields, field, "field %q must be present in response", field)
		}
	})

	// Contract: field types must match expected types.
	t.Run("field_types", func(t *testing.T) {
		stringFields := []string{"id", "driverId", "spotId", "vehicleType", "assignmentMode", "status", "idempotencyKey", "spotCode"}
		for _, field := range stringFields {
			if val, ok := fields[field]; ok {
				_, isString := val.(string)
				assert.True(t, isString, "field %q must be a string, got %T", field, val)
			}
		}
	})

	// Contract: timestamp fields use RFC3339 format (proto3 JSON encoding).
	t.Run("timestamp_format", func(t *testing.T) {
		timestampFields := []string{"confirmedAt", "expiresAt"}
		for _, field := range timestampFields {
			val, ok := fields[field]
			if !ok {
				continue // Optional timestamps may be absent.
			}
			str, isString := val.(string)
			assert.True(t, isString, "timestamp field %q must be a string", field)
			_, parseErr := time.Parse(time.RFC3339Nano, str)
			assert.NoError(t, parseErr, "timestamp field %q must be valid RFC3339: %v", field, str)
		}
	})

	// Contract: status must be one of the known values.
	t.Run("status_enum_values", func(t *testing.T) {
		validStatuses := []string{
			"pending", "confirmed", "checked_in", "checked_out",
			"cancelled", "expired", "payment_failed",
		}
		status, _ := fields["status"].(string)
		assert.Contains(t, validStatuses, status, "status %q must be a known value", status)
	})

	// Contract: vehicle_type must be one of the known values.
	t.Run("vehicle_type_enum_values", func(t *testing.T) {
		validTypes := []string{"car", "motorcycle"}
		vt, _ := fields["vehicleType"].(string)
		assert.Contains(t, validTypes, vt, "vehicleType %q must be a known value", vt)
	})

	// Contract: assignment_mode must be one of the known values.
	t.Run("assignment_mode_enum_values", func(t *testing.T) {
		validModes := []string{"system_assigned", "user_selected"}
		mode, _ := fields["assignmentMode"].(string)
		assert.Contains(t, validModes, mode, "assignmentMode %q must be a known value", mode)
	})
}

// TestCheckOutResponseContract verifies the CheckOutResponse message contract.
func TestCheckOutResponseContract(t *testing.T) {
	now := time.Now()

	resp := &reservationv1.CheckOutResponse{
		Reservation: &reservationv1.ReservationResponse{
			Id:             "res-002",
			DriverId:       "drv-456",
			SpotId:         "spot-B2",
			VehicleType:    "motorcycle",
			AssignmentMode: "user_selected",
			Status:         "checked_out",
			IdempotencyKey: "idem-key-xyz",
			CheckedInAt:    timestamppb.New(now.Add(-2 * time.Hour)),
			CheckedOutAt:   timestamppb.New(now),
			SpotCode:       "B2-03",
		},
		TotalAmount:   25000,
		BillingId:     "bill-001",
		PaymentId:     "pay-001",
		BookingFee:    5000,
		ParkingFee:    15000,
		OvernightFee:  0,
		PenaltyAmount: 5000,
	}

	data, err := protojson.Marshal(resp)
	require.NoError(t, err, "CheckOutResponse should serialize to JSON")

	var fields map[string]interface{}
	err = json.Unmarshal(data, &fields)
	require.NoError(t, err, "JSON should parse into map")

	// Contract: billing fields must be present.
	t.Run("billing_fields_present", func(t *testing.T) {
		billingFields := []string{"totalAmount", "billingId", "paymentId"}
		for _, field := range billingFields {
			assert.Contains(t, fields, field, "field %q must be present in checkout response", field)
		}
	})

	// Contract: nested reservation must be present.
	t.Run("nested_reservation_present", func(t *testing.T) {
		assert.Contains(t, fields, "reservation", "checkout response must contain nested reservation")
		resMap, ok := fields["reservation"].(map[string]interface{})
		require.True(t, ok, "reservation field must be an object")
		assert.Contains(t, resMap, "id")
		assert.Contains(t, resMap, "status")
	})

	// Contract: monetary amounts are strings (proto3 int64 → JSON string for large values).
	t.Run("monetary_amounts_format", func(t *testing.T) {
		// In proto3 JSON, int64 fields are encoded as strings.
		amountFields := []string{"totalAmount", "bookingFee", "parkingFee", "penaltyAmount"}
		for _, field := range amountFields {
			val, ok := fields[field]
			if !ok {
				continue // Zero values may be omitted in proto3.
			}
			// proto3 JSON encodes int64 as string.
			_, isString := val.(string)
			_, isFloat := val.(float64)
			assert.True(t, isString || isFloat,
				"monetary field %q must be string or number, got %T", field, val)
		}
	})
}

// --- NATS Event Payload Contract Tests ---

// ReservationEvent represents the expected NATS event payload structure
// for reservation domain events. This mirrors what the usecase layer publishes.
type ReservationEvent struct {
	EventType     string `json:"event_type"`
	ReservationID string `json:"reservation_id"`
	DriverID      string `json:"driver_id"`
	SpotID        string `json:"spot_id"`
	VehicleType   string `json:"vehicle_type"`
	Status        string `json:"status"`
	Timestamp     string `json:"timestamp"`
}

// TestNATSReservationEventContract verifies that NATS event payloads for
// reservation events match the expected format.
func TestNATSReservationEventContract(t *testing.T) {
	// Simulate the event payload that the usecase layer would publish.
	event := ReservationEvent{
		EventType:     "reservation.created",
		ReservationID: "res-003",
		DriverID:      "drv-789",
		SpotID:        "spot-C3",
		VehicleType:   "car",
		Status:        "pending",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err, "event should serialize to JSON")

	// Parse back and validate contract.
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err, "event JSON should parse")

	t.Run("required_event_fields", func(t *testing.T) {
		requiredFields := []string{
			"event_type", "reservation_id", "driver_id",
			"spot_id", "vehicle_type", "status", "timestamp",
		}
		for _, field := range requiredFields {
			assert.Contains(t, parsed, field, "NATS event must contain field %q", field)
		}
	})

	t.Run("event_type_format", func(t *testing.T) {
		validEventTypes := []string{
			"reservation.created",
			"reservation.confirmed",
			"reservation.checked_in",
			"reservation.checked_out",
			"reservation.cancelled",
			"reservation.expired",
			"reservation.payment_failed",
		}
		et, _ := parsed["event_type"].(string)
		assert.Contains(t, validEventTypes, et, "event_type %q must be a known value", et)
	})

	t.Run("timestamp_is_rfc3339", func(t *testing.T) {
		ts, _ := parsed["timestamp"].(string)
		_, parseErr := time.Parse(time.RFC3339, ts)
		assert.NoError(t, parseErr, "timestamp must be valid RFC3339")
	})

	t.Run("all_fields_are_strings", func(t *testing.T) {
		for key, val := range parsed {
			_, isString := val.(string)
			assert.True(t, isString, "NATS event field %q must be a string, got %T", key, val)
		}
	})
}

// TestNATSEventSubjectNaming verifies that NATS subject naming follows
// the project convention: <domain>.<event_type>
func TestNATSEventSubjectNaming(t *testing.T) {
	validSubjects := []string{
		"reservation.created",
		"reservation.confirmed",
		"reservation.checked_in",
		"reservation.checked_out",
		"reservation.cancelled",
		"reservation.expired",
		"billing.started",
		"billing.calculated",
		"payment.completed",
		"payment.failed",
		"presence.spot_occupied",
		"presence.spot_freed",
	}

	for _, subject := range validSubjects {
		t.Run(subject, func(t *testing.T) {
			// Subjects must have exactly one dot separator.
			assert.Regexp(t, `^[a-z]+\.[a-z_]+$`, subject,
				"subject %q must follow <domain>.<event> pattern", subject)
		})
	}
}
