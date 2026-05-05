package grpcmiddleware

import (
	"context"
	"sync"
	"time"

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

// grpcTokenBucket implements a simple token bucket rate limiter.
type grpcTokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

func newGRPCTokenBucket(maxTokens float64, refillRate float64) *grpcTokenBucket {
	return &grpcTokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// allow checks if a request is allowed and consumes a token if so.
func (tb *grpcTokenBucket) allow() bool {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// grpcRateLimitStore is a thread-safe in-memory store of per-key token buckets.
type grpcRateLimitStore struct {
	mu      sync.Mutex
	buckets map[string]*grpcTokenBucket
	cfg     RateLimitConfig
	stopCh  chan struct{}
}

func newGRPCRateLimitStore(cfg RateLimitConfig) *grpcRateLimitStore {
	store := &grpcRateLimitStore{
		buckets: make(map[string]*grpcTokenBucket),
		cfg:     cfg,
		stopCh:  make(chan struct{}),
	}

	if cfg.CleanupInterval > 0 {
		go store.cleanup(cfg.CleanupInterval)
	}

	return store
}

func (s *grpcRateLimitStore) allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	bucket, exists := s.buckets[key]
	if !exists {
		bucket = newGRPCTokenBucket(float64(s.cfg.BurstSize), float64(s.cfg.RequestsPerSecond))
		s.buckets[key] = bucket
	}

	return bucket.allow()
}

func (s *grpcRateLimitStore) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for key, bucket := range s.buckets {
				if now.Sub(bucket.lastRefill) > 2*interval {
					delete(s.buckets, key)
				}
			}
			s.mu.Unlock()
		case <-s.stopCh:
			return
		}
	}
}

// Stop terminates the background cleanup goroutine.
func (s *grpcRateLimitStore) Stop() {
	close(s.stopCh)
}

// RateLimitUnaryInterceptor returns a grpc.UnaryServerInterceptor that
// enforces per-client rate limiting using a token bucket algorithm. The client
// is identified by its peer address from the gRPC transport. Requests
// exceeding the limit receive a gRPC ResourceExhausted status code with the
// message "rate limit exceeded".
func (i *Interceptors) RateLimitUnaryInterceptor(cfg RateLimitConfig) grpc.UnaryServerInterceptor {
	store := newGRPCRateLimitStore(cfg)

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		key := "unknown"
		if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
			key = p.Addr.String()
		}

		if !store.allow(key) {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}
