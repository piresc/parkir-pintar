package asynq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockExpirer struct {
	called        bool
	reservationID string
	err           error
}

func (m *mockExpirer) ExpireReservation(_ context.Context, reservationID string) error {
	m.called = true
	m.reservationID = reservationID
	return m.err
}

type mockFailer struct {
	called        bool
	reservationID string
	paymentID     string
	err           error
}

func (m *mockFailer) FailReservation(_ context.Context, reservationID string, paymentID string) error {
	m.called = true
	m.reservationID = reservationID
	m.paymentID = paymentID
	return m.err
}

func TestTaskTypeConstants(t *testing.T) {
	assert.Equal(t, "task:reservation:expire", TypeReservationExpire)
	assert.Equal(t, "task:payment:hold_timeout", TypePaymentHoldTimeout)
}

func TestReservationExpiryPayload_ShouldMarshalCorrectly(t *testing.T) {
	payload := ReservationExpiryPayload{ReservationID: "res-123"}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded ReservationExpiryPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "res-123", decoded.ReservationID)
}

func TestPaymentHoldTimeoutPayload_ShouldMarshalCorrectly(t *testing.T) {
	payload := PaymentHoldTimeoutPayload{
		ReservationID: "res-456",
		PaymentID:     "pay-789",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded PaymentHoldTimeoutPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "res-456", decoded.ReservationID)
	assert.Equal(t, "pay-789", decoded.PaymentID)
}

func TestNewExpiryHandler_ShouldReturnHandler(t *testing.T) {
	expirer := &mockExpirer{}
	handler := NewExpiryHandler(expirer)
	require.NotNil(t, handler)
}

func TestExpiryHandler_ProcessTask_ShouldCallExpirer_WhenValidPayload(t *testing.T) {
	expirer := &mockExpirer{}
	handler := NewExpiryHandler(expirer)

	payload, err := json.Marshal(ReservationExpiryPayload{ReservationID: "res-abc"})
	require.NoError(t, err)

	task := asynq.NewTask(TypeReservationExpire, payload)
	err = handler.ProcessTask(context.Background(), task)

	require.NoError(t, err)
	assert.True(t, expirer.called)
	assert.Equal(t, "res-abc", expirer.reservationID)
}

func TestExpiryHandler_ProcessTask_ShouldReturnError_WhenInvalidJSON(t *testing.T) {
	expirer := &mockExpirer{}
	handler := NewExpiryHandler(expirer)

	task := asynq.NewTask(TypeReservationExpire, []byte("invalid json"))
	err := handler.ProcessTask(context.Background(), task)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal reservation expiry payload")
	assert.False(t, expirer.called)
}

func TestExpiryHandler_ProcessTask_ShouldReturnError_WhenReservationIDEmpty(t *testing.T) {
	expirer := &mockExpirer{}
	handler := NewExpiryHandler(expirer)

	payload, err := json.Marshal(ReservationExpiryPayload{ReservationID: ""})
	require.NoError(t, err)

	task := asynq.NewTask(TypeReservationExpire, payload)
	err = handler.ProcessTask(context.Background(), task)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reservation_id is required")
	assert.False(t, expirer.called)
}

func TestExpiryHandler_ProcessTask_ShouldPropagateError_WhenExpirerFails(t *testing.T) {
	expirer := &mockExpirer{err: errors.New("db connection failed")}
	handler := NewExpiryHandler(expirer)

	payload, err := json.Marshal(ReservationExpiryPayload{ReservationID: "res-fail"})
	require.NoError(t, err)

	task := asynq.NewTask(TypeReservationExpire, payload)
	err = handler.ProcessTask(context.Background(), task)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db connection failed")
}

func TestNewPaymentTimeoutHandler_ShouldReturnHandler(t *testing.T) {
	failer := &mockFailer{}
	handler := NewPaymentTimeoutHandler(failer)
	require.NotNil(t, handler)
}

func TestPaymentTimeoutHandler_ProcessTask_ShouldCallFailer_WhenValidPayload(t *testing.T) {
	failer := &mockFailer{}
	handler := NewPaymentTimeoutHandler(failer)

	payload, err := json.Marshal(PaymentHoldTimeoutPayload{
		ReservationID: "res-xyz",
		PaymentID:     "pay-xyz",
	})
	require.NoError(t, err)

	task := asynq.NewTask(TypePaymentHoldTimeout, payload)
	err = handler.ProcessTask(context.Background(), task)

	require.NoError(t, err)
	assert.True(t, failer.called)
	assert.Equal(t, "res-xyz", failer.reservationID)
	assert.Equal(t, "pay-xyz", failer.paymentID)
}

func TestPaymentTimeoutHandler_ProcessTask_ShouldReturnError_WhenInvalidJSON(t *testing.T) {
	failer := &mockFailer{}
	handler := NewPaymentTimeoutHandler(failer)

	task := asynq.NewTask(TypePaymentHoldTimeout, []byte("{bad"))
	err := handler.ProcessTask(context.Background(), task)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal payment hold timeout payload")
	assert.False(t, failer.called)
}

func TestPaymentTimeoutHandler_ProcessTask_ShouldReturnError_WhenReservationIDEmpty(t *testing.T) {
	failer := &mockFailer{}
	handler := NewPaymentTimeoutHandler(failer)

	payload, err := json.Marshal(PaymentHoldTimeoutPayload{
		ReservationID: "",
		PaymentID:     "pay-001",
	})
	require.NoError(t, err)

	task := asynq.NewTask(TypePaymentHoldTimeout, payload)
	err = handler.ProcessTask(context.Background(), task)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reservation_id is required")
	assert.False(t, failer.called)
}

func TestPaymentTimeoutHandler_ProcessTask_ShouldCallFailer_WhenPaymentIDEmpty(t *testing.T) {
	failer := &mockFailer{}
	handler := NewPaymentTimeoutHandler(failer)

	payload, err := json.Marshal(PaymentHoldTimeoutPayload{
		ReservationID: "res-001",
		PaymentID:     "",
	})
	require.NoError(t, err)

	task := asynq.NewTask(TypePaymentHoldTimeout, payload)
	err = handler.ProcessTask(context.Background(), task)

	require.NoError(t, err)
	assert.True(t, failer.called)
}

func TestPaymentTimeoutHandler_ProcessTask_ShouldPropagateError_WhenFailerFails(t *testing.T) {
	failer := &mockFailer{err: errors.New("service unavailable")}
	handler := NewPaymentTimeoutHandler(failer)

	payload, err := json.Marshal(PaymentHoldTimeoutPayload{
		ReservationID: "res-err",
		PaymentID:     "pay-err",
	})
	require.NoError(t, err)

	task := asynq.NewTask(TypePaymentHoldTimeout, payload)
	err = handler.ProcessTask(context.Background(), task)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "service unavailable")
}
