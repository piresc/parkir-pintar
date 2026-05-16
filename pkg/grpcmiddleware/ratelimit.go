package grpcmiddleware

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RateLimitConfig holds configuration for the gRPC rate limiter.
type RateLimitConfig struct {
	// RequestsPerSecond is the token refill rate per second per client.
	RequestsPerSecond int
	// BurstSize is the maximum token bucket capacity (max burst).
	BurstSize int
	// CleanupInterval is how often stale per-client buckets are removed.
	CleanupInterval time.Duration
}

// rateLimitStore is a thread-safe in-memory store of per-client rate limiters.
type rateLimitStore struct {
	mu       sync.Mutex
	limiters map[string]*rateLimitEntry
	cfg      RateLimitConfig
	stopCh   chan struct{}
}

type rateLimitEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newRateLimitStore(cfg RateLimitConfig) *rateLimitStore {
	store := &rateLimitStore{
		limiters: make(map[string]*rateLimitEntry),
		cfg:      cfg,
		stopCh:   make(chan struct{}),
	}

	if cfg.CleanupInterval > 0 {
		go store.cleanup(cfg.CleanupInterval)
	}

	return store
}

func (s *rateLimitStore) allow(ctx context.Context) bool {
	key := "unknown"
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		key = p.Addr.String()
	}

	s.mu.Lock()
	entry, exists := s.limiters[key]
	if !exists {
		entry = &rateLimitEntry{
			limiter:  rate.NewLimiter(rate.Limit(s.cfg.RequestsPerSecond), s.cfg.BurstSize),
			lastSeen: time.Now(),
		}
		s.limiters[key] = entry
	} else {
		entry.lastSeen = time.Now()
	}
	s.mu.Unlock()

	return entry.limiter.Allow()
}

func (s *rateLimitStore) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for key, entry := range s.limiters {
				if now.Sub(entry.lastSeen) > 2*interval {
					delete(s.limiters, key)
				}
			}
			s.mu.Unlock()
		case <-s.stopCh:
			return
		}
	}
}

// Stop terminates the background cleanup goroutine.
func (s *rateLimitStore) Stop() {
	close(s.stopCh)
}

// RateLimitUnaryInterceptor returns a grpc.UnaryServerInterceptor that
// enforces per-client rate limiting using golang.org/x/time/rate (token bucket).
// The client is identified by its peer address from the gRPC transport.
// Requests exceeding the limit receive a gRPC ResourceExhausted status code.
func (i *Interceptors) RateLimitUnaryInterceptor(cfg RateLimitConfig) grpc.UnaryServerInterceptor {
	store := newRateLimitStore(cfg)

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if !store.allow(ctx) {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}
