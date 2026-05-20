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

func (m *mockExpirer) ExpireReservation(ctx context.Context, reservationID string) error {
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

func (m *mockFailer) FailReservation(ctx context.Context, reservationID string, paymentID string) error {
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

func TestNewReservationExpiryHandler_ShouldReturnHandler(t *testing.T) {
	expirer := &mockExpirer{}
	handler := NewReservationExpiryHandler(expirer)
	require.NotNil(t, handler)
}

func TestReservationExpiryHandler_ProcessTask_ShouldCallExpirer_WhenValidPayload(t *testing.T) {
	expirer := &mockExpirer{}
	handler := NewReservationExpiryHandler(expirer)

	payload, err := json.Marshal(ReservationExpiryPayload{ReservationID: "res-abc"})
	require.NoError(t, err)

	task := asynq.NewTask(TypeReservationExpire, payload)
	err = handler.ProcessTask(context.Background(), task)

	require.NoError(t, err)
	assert.True(t, expirer.called)
	assert.Equal(t, "res-abc", expirer.reservationID)
}

func TestReservationExpiryHandler_ProcessTask_ShouldReturnError_WhenInvalidJSON(t *testing.T) {
	expirer := &mockExpirer{}
	handler := NewReservationExpiryHandler(expirer)

	task := asynq.NewTask(TypeReservationExpire, []byte("invalid json"))
	err := handler.ProcessTask(context.Background(), task)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal reservation expiry payload")
	assert.False(t, expirer.called)
}

func TestReservationExpiryHandler_ProcessTask_ShouldReturnError_WhenReservationIDEmpty(t *testing.T) {
	expirer := &mockExpirer{}
	handler := NewReservationExpiryHandler(expirer)

	payload, err := json.Marshal(ReservationExpiryPayload{ReservationID: ""})
	require.NoError(t, err)

	task := asynq.NewTask(TypeReservationExpire, payload)
	err = handler.ProcessTask(context.Background(), task)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reservation_id is required")
	assert.False(t, expirer.called)
}

func TestReservationExpiryHandler_ProcessTask_ShouldPropagateError_WhenExpirerFails(t *testing.T) {
	expirer := &mockExpirer{err: errors.New("db connection failed")}
	handler := NewReservationExpiryHandler(expirer)

	payload, err := json.Marshal(ReservationExpiryPayload{ReservationID: "res-fail"})
	require.NoError(t, err)

	task := asynq.NewTask(TypeReservationExpire, payload)
	err = handler.ProcessTask(context.Background(), task)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db connection failed")
}

func TestNewPaymentHoldTimeoutHandler_ShouldReturnHandler(t *testing.T) {
	failer := &mockFailer{}
	handler := NewPaymentHoldTimeoutHandler(failer)
	require.NotNil(t, handler)
}

func TestPaymentHoldTimeoutHandler_ProcessTask_ShouldCallFailer_WhenValidPayload(t *testing.T) {
	failer := &mockFailer{}
	handler := NewPaymentHoldTimeoutHandler(failer)

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

func TestPaymentHoldTimeoutHandler_ProcessTask_ShouldReturnError_WhenInvalidJSON(t *testing.T) {
	failer := &mockFailer{}
	handler := NewPaymentHoldTimeoutHandler(failer)

	task := asynq.NewTask(TypePaymentHoldTimeout, []byte("{bad"))
	err := handler.ProcessTask(context.Background(), task)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal payment hold timeout payload")
	assert.False(t, failer.called)
}

func TestPaymentHoldTimeoutHandler_ProcessTask_ShouldReturnError_WhenReservationIDEmpty(t *testing.T) {
	failer := &mockFailer{}
	handler := NewPaymentHoldTimeoutHandler(failer)

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

func TestPaymentHoldTimeoutHandler_ProcessTask_ShouldCallFailer_WhenPaymentIDEmpty(t *testing.T) {
	failer := &mockFailer{}
	handler := NewPaymentHoldTimeoutHandler(failer)

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

func TestPaymentHoldTimeoutHandler_ProcessTask_ShouldPropagateError_WhenFailerFails(t *testing.T) {
	failer := &mockFailer{err: errors.New("service unavailable")}
	handler := NewPaymentHoldTimeoutHandler(failer)

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

func TestNewClient_ShouldReturnNonNilClient(t *testing.T) {
	client := NewClient("localhost:6379", "")
	require.NotNil(t, client)
	_ = client.Close()
}

func TestNewClient_ShouldAcceptPassword(t *testing.T) {
	client := NewClient("localhost:6379", "secret-password")
	require.NotNil(t, client)
	_ = client.Close()
}

func TestNewServer_ShouldReturnNonNilServer(t *testing.T) {
	server := NewServer("localhost:6379", "", 10)
	require.NotNil(t, server)
}

func TestNewServer_ShouldAcceptConcurrencyValue(t *testing.T) {
	server := NewServer("localhost:6379", "pass", 5)
	require.NotNil(t, server)
}
