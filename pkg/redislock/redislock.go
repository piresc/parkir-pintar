// Package redislock provides a Redis-based distributed lock utility for
// mutual exclusion across concurrent processes. It uses SET NX with TTL
// for acquisition and a Lua script for atomic check-and-delete on release.
package redislock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"parkir-pintar/pkg/redis"
)

// releaseScript is the Lua script for atomic check-and-delete during lock release.
// It ensures only the holder of the lock (matching value) can delete the key.
const releaseScript = `if redis.call("get",KEYS[1]) == ARGV[1] then return redis.call("del",KEYS[1]) else return 0 end`

// Sentinel errors for lock operations.
var (
	ErrLockUnavailable = errors.New("lock unavailable")
	ErrNilClient       = errors.New("redis client must not be nil")
	ErrLockNotHeld     = errors.New("lock not held by this owner")
)

// Config holds configuration for the distributed lock.
type Config struct {
	Prefix        string
	TTL           time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}

// Locker manages distributed lock acquisition using a Redis client.
type Locker struct {
	client *redis.Client
	cfg    Config
}

// Lock represents an acquired distributed lock with a unique owner value.
type Lock struct {
	key    string
	value  string
	client *redis.Client
}

// NewLocker creates a new Locker with the given Redis client and configuration.
// It returns an error if the client is nil.
func NewLocker(client *redis.Client, cfg Config) (*Locker, error) {
	if client == nil {
		return nil, ErrNilClient
	}
	return &Locker{
		client: client,
		cfg:    cfg,
	}, nil
}

// Acquire attempts to acquire a distributed lock on the given key.
// It retries with backoff according to the Locker's configuration.
// Returns a Lock handle on success, or ErrLockUnavailable if the lock
// cannot be acquired after all retry attempts.
func (l *Locker) Acquire(ctx context.Context, key string) (*Lock, error) {
	lockKey := l.lockKey(key)
	lockValue := uuid.New().String()

	for attempt := 0; attempt <= l.cfg.RetryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled while acquiring lock: %w", ctx.Err())
			case <-time.After(l.cfg.RetryDelay):
			}
		}

		ok, err := l.client.SetNX(ctx, lockKey, lockValue, l.cfg.TTL)
		if err != nil {
			return nil, fmt.Errorf("redis error during lock acquisition: %w", err)
		}
		if ok {
			return &Lock{
				key:    lockKey,
				value:  lockValue,
				client: l.client,
			}, nil
		}
	}

	return nil, ErrLockUnavailable
}

// Release releases the lock only if the caller holds it, using a Lua script
// for atomic check-and-delete to prevent race conditions.
func (lock *Lock) Release(ctx context.Context) error {
	result, err := lock.client.GetClient().Eval(ctx, releaseScript, []string{lock.key}, lock.value).Int64()
	if err != nil {
		return fmt.Errorf("redis error during lock release: %w", err)
	}
	if result == 0 {
		return ErrLockNotHeld
	}
	return nil
}

// lockKey builds the full Redis key for a lock using the configured prefix.
func (l *Locker) lockKey(key string) string {
	if l.cfg.Prefix == "" {
		return "lock:" + key
	}
	return l.cfg.Prefix + ":lock:" + key
}

// Key returns the Redis key of the lock.
func (lock *Lock) Key() string {
	return lock.key
}

// Value returns the unique owner value of the lock.
func (lock *Lock) Value() string {
	return lock.value
}
