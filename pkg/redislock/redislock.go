// mutual exclusion across concurrent processes. It wraps github.com/bsm/redislock
package redislock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bsm/redislock"

	pkgredis "parkir-pintar/pkg/redis"
)

var (
	ErrLockUnavailable = errors.New("lock unavailable")
	ErrNilClient       = errors.New("redis client must not be nil")
	ErrLockNotHeld     = errors.New("lock not held by this owner")
)

type Config struct {
	Prefix        string
	TTL           time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}

type Locker struct {
	client *redislock.Client
	cfg    Config
}

type Lock struct {
	inner *redislock.Lock
	key   string
}

func NewLocker(client *pkgredis.Client, cfg Config) (*Locker, error) {
	if client == nil {
		return nil, ErrNilClient
	}
	return &Locker{
		client: redislock.New(client.GetClient()),
		cfg:    cfg,
	}, nil
}

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

func (l *Locker) lockKey(key string) string {
	if l.cfg.Prefix == "" {
		return "lock:" + key
	}
	return l.cfg.Prefix + ":lock:" + key
}

func (lock *Lock) Key() string {
	return lock.key
}

func (lock *Lock) Value() string {
	return lock.inner.Token()
}
