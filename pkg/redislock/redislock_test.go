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

func TestNewLocker_ShouldReturnError_WhenClientIsNil(t *testing.T) {
	locker, err := NewLocker(nil, Config{})

	assert.ErrorIs(t, err, ErrNilClient)
	assert.Nil(t, locker)
}

func TestNewLocker_ShouldReturnLocker_WhenClientIsValid(t *testing.T) {
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

	locker, err := NewLocker(rc, Config{
		Prefix: "myapp",
		TTL:    30 * time.Second,
	})

	require.NoError(t, err)
	assert.NotNil(t, locker)
}

func TestAcquire_ShouldReturnLock_WhenKeyIsNotHeld(t *testing.T) {
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
	require.NotNil(t, lock)
	assert.NotEmpty(t, lock.Value())
	assert.Equal(t, "test:lock:resource-1", lock.Key())
}

func TestRelease_ShouldSucceed_WhenCallerHoldsLock(t *testing.T) {
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

	err = lock.Release(ctx)

	assert.NoError(t, err)

	lock2, err := locker.Acquire(ctx, "resource-1")
	require.NoError(t, err)
	assert.NotNil(t, lock2)
	_ = lock2.Release(ctx)
}

func TestAcquire_ShouldFail_WhenKeyIsAlreadyHeld(t *testing.T) {
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

	lock2, err := locker.Acquire(ctx, "contested")

	assert.ErrorIs(t, err, ErrLockUnavailable)
	assert.Nil(t, lock2)
}

func TestRelease_ShouldFail_WhenLockAlreadyReleased(t *testing.T) {
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

	err = lock1.Release(ctx)
	require.NoError(t, err)

	err = lock1.Release(ctx)

	assert.ErrorIs(t, err, ErrLockNotHeld)
}

func TestAcquire_ShouldAutoExpire_WhenTTLElapses(t *testing.T) {
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

	mr.FastForward(3 * time.Second)

	lock2, err := locker.Acquire(ctx, "expiring")
	require.NoError(t, err)
	require.NotNil(t, lock2)
	assert.NotEqual(t, lock1.Value(), lock2.Value())
	_ = lock2.Release(ctx)
}

func TestAcquire_ShouldRetryAndSucceed_WhenLockReleasedDuringRetry(t *testing.T) {
	locker, mr, cleanup := setupTestLocker(t, Config{
		Prefix:        "test",
		TTL:           2 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    100 * time.Millisecond,
	})
	defer cleanup()
	ctx := context.Background()

	lock1, err := locker.Acquire(ctx, "retry-key")
	require.NoError(t, err)

	go func() {
		time.Sleep(150 * time.Millisecond)
		_ = lock1.Release(ctx)
	}()

	lock2, err := locker.Acquire(ctx, "retry-key")

	require.NoError(t, err)
	require.NotNil(t, lock2)
	_ = lock2.Release(ctx)
	_ = mr // keep reference
}

func TestAcquire_ShouldFail_WhenAllRetriesExhausted(t *testing.T) {
	locker, _, cleanup := setupTestLocker(t, Config{
		Prefix:        "test",
		TTL:           10 * time.Second,
		RetryAttempts: 2,
		RetryDelay:    10 * time.Millisecond,
	})
	defer cleanup()
	ctx := context.Background()

	lock1, err := locker.Acquire(ctx, "no-release")
	require.NoError(t, err)
	defer func() { _ = lock1.Release(ctx) }()

	lock2, err := locker.Acquire(ctx, "no-release")

	assert.ErrorIs(t, err, ErrLockUnavailable)
	assert.Nil(t, lock2)
}

func TestLockKey_ShouldIncludePrefix_WhenPrefixConfigured(t *testing.T) {
	locker, _, cleanup := setupTestLocker(t, Config{
		Prefix:        "myapp",
		TTL:           10 * time.Second,
		RetryAttempts: 0,
		RetryDelay:    time.Millisecond,
	})
	defer cleanup()
	ctx := context.Background()

	lock, err := locker.Acquire(ctx, "resource")

	require.NoError(t, err)
	assert.Equal(t, "myapp:lock:resource", lock.Key())
	_ = lock.Release(ctx)
}

func TestLockKey_ShouldUseDefaultPrefix_WhenPrefixEmpty(t *testing.T) {
	locker, _, cleanup := setupTestLocker(t, Config{
		Prefix:        "",
		TTL:           10 * time.Second,
		RetryAttempts: 0,
		RetryDelay:    time.Millisecond,
	})
	defer cleanup()
	ctx := context.Background()

	lock, err := locker.Acquire(ctx, "resource")

	require.NoError(t, err)
	assert.Equal(t, "lock:resource", lock.Key())
	_ = lock.Release(ctx)
}

func TestAcquire_ShouldReturnUniqueLockValues_WhenAcquiredMultipleTimes(t *testing.T) {
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

func TestAcquire_ShouldReturnError_WhenContextCancelled(t *testing.T) {
	locker, _, cleanup := setupTestLocker(t, Config{
		Prefix:        "test",
		TTL:           10 * time.Second,
		RetryAttempts: 5,
		RetryDelay:    100 * time.Millisecond,
	})
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	lock1, err := locker.Acquire(ctx, "cancel-test")
	require.NoError(t, err)
	defer func() { _ = lock1.Release(context.Background()) }()

	cancel()

	lock2, err := locker.Acquire(ctx, "cancel-test")

	assert.Error(t, err)
	assert.Nil(t, lock2)
}
