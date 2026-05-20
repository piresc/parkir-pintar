package grpc

import (
	"context"
	"errors"
	"testing"

	"parkir-pintar/internal/presence/usecase"
	presencev1 "parkir-pintar/proto/presence/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockUsecase implements usecase.Usecase for testing.
type mockUsecase struct {
	result *usecase.VerifyResult
	err    error
}

func (m *mockUsecase) VerifyPresence(_ context.Context, _ string, _ int, _ int) (*usecase.VerifyResult, error) {
	return m.result, m.err
}

func TestVerifyPresence_ShouldReturnInvalidArgument_WhenDriverIdEmpty(t *testing.T) {
	h := NewHandler(&mockUsecase{})

	req := &presencev1.VerifyPresenceRequest{
		DriverId:      "",
		ReservationId: "res-123",
		FloorNumber:   1,
		SpotNumber:    5,
	}

	resp, err := h.VerifyPresence(context.Background(), req)

	assert.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "driver_id")
}

func TestVerifyPresence_ShouldReturnInvalidArgument_WhenReservationIdEmpty(t *testing.T) {
	h := NewHandler(&mockUsecase{})

	req := &presencev1.VerifyPresenceRequest{
		DriverId:      "driver-1",
		ReservationId: "",
		FloorNumber:   1,
		SpotNumber:    5,
	}

	resp, err := h.VerifyPresence(context.Background(), req)

	assert.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "reservation_id")
}

func TestVerifyPresence_ShouldReturnInternal_WhenUsecaseFails(t *testing.T) {
	mock := &mockUsecase{err: errors.New("unexpected failure")}
	h := NewHandler(mock)

	req := &presencev1.VerifyPresenceRequest{
		DriverId:      "driver-1",
		ReservationId: "res-123",
		FloorNumber:   1,
		SpotNumber:    5,
	}

	resp, err := h.VerifyPresence(context.Background(), req)

	assert.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestVerifyPresence_ShouldReturnVerifiedResponse_WhenSuccess(t *testing.T) {
	mock := &mockUsecase{
		result: &usecase.VerifyResult{
			Verified: true,
			Message:  "spot occupied, presence confirmed",
		},
	}
	h := NewHandler(mock)

	req := &presencev1.VerifyPresenceRequest{
		DriverId:      "driver-1",
		ReservationId: "res-123",
		FloorNumber:   2,
		SpotNumber:    10,
	}

	resp, err := h.VerifyPresence(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.Verified)
	assert.Equal(t, "spot occupied, presence confirmed", resp.Message)
}

func TestVerifyPresence_ShouldReturnNotVerifiedResponse_WhenSpotEmpty(t *testing.T) {
	mock := &mockUsecase{
		result: &usecase.VerifyResult{
			Verified: false,
			Message:  "spot not occupied, driver may be at wrong spot",
		},
	}
	h := NewHandler(mock)

	req := &presencev1.VerifyPresenceRequest{
		DriverId:      "driver-2",
		ReservationId: "res-456",
		FloorNumber:   1,
		SpotNumber:    3,
	}

	resp, err := h.VerifyPresence(context.Background(), req)

	require.NoError(t, err)
	assert.False(t, resp.Verified)
	assert.Equal(t, "spot not occupied, driver may be at wrong spot", resp.Message)
}

func TestNewHandler_ShouldReturnHandlerInstance(t *testing.T) {
	mock := &mockUsecase{}
	h := NewHandler(mock)

	assert.NotNil(t, h)
}
