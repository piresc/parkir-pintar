package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 100, cfg.RequestsPerSecond)
	assert.Equal(t, 200, cfg.BurstSize)
	assert.Equal(t, 5*time.Minute, cfg.CleanupInterval)
}

func TestNewStore(t *testing.T) {
	cfg := DefaultConfig()
	s := NewStore(cfg)
	require.NotNil(t, s)
	assert.NotNil(t, s.limiters)
	assert.NotNil(t, s.stopCh)
	s.Stop()
}

func TestNewStore_NoCleanup(t *testing.T) {
	cfg := Config{
		RequestsPerSecond: 10,
		BurstSize:         10,
		CleanupInterval:   0, // no cleanup goroutine
	}
	s := NewStore(cfg)
	require.NotNil(t, s)
	// stopCh is unbuffered so we can't close it without a reader — but the code
}

func TestAllow_UnderLimit(t *testing.T) {
	cfg := Config{
		RequestsPerSecond: 10,
		BurstSize:         5,
		CleanupInterval:   0,
	}
	s := NewStore(cfg)

	for i := 0; i < 5; i++ {
		assert.True(t, s.Allow("client1"), "request %d should be allowed", i)
	}
}

func TestAllow_ExceedsLimit(t *testing.T) {
	cfg := Config{
		RequestsPerSecond: 1,
		BurstSize:         2,
		CleanupInterval:   0,
	}
	s := NewStore(cfg)

	assert.True(t, s.Allow("client1"))
	assert.True(t, s.Allow("client1"))

	assert.False(t, s.Allow("client1"))
}

func TestAllow_DifferentKeys(t *testing.T) {
	cfg := Config{
		RequestsPerSecond: 1,
		BurstSize:         1,
		CleanupInterval:   0,
	}
	s := NewStore(cfg)

	assert.True(t, s.Allow("client1"))
	assert.True(t, s.Allow("client2"))

	assert.False(t, s.Allow("client1"))
	assert.False(t, s.Allow("client2"))
}

func TestCleanup_RemovesStaleEntries(t *testing.T) {
	cfg := Config{
		RequestsPerSecond: 10,
		BurstSize:         10,
		CleanupInterval:   50 * time.Millisecond,
	}
	s := NewStore(cfg)
	defer s.Stop()

	s.Allow("stale-client")

	s.mu.Lock()
	s.limiters["stale-client"].lastSeen = time.Now().Add(-1 * time.Hour)
	s.mu.Unlock()

	time.Sleep(150 * time.Millisecond)

	s.mu.Lock()
	_, exists := s.limiters["stale-client"]
	s.mu.Unlock()

	assert.False(t, exists, "stale entry should have been cleaned up")
}

func TestCleanup_KeepsFreshEntries(t *testing.T) {
	cfg := Config{
		RequestsPerSecond: 10,
		BurstSize:         10,
		CleanupInterval:   200 * time.Millisecond,
	}
	s := NewStore(cfg)
	defer s.Stop()

	s.Allow("fresh-client")

	time.Sleep(250 * time.Millisecond)

	s.mu.Lock()
	_, exists := s.limiters["fresh-client"]
	s.mu.Unlock()

	assert.True(t, exists, "fresh entry should not be cleaned up")
}

func TestStop_NoPanic(t *testing.T) {
	cfg := DefaultConfig()
	s := NewStore(cfg)
	assert.NotPanics(t, func() {
		s.Stop()
	})
}

func TestStop_NoCleanupNoPanic(t *testing.T) {
	cfg := Config{
		RequestsPerSecond: 10,
		BurstSize:         10,
		CleanupInterval:   0,
	}
	s := NewStore(cfg)
	assert.NotPanics(t, func() {
		s.Stop()
	})
}
