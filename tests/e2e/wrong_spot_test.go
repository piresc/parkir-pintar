// Package e2e_test — wrong-spot detection integration test (sensor-based).
//
// Best practices applied (from Go testify testing standards):
// - Use require for assertions that must pass to continue (fail-fast)
// - Use assert for non-critical checks
// - Follow AAA (Arrange-Act-Assert) structure
// - Use descriptive test names: Test[Scenario]_Should[Expected]_When[Condition]
// - Do not mock the database — these are integration tests against real PostgreSQL
package e2e_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/reservation/constants"
	"parkir-pintar/internal/reservation/model"
	reservationuc "parkir-pintar/internal/reservation/usecase"
	"parkir-pintar/tests/testhelpers"
)

// stubPresenceClient is a test stub implementing the PresenceClient interface.
// It returns a configurable PresenceResult for sensor-based verification.
type stubPresenceClient struct {
	verified bool
	message  string
}

func (s *stubPresenceClient) VerifyPresence(_ context.Context, _ string, _ string, _ int, _ int) (*reservationuc.PresenceResult, error) {
	return &reservationuc.PresenceResult{
		Verified: s.verified,
		Message:  s.message,
	}, nil
}

// newUsecaseWithPresence creates a reservation usecase wired with the given
// presence client stub, reusing the shared test environment's repos and adapters.
func newUsecaseWithPresence(presence reservationuc.PresenceClient) reservationuc.Usecase {
	redisAdapter := &reservationLockerAdapter{client: env.redisClient}
	billAdapter := &billingAdapter{uc: env.billingUC}
	payAdapter := &paymentAdapter{uc: env.paymentUC}

	return reservationuc.NewUsecase(
		env.reservationRepo,
		redisAdapter,
		billAdapter,
		payAdapter,
		presence,
		nil, // taskEnqueuer
		nil, // eventPublisher
		60,
		10,
	)
}

// TestWrongSpotDetection_ShouldFlagWrongSpot_WhenSensorShowsEmpty verifies
// that when the presence sensor reports the spot is empty (driver not detected),
// the check-in still succeeds but WrongSpotWarning is set to true.
//
// Validates: Requirement 9.1 — sensor-based wrong-spot detection (non-blocking).
func TestWrongSpotDetection_ShouldFlagWrongSpot_WhenSensorShowsEmpty(t *testing.T) {
	// Arrange — Create a usecase with a presence stub that returns verified=false
	ctx := context.Background()

	err := testhelpers.TruncateTables(ctx, env.db, "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)

	presenceStub := &stubPresenceClient{
		verified: false,
		message:  "sensor shows spot empty",
	}
	uc := newUsecaseWithPresence(presenceStub)

	driverID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
	require.NoError(t, err)

	// Create reservation
	reservation, err := uc.CreateReservation(ctx, &model.CreateReservationRequest{
		DriverID:       driverID,
		VehicleType:    "car",
		AssignmentMode: string(constants.AssignmentSystemAssigned),
		IdempotencyKey: uuid.New().String(),
	})
	require.NoError(t, err)
	require.NotNil(t, reservation)
	require.Equal(t, string(constants.StatusWaitingPayment), reservation.Status)

	// Confirm reservation (processes booking fee payment)
	confirmed, err := uc.ConfirmReservation(ctx, &model.ConfirmReservationRequest{
		ReservationID: reservation.ID,
		CallerID:      driverID,
	})
	require.NoError(t, err)
	require.Equal(t, string(constants.StatusConfirmed), confirmed.Status)

	// Act — Check in (presence sensor returns verified=false)
	checkInResp, err := uc.CheckIn(ctx, &model.CheckInRequest{
		ReservationID: reservation.ID,
		CallerID:      driverID,
	})

	// Assert — Check-in succeeds (non-blocking) but flags wrong spot
	require.NoError(t, err, "check-in should succeed even when sensor shows empty")
	require.NotNil(t, checkInResp)
	assert.Equal(t, string(constants.StatusCheckedIn), checkInResp.Reservation.Status,
		"reservation should transition to checked_in regardless of presence result")
	assert.True(t, checkInResp.WrongSpotWarning,
		"WrongSpotWarning should be true when sensor shows spot is empty")
}

// TestWrongSpotDetection_ShouldVerifyOK_WhenSensorShowsOccupied verifies
// that when the presence sensor confirms the spot is occupied (driver detected),
// the check-in succeeds with WrongSpotWarning=false.
//
// Validates: Requirement 9.1 — correct spot detection (positive case).
func TestWrongSpotDetection_ShouldVerifyOK_WhenSensorShowsOccupied(t *testing.T) {
	// Arrange — Create a usecase with a presence stub that returns verified=true
	ctx := context.Background()

	err := testhelpers.TruncateTables(ctx, env.db, "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)

	presenceStub := &stubPresenceClient{
		verified: true,
		message:  "sensor confirms presence",
	}
	uc := newUsecaseWithPresence(presenceStub)

	driverID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
	require.NoError(t, err)

	// Create reservation
	reservation, err := uc.CreateReservation(ctx, &model.CreateReservationRequest{
		DriverID:       driverID,
		VehicleType:    "car",
		AssignmentMode: string(constants.AssignmentSystemAssigned),
		IdempotencyKey: uuid.New().String(),
	})
	require.NoError(t, err)
	require.NotNil(t, reservation)

	// Confirm reservation
	confirmed, err := uc.ConfirmReservation(ctx, &model.ConfirmReservationRequest{
		ReservationID: reservation.ID,
		CallerID:      driverID,
	})
	require.NoError(t, err)
	require.Equal(t, string(constants.StatusConfirmed), confirmed.Status)

	// Act — Check in (presence sensor returns verified=true)
	checkInResp, err := uc.CheckIn(ctx, &model.CheckInRequest{
		ReservationID: reservation.ID,
		CallerID:      driverID,
	})

	// Assert — Check-in succeeds with no warning
	require.NoError(t, err)
	require.NotNil(t, checkInResp)
	assert.Equal(t, string(constants.StatusCheckedIn), checkInResp.Reservation.Status)
	assert.False(t, checkInResp.WrongSpotWarning,
		"WrongSpotWarning should be false when sensor confirms presence")
}
