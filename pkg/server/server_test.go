// Best practices applied from MCP knowledgebase (Go Testing Guidelines):
// - Descriptive test names using Test[FunctionName]_Should[Result]_When[Condition] pattern
// - AAA (Arrange-Act-Assert) structure
// - Test both success and error scenarios
// - Use testify/assert and testify/require for assertions
// - Tests are fast, isolated, repeatable, clear, and comprehensive
// - Mock external dependencies via interfaces
package server

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewShutdownManager_ShouldReturnManager_WhenLoggerProvided(t *testing.T) {
	// Arrange
	logger := newTestLogger()

	// Act
	sm := NewShutdownManager(logger)

	// Assert
	require.NotNil(t, sm)
	assert.NotNil(t, sm.functions)
	assert.Equal(t, 0, len(sm.functions))
}

func TestShutdownManagerRegister_ShouldAddFunction_WhenFunctionProvided(t *testing.T) {
	// Arrange
	sm := NewShutdownManager(newTestLogger())
	fn := func(_ context.Context) error { return nil }

	// Act
	sm.Register(fn)

	// Assert
	assert.Equal(t, 1, len(sm.functions))
}

func TestShutdownManagerShutdown_ShouldReturnNil_WhenAllFunctionsSucceed(t *testing.T) {
	// Arrange
	sm := NewShutdownManager(newTestLogger())
	callOrder := make([]int, 0)
	sm.Register(func(_ context.Context) error {
		callOrder = append(callOrder, 1)
		return nil
	})
	sm.Register(func(_ context.Context) error {
		callOrder = append(callOrder, 2)
		return nil
	})
	ctx := context.Background()

	// Act
	err := sm.Shutdown(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2}, callOrder, "functions should be called in registration order")
}

func TestShutdownManagerShutdown_ShouldContinueOnFailure_WhenFunctionFails(t *testing.T) {
	// Arrange
	sm := NewShutdownManager(newTestLogger())
	callOrder := make([]int, 0)
	sm.Register(func(_ context.Context) error {
		callOrder = append(callOrder, 1)
		return errors.New("first cleanup failed")
	})
	sm.Register(func(_ context.Context) error {
		callOrder = append(callOrder, 2)
		return nil
	})
	sm.Register(func(_ context.Context) error {
		callOrder = append(callOrder, 3)
		return nil
	})
	ctx := context.Background()

	// Act
	err := sm.Shutdown(ctx)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "first cleanup failed")
	// All 3 functions should have been called despite the first one failing
	assert.Equal(t, []int{1, 2, 3}, callOrder, "all functions should run even if one fails")
}

func TestShutdownManagerShutdown_ShouldReturnFirstError_WhenMultipleFunctionsFail(t *testing.T) {
	// Arrange
	sm := NewShutdownManager(newTestLogger())
	sm.Register(func(_ context.Context) error {
		return errors.New("first error")
	})
	sm.Register(func(_ context.Context) error {
		return errors.New("second error")
	})
	ctx := context.Background()

	// Act
	err := sm.Shutdown(ctx)

	// Assert
	require.Error(t, err)
	assert.Equal(t, "first error", err.Error(), "should return the first error encountered")
}

func TestShutdownManagerShutdown_ShouldReturnNil_WhenNoFunctionsRegistered(t *testing.T) {
	// Arrange
	sm := NewShutdownManager(newTestLogger())
	ctx := context.Background()

	// Act
	err := sm.Shutdown(ctx)

	// Assert
	require.NoError(t, err)
}

func TestShutdownManagerShutdown_ShouldStopEarly_WhenContextCancelled(t *testing.T) {
	// Arrange
	sm := NewShutdownManager(newTestLogger())
	callCount := 0
	sm.Register(func(_ context.Context) error {
		callCount++
		return nil
	})
	sm.Register(func(_ context.Context) error {
		callCount++
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Act
	err := sm.Shutdown(ctx)

	// Assert
	require.Error(t, err)
	assert.Equal(t, 0, callCount, "no functions should run when context is already cancelled")
}

func TestNewGracefulServer_ShouldReturnServer_WhenValidParamsProvided(t *testing.T) {
	// Arrange
	logger := newTestLogger()

	// Act
	srv := NewGracefulServer(nil, logger, 8080, 30*time.Second)

	// Assert
	require.NotNil(t, srv)
	assert.Equal(t, 8080, srv.port)
	assert.Equal(t, logger, srv.logger)
}
