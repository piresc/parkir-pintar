package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpotUpdatedEvent_ShouldMarshalAndUnmarshal_WhenAllFieldsSet(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	original := SpotUpdatedEvent{
		SpotID:      "spot-123",
		FloorNumber: 2,
		SpotNumber:  15,
		VehicleType: "car",
		SpotCode:    "A2-15",
		Status:      "available",
		UpdatedAt:   now,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded SpotUpdatedEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.SpotID, decoded.SpotID)
	assert.Equal(t, original.FloorNumber, decoded.FloorNumber)
	assert.Equal(t, original.SpotNumber, decoded.SpotNumber)
	assert.Equal(t, original.VehicleType, decoded.VehicleType)
	assert.Equal(t, original.SpotCode, decoded.SpotCode)
	assert.Equal(t, original.Status, decoded.Status)
	assert.True(t, original.UpdatedAt.Equal(decoded.UpdatedAt))
}

func TestSpotUpdatedEvent_ShouldProduceExpectedJSON_WhenMarshal(t *testing.T) {
	event := SpotUpdatedEvent{
		SpotID:      "spot-abc",
		FloorNumber: 1,
		SpotNumber:  3,
		VehicleType: "motorcycle",
		SpotCode:    "B1-03",
		Status:      "occupied",
		UpdatedAt:   time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	assert.Equal(t, "spot-abc", m["spot_id"])
	assert.Equal(t, float64(1), m["floor_number"])
	assert.Equal(t, float64(3), m["spot_number"])
	assert.Equal(t, "motorcycle", m["vehicle_type"])
	assert.Equal(t, "B1-03", m["spot_code"])
	assert.Equal(t, "occupied", m["status"])
	assert.Contains(t, m, "updated_at")
}

func TestPaymentResultEvent_ShouldMarshalAndUnmarshal_WhenAllFieldsSet(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	original := PaymentResultEvent{
		PaymentID:     "pay-456",
		ReservationID: "res-789",
		Amount:        50000,
		Status:        "success",
		Reason:        "",
		Timestamp:     now,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded PaymentResultEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.PaymentID, decoded.PaymentID)
	assert.Equal(t, original.ReservationID, decoded.ReservationID)
	assert.Equal(t, original.Amount, decoded.Amount)
	assert.Equal(t, original.Status, decoded.Status)
	assert.Equal(t, original.Reason, decoded.Reason)
	assert.True(t, original.Timestamp.Equal(decoded.Timestamp))
}

func TestPaymentResultEvent_ShouldOmitReason_WhenEmpty(t *testing.T) {
	event := PaymentResultEvent{
		PaymentID:     "pay-001",
		ReservationID: "res-001",
		Amount:        25000,
		Status:        "success",
		Reason:        "",
		Timestamp:     time.Now().UTC(),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	_, hasReason := m["reason"]
	assert.False(t, hasReason, "reason should be omitted when empty")
}

func TestPaymentResultEvent_ShouldIncludeReason_WhenFailed(t *testing.T) {
	event := PaymentResultEvent{
		PaymentID:     "pay-002",
		ReservationID: "res-002",
		Amount:        25000,
		Status:        "failed",
		Reason:        "insufficient balance",
		Timestamp:     time.Now().UTC(),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	assert.Equal(t, "insufficient balance", m["reason"])
}

func TestReservationEvent_ShouldMarshalAndUnmarshal_WhenAllFieldsSet(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	original := ReservationEvent{
		ReservationID: "res-100",
		DriverID:      "driver-200",
		SpotID:        "spot-300",
		VehicleType:   "car",
		Status:        "confirmed",
		Timestamp:     now,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ReservationEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.ReservationID, decoded.ReservationID)
	assert.Equal(t, original.DriverID, decoded.DriverID)
	assert.Equal(t, original.SpotID, decoded.SpotID)
	assert.Equal(t, original.VehicleType, decoded.VehicleType)
	assert.Equal(t, original.Status, decoded.Status)
	assert.True(t, original.Timestamp.Equal(decoded.Timestamp))
}

func TestReservationEvent_ShouldProduceExpectedJSONKeys(t *testing.T) {
	event := ReservationEvent{
		ReservationID: "res-x",
		DriverID:      "drv-y",
		SpotID:        "spt-z",
		VehicleType:   "truck",
		Status:        "checked-in",
		Timestamp:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	expectedKeys := []string{"reservation_id", "driver_id", "spot_id", "vehicle_type", "status", "timestamp"}
	for _, key := range expectedKeys {
		assert.Contains(t, m, key, "expected key %s in JSON output", key)
	}
}

func TestSpotUpdatedEvent_ShouldHandleZeroValues(t *testing.T) {
	var event SpotUpdatedEvent

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded SpotUpdatedEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, event, decoded)
}
