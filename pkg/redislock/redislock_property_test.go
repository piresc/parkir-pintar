// Package redislock property-based tests
//
// Best practices applied (from Go testing standards):
// - Descriptive names: TestProperty[N]_[PropertyName]
// - AAA (Arrange-Act-Assert) pattern
// - miniredis/v2 for in-memory Redis testing without real server
// - pgregory.net/rapid for property-based testing with minimum 100 iterations
// - testify assertions for clear failure messages
// - Redis setup happens in outer *testing.T scope (rapid.T does not implement testing.TB)
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
	"pgregory.net/rapid"

	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/redis"
)

// newTestRedisClient creates a miniredis server and a RedisClient connected to it.
func newTestRedisClient(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
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

	t.Cleanup(func() {
		_ = rc.Close()
		mr.Close()
	})
	return rc, mr
}

// Feature: grpc-jwt-pkg-integration, Property 12: Lock acquisition mutual exclusion
// **Validates: Requirements 10.1, 10.2, 10.3**
//
// For any lock key, acquiring the lock SHALL succeed if the key is not held,
// and SHALL fail with an error if the key is already held by another caller.
// A successful acquisition SHALL return a handle with a unique value.
func TestProperty12_LockAcquisitionMutualExclusion(t *testing.T) {
	rc, _ := newTestRedisClient(t)

	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()
		key := rapid.StringMatching(`[a-zA-Z0-9]{1,20}`).Draw(t, "lockKey")

		locker, err := NewLocker(rc, Config{
			Prefix:        "prop12",
			TTL:           10 * time.Second,
			RetryAttempts: 0, // no retries — fail immediately on contention
			RetryDelay:    time.Millisecond,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Act — first acquire should succeed
		lock1, err := locker.Acquire(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		// Assert — lock handle has a non-empty unique value
		assert.NotEmpty(t, lock1.Value(), "lock value must be non-empty")

		// Act — second acquire on the same key should fail (mutual exclusion)
		lock2, err := locker.Acquire(ctx, key)
		assert.ErrorIs(t, err, ErrLockUnavailable, "second acquire must fail when key is held")
		assert.Nil(t, lock2)

		// Cleanup — release the first lock
		if releaseErr := lock1.Release(ctx); releaseErr != nil {
			t.Fatal(releaseErr)
		}

		// Act — after release, acquire should succeed again
		lock3, err := locker.Acquire(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		// Assert — new lock has a different unique value
		assert.NotEqual(t, lock1.Value(), lock3.Value(),
			"each acquisition must produce a unique value")

		// Cleanup
		_ = lock3.Release(ctx)
	})
}

// Feature: grpc-jwt-pkg-integration, Property 13: Lock safe release
// **Validates: Requirements 10.4**
//
// For any lock key, releasing the lock SHALL only succeed if the caller holds it.
// A double release SHALL fail with ErrLockNotHeld, proving the lock was properly
// freed on the first release and cannot be released again.
func TestProperty13_LockSafeRelease(t *testing.T) {
	rc, _ := newTestRedisClient(t)

	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()
		key := rapid.StringMatching(`[a-zA-Z0-9]{1,20}`).Draw(t, "lockKey")

		locker, err := NewLocker(rc, Config{
			Prefix:        "prop13",
			TTL:           10 * time.Second,
			RetryAttempts: 0,
			RetryDelay:    time.Millisecond,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Acquire the lock
		lock1, err := locker.Acquire(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		// Act — releasing with correct owner should succeed
		err = lock1.Release(ctx)
		assert.NoError(t, err, "releasing with correct owner must succeed")

		// Assert — double release should fail (lock no longer held)
		err = lock1.Release(ctx)
		assert.ErrorIs(t, err, ErrLockNotHeld,
			"double release must return ErrLockNotHeld")

		// Assert — lock is now free, new acquire should succeed
		lock2, err := locker.Acquire(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		// The new lock should have a different token
		assert.NotEqual(t, lock1.Value(), lock2.Value(),
			"new acquisition must have different value")

		_ = lock2.Release(ctx)
	})
}
