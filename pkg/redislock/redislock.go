// Package redislock provides a Redis-based distributed lock utility for
// mutual exclusion across concurrent processes. It wraps github.com/bsm/redislock
// which implements the Redlock algorithm with proper fencing.
package redislock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bsm/redislock"

	pkgredis "parkir-pintar/pkg/redis"
)

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
	client *redislock.Client
	cfg    Config
}

// Lock represents an acquired distributed lock.
type Lock struct {
	inner *redislock.Lock
	key   string
}

// NewLocker creates a new Locker with the given Redis client and configuration.
// It returns an error if the client is nil.
func NewLocker(client *pkgredis.Client, cfg Config) (*Locker, error) {
	if client == nil {
		return nil, ErrNilClient
	}
	return &Locker{
		client: redislock.New(client.GetClient()),
		cfg:    cfg,
	}, nil
}

// Acquire attempts to acquire a distributed lock on the given key.
// It retries according to the Locker's configuration.
// Returns a Lock handle on success, or ErrLockUnavailable if the lock
// cannot be acquired after all retry attempts.
func (l *Locker) Acquire(ctx context.Context, key string) (*Lock, error) {
	lockKey := l.lockKey(key)

	opts := &redislock.Options{}
	if l.cfg.RetryAttempts > 0 && l.cfg.RetryDelay > 0 {
		opts.RetryStrategy = redislock.LimitRetry(
			redislock.LinearBackoff(l.cfg.RetryDelay),
			l.cfg.RetryAttempts,
		)
	}

	lock, err := l.client.Obtain(ctx, lockKey, l.cfg.TTL, opts)
	if err != nil {
		if errors.Is(err, redislock.ErrNotObtained) {
			return nil, ErrLockUnavailable
		}
		return nil, fmt.Errorf("redis error during lock acquisition: %w", err)
	}

	return &Lock{
		inner: lock,
		key:   lockKey,
	}, nil
}

// Release releases the lock only if the caller holds it.
func (lock *Lock) Release(ctx context.Context) error {
	err := lock.inner.Release(ctx)
	if err != nil {
		if errors.Is(err, redislock.ErrLockNotHeld) {
			return ErrLockNotHeld
		}
		return fmt.Errorf("redis error during lock release: %w", err)
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
	return lock.inner.Token()
}
