// Package usecase implements the business logic layer for the notification domain.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/assert and testify/require for assertions
// - Each test is isolated with its own setup
// - Use t.Context() for Go 1.24+ context in tests
// - Keep tests simple and focused on the behavior being tested
// - Don't mock the class under test
// - Don't write weak assertions (e.g., just checking assert.NotNil)
package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/notification/model"
)

// TestSendPush_ShouldReturnStatusLogged_WhenCalled verifies that SendPush
// returns a NotificationResult with Status="logged" and a non-empty ID.
func TestSendPush_ShouldReturnStatusLogged_WhenCalled(t *testing.T) {
	// Arrange
	uc := NewUsecase()
	req := &model.SendPushRequest{
		DriverID: "driver-1",
		Title:    "Reservation Confirmed",
		Body:     "Your spot F3-C-012 is reserved.",
		Data:     map[string]string{"reservation_id": "res-1"},
	}

	// Act
	result, err := uc.SendPush(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "logged", result.Status)
	assert.Equal(t, "push", result.Channel)
	assert.NotEmpty(t, result.ID)
	assert.Contains(t, result.Payload, "driver-1")
	assert.False(t, result.CreatedAt.IsZero())
}

// TestSendSMS_ShouldReturnStatusLogged_WhenCalled verifies that SendSMS
// returns a NotificationResult with Status="logged" and a non-empty ID.
func TestSendSMS_ShouldReturnStatusLogged_WhenCalled(t *testing.T) {
	// Arrange
	uc := NewUsecase()
	req := &model.SendSMSRequest{
		PhoneNumber: "+6281234567890",
		Message:     "Your parking session has started.",
	}

	// Act
	result, err := uc.SendSMS(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "logged", result.Status)
	assert.Equal(t, "sms", result.Channel)
	assert.NotEmpty(t, result.ID)
	assert.Contains(t, result.Payload, "+6281234567890")
	assert.False(t, result.CreatedAt.IsZero())
}

// TestSendEmail_ShouldReturnStatusLogged_WhenCalled verifies that SendEmail
// returns a NotificationResult with Status="logged" and a non-empty ID.
func TestSendEmail_ShouldReturnStatusLogged_WhenCalled(t *testing.T) {
	// Arrange
	uc := NewUsecase()
	req := &model.SendEmailRequest{
		Email:   "driver@example.com",
		Subject: "Parking Receipt",
		Body:    "Your total is 15,000 IDR.",
	}

	// Act
	result, err := uc.SendEmail(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "logged", result.Status)
	assert.Equal(t, "email", result.Channel)
	assert.NotEmpty(t, result.ID)
	assert.Contains(t, result.Payload, "driver@example.com")
	assert.False(t, result.CreatedAt.IsZero())
}

// TestSendPush_ShouldReturnUniqueIDs_WhenCalledMultipleTimes verifies that
// each call to SendPush returns a unique ID.
func TestSendPush_ShouldReturnUniqueIDs_WhenCalledMultipleTimes(t *testing.T) {
	// Arrange
	uc := NewUsecase()
	req := &model.SendPushRequest{
		DriverID: "driver-1",
		Title:    "Test",
		Body:     "Test body",
	}

	// Act
	result1, err1 := uc.SendPush(t.Context(), req)
	result2, err2 := uc.SendPush(t.Context(), req)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, result1.ID, result2.ID)
}
