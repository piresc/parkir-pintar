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

		lock1, err := locker.Acquire(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotEmpty(t, lock1.Value(), "lock value must be non-empty")

		lock2, err := locker.Acquire(ctx, key)
		assert.ErrorIs(t, err, ErrLockUnavailable, "second acquire must fail when key is held")
		assert.Nil(t, lock2)

		if releaseErr := lock1.Release(ctx); releaseErr != nil {
			t.Fatal(releaseErr)
		}

		lock3, err := locker.Acquire(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotEqual(t, lock1.Value(), lock3.Value(),
			"each acquisition must produce a unique value")

		_ = lock3.Release(ctx)
	})
}

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

		lock1, err := locker.Acquire(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		err = lock1.Release(ctx)
		assert.NoError(t, err, "releasing with correct owner must succeed")

		err = lock1.Release(ctx)
		assert.ErrorIs(t, err, ErrLockNotHeld,
			"double release must return ErrLockNotHeld")

		lock2, err := locker.Acquire(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		assert.NotEqual(t, lock1.Value(), lock2.Value(),
			"new acquisition must have different value")

		_ = lock2.Release(ctx)
	})
}
