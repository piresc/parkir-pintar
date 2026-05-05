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
func newTestRedisClient(t *testing.T) (*redis.RedisClient, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)

	host, portStr, err := net.SplitHostPort(mr.Addr())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	rc, err := redis.NewRedisClient(config.RedisConfig{
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
		assert.NotEmpty(t, lock1.Value(), "lock value must be non-empty UUID")

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
// For any two distinct lock values V1 and V2 on the same key, releasing the
// lock with V1 SHALL only succeed if V1 is the current holder; attempting to
// release with V2 when V1 holds the lock SHALL leave the lock intact.
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

		// Acquire the lock — this is V1
		lock1, err := locker.Acquire(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		// Create a fake lock with a different value (V2) on the same key
		fakeLock := &Lock{
			key:    lock1.Key(),
			value:  "fake-value-that-does-not-match",
			client: rc,
		}

		// Act — releasing with V2 (wrong owner) should fail
		err = fakeLock.Release(ctx)
		assert.ErrorIs(t, err, ErrLockNotHeld,
			"releasing with wrong value must return ErrLockNotHeld")

		// Assert — the lock is still intact (V1 still holds it)
		lock2, err := locker.Acquire(ctx, key)
		assert.ErrorIs(t, err, ErrLockUnavailable,
			"lock must still be held after failed release with wrong value")
		assert.Nil(t, lock2)

		// Act — releasing with V1 (correct owner) should succeed
		err = lock1.Release(ctx)
		assert.NoError(t, err, "releasing with correct value must succeed")

		// Assert — lock is now free
		lock3, err := locker.Acquire(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		_ = lock3.Release(ctx)
	})
}
