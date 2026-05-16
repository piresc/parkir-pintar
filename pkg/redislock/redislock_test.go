// Package redislock unit tests
//
// Best practices applied (from Go testing standards):
// - Descriptive names: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA (Arrange-Act-Assert) pattern
// - miniredis/v2 for in-memory Redis testing without real server
// - testify assertions for clear failure messages
// - Each test gets a fresh miniredis instance for isolation
// - Table-driven tests for multiple scenarios
// - Test both success and error/edge cases
package redislock

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/redis"
)

// setupTestLocker creates a miniredis server, RedisClient, and Locker for testing.
func setupTestLocker(t *testing.T, cfg Config) (*Locker, *miniredis.Miniredis, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)

	host, portStr, err := net.SplitHostPort(mr.Addr())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	rc, err := redis.NewClient(config.RedisConfig{
		Host:     host,
		Port:     port,
		Password: "",
		DB:       0,
		PoolSize: 5,
	})
	require.NoError(t, err)

	locker, err := NewLocker(rc, cfg)
	require.NoError(t, err)

	cleanup := func() {
		_ = rc.Close()
		mr.Close()
	}
	return locker, mr, cleanup
}

// --- NewLocker constructor tests ---

func TestNewLocker_ShouldReturnError_WhenClientIsNil(t *testing.T) {
	// Arrange & Act
	locker, err := NewLocker(nil, Config{})

	// Assert
	assert.ErrorIs(t, err, ErrNilClient)
	assert.Nil(t, locker)
}

func TestNewLocker_ShouldReturnLocker_WhenClientIsValid(t *testing.T) {
	// Arrange
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	host, portStr, err := net.SplitHostPort(mr.Addr())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	rc, err := redis.NewClient(config.RedisConfig{
		Host: host, Port: port, PoolSize: 5,
	})
	require.NoError(t, err)
	defer func() { _ = rc.Close() }()

	// Act
	locker, err := NewLocker(rc, Config{
		Prefix: "myapp",
		TTL:    30 * time.Second,
	})

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, locker)
}

// --- Acquire/Release happy path (Requirements 10.1, 10.2) ---

func TestAcquire_ShouldReturnLock_WhenKeyIsNotHeld(t *testing.T) {
	// Arrange
	locker, _, cleanup := setupTestLocker(t, Config{
		Prefix:        "test",
		TTL:           10 * time.Second,
		RetryAttempts: 0,
		RetryDelay:    time.Millisecond,
	})
	defer cleanup()
	ctx := context.Background()

	// Act
	lock, err := locker.Acquire(ctx, "resource-1")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, lock)
	assert.NotEmpty(t, lock.Value())
	assert.Equal(t, "test:lock:resource-1", lock.Key())
}

func TestRelease_ShouldSucceed_WhenCallerHoldsLock(t *testing.T) {
	// Arrange
	locker, _, cleanup := setupTestLocker(t, Config{
		Prefix:        "test",
		TTL:           10 * time.Second,
		RetryAttempts: 0,
		RetryDelay:    time.Millisecond,
	})
	defer cleanup()
	ctx := context.Background()

	lock, err := locker.Acquire(ctx, "resource-1")
	require.NoError(t, err)

	// Act
	err = lock.Release(ctx)

	// Assert
	assert.NoError(t, err)

	// Verify lock is released — re-acquire should succeed
	lock2, err := locker.Acquire(ctx, "resource-1")
	require.NoError(t, err)
	assert.NotNil(t, lock2)
	_ = lock2.Release(ctx)
}

// --- Contention: double acquire fails (Requirement 10.3) ---

func TestAcquire_ShouldFail_WhenKeyIsAlreadyHeld(t *testing.T) {
	// Arrange
	locker, _, cleanup := setupTestLocker(t, Config{
		Prefix:        "test",
		TTL:           10 * time.Second,
		RetryAttempts: 0,
		RetryDelay:    time.Millisecond,
	})
	defer cleanup()
	ctx := context.Background()

	lock1, err := locker.Acquire(ctx, "contested")
	require.NoError(t, err)
	defer func() { _ = lock1.Release(ctx) }()

	// Act — second acquire on same key
	lock2, err := locker.Acquire(ctx, "contested")

	// Assert
	assert.ErrorIs(t, err, ErrLockUnavailable)
	assert.Nil(t, lock2)
}

// --- Safe release: double release fails (Requirement 10.4, 10.5) ---

func TestRelease_ShouldFail_WhenLockAlreadyReleased(t *testing.T) {
	// Arrange
	locker, _, cleanup := setupTestLocker(t, Config{
		Prefix:        "test",
		TTL:           10 * time.Second,
		RetryAttempts: 0,
		RetryDelay:    time.Millisecond,
	})
	defer cleanup()
	ctx := context.Background()

	lock1, err := locker.Acquire(ctx, "safe-release")
	require.NoError(t, err)

	// Release once (should succeed)
	err = lock1.Release(ctx)
	require.NoError(t, err)

	// Act — release again (lock no longer held)
	err = lock1.Release(ctx)

	// Assert
	assert.ErrorIs(t, err, ErrLockNotHeld)
}

// --- TTL expiry (Requirement 10.7) ---

func TestAcquire_ShouldAutoExpire_WhenTTLElapses(t *testing.T) {
	// Arrange
	locker, mr, cleanup := setupTestLocker(t, Config{
		Prefix:        "test",
		TTL:           2 * time.Second,
		RetryAttempts: 0,
		RetryDelay:    time.Millisecond,
	})
	defer cleanup()
	ctx := context.Background()

	lock1, err := locker.Acquire(ctx, "expiring")
	require.NoError(t, err)
	require.NotNil(t, lock1)

	// Act — fast-forward miniredis time past TTL
	mr.FastForward(3 * time.Second)

	// Assert — lock should have expired, new acquire should succeed
	lock2, err := locker.Acquire(ctx, "expiring")
	require.NoError(t, err)
	require.NotNil(t, lock2)
	assert.NotEqual(t, lock1.Value(), lock2.Value())
	_ = lock2.Release(ctx)
}

// --- Retry with backoff (Requirement 10.6) ---

func TestAcquire_ShouldRetryAndSucceed_WhenLockReleasedDuringRetry(t *testing.T) {
	// Arrange
	locker, mr, cleanup := setupTestLocker(t, Config{
		Prefix:        "test",
		TTL:           2 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    100 * time.Millisecond,
	})
	defer cleanup()
	ctx := context.Background()

	// Hold the lock initially
	lock1, err := locker.Acquire(ctx, "retry-key")
	require.NoError(t, err)

	// Release the lock after a short delay in a goroutine so retry can succeed
	go func() {
		time.Sleep(150 * time.Millisecond)
		_ = lock1.Release(ctx)
	}()

	// Act — second locker retries and should eventually succeed
	lock2, err := locker.Acquire(ctx, "retry-key")

	// Assert
	require.NoError(t, err)
	require.NotNil(t, lock2)
	_ = lock2.Release(ctx)
	_ = mr // keep reference
}

func TestAcquire_ShouldFail_WhenAllRetriesExhausted(t *testing.T) {
	// Arrange
	locker, _, cleanup := setupTestLocker(t, Config{
		Prefix:        "test",
		TTL:           10 * time.Second,
		RetryAttempts: 2,
		RetryDelay:    10 * time.Millisecond,
	})
	defer cleanup()
	ctx := context.Background()

	// Hold the lock permanently
	lock1, err := locker.Acquire(ctx, "no-release")
	require.NoError(t, err)
	defer func() { _ = lock1.Release(ctx) }()

	// Act — second acquire with retries should still fail
	lock2, err := locker.Acquire(ctx, "no-release")

	// Assert
	assert.ErrorIs(t, err, ErrLockUnavailable)
	assert.Nil(t, lock2)
}

// --- Lock key prefix (Requirement 10.6) ---

func TestLockKey_ShouldIncludePrefix_WhenPrefixConfigured(t *testing.T) {
	// Arrange
	locker, _, cleanup := setupTestLocker(t, Config{
		Prefix:        "myapp",
		TTL:           10 * time.Second,
		RetryAttempts: 0,
		RetryDelay:    time.Millisecond,
	})
	defer cleanup()
	ctx := context.Background()

	// Act
	lock, err := locker.Acquire(ctx, "resource")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "myapp:lock:resource", lock.Key())
	_ = lock.Release(ctx)
}

func TestLockKey_ShouldUseDefaultPrefix_WhenPrefixEmpty(t *testing.T) {
	// Arrange
	locker, _, cleanup := setupTestLocker(t, Config{
		Prefix:        "",
		TTL:           10 * time.Second,
		RetryAttempts: 0,
		RetryDelay:    time.Millisecond,
	})
	defer cleanup()
	ctx := context.Background()

	// Act
	lock, err := locker.Acquire(ctx, "resource")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "lock:resource", lock.Key())
	_ = lock.Release(ctx)
}

// --- Unique lock values (Requirement 10.2) ---

func TestAcquire_ShouldReturnUniqueLockValues_WhenAcquiredMultipleTimes(t *testing.T) {
	// Arrange
	locker, _, cleanup := setupTestLocker(t, Config{
		Prefix:        "test",
		TTL:           10 * time.Second,
		RetryAttempts: 0,
		RetryDelay:    time.Millisecond,
	})
	defer cleanup()
	ctx := context.Background()

	values := make(map[string]bool)
	for i := 0; i < 10; i++ {
		lock, err := locker.Acquire(ctx, "unique-test")
		require.NoError(t, err)
		assert.False(t, values[lock.Value()], "lock value must be unique across acquisitions")
		values[lock.Value()] = true
		err = lock.Release(ctx)
		require.NoError(t, err)
	}
}

// --- Context cancellation ---

func TestAcquire_ShouldReturnError_WhenContextCancelled(t *testing.T) {
	// Arrange
	locker, _, cleanup := setupTestLocker(t, Config{
		Prefix:        "test",
		TTL:           10 * time.Second,
		RetryAttempts: 5,
		RetryDelay:    100 * time.Millisecond,
	})
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	// Hold the lock so retries are needed
	lock1, err := locker.Acquire(ctx, "cancel-test")
	require.NoError(t, err)
	defer func() { _ = lock1.Release(context.Background()) }()

	// Cancel context immediately
	cancel()

	// Act
	lock2, err := locker.Acquire(ctx, "cancel-test")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, lock2)
}
