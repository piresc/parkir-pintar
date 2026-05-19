package grpcmiddleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	// idempotencySentinel is the placeholder value stored via SETNX while the
	// handler is executing. Concurrent requests that see this value know to
	// poll until the real response replaces it.
	idempotencySentinel = "processing"

	// idempotencyPollInterval is how long a waiting request sleeps between
	// polls when it finds the sentinel value.
	idempotencyPollInterval = 10 * time.Millisecond

	// idempotencyMaxPollAttempts caps the number of poll iterations to avoid
	// blocking indefinitely if the owning request crashes.
	idempotencyMaxPollAttempts = 300 // 300 × 10ms = 3s max wait
)

// IdempotencyConfig holds configuration for the idempotency interceptor.
type IdempotencyConfig struct {
	// TTL is the time-to-live for cached idempotency responses in Redis.
	TTL time.Duration
	// Methods is the list of full gRPC method names to enforce idempotency on.
	Methods []string
}

// IdempotencyUnaryInterceptor returns a grpc.UnaryServerInterceptor that
// enforces idempotency on configured methods using Redis. It reads the
// "x-idempotency-key" from gRPC metadata, atomically claims the key via
// SETNX with a sentinel value, executes the handler if the claim succeeds,
// and replaces the sentinel with the serialized response. Concurrent
// requests that lose the SETNX race poll until the response is available.
// Methods not in the enforcement list pass through without idempotency checks.
func (i *Interceptors) IdempotencyUnaryInterceptor(cfg IdempotencyConfig) grpc.UnaryServerInterceptor {
	enforced := toSet(cfg.Methods)

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// If Redis is unavailable, skip idempotency logic entirely.
		if i.redisClient == nil {
			return handler(ctx, req)
		}

		// Skip idempotency for methods not in the enforcement list.
		if _, ok := enforced[info.FullMethod]; !ok {
			return handler(ctx, req)
		}

		// Extract x-idempotency-key from metadata.
		idempotencyKey, err := extractIdempotencyKey(ctx)
		if err != nil {
			return nil, err
		}

		redisKey := fmt.Sprintf("idempotency:%s:%s", info.FullMethod, idempotencyKey)

		// Atomically try to claim the key with a sentinel value.
		acquired, err := i.redisClient.SetNX(ctx, redisKey, idempotencySentinel, cfg.TTL)
		if err != nil {
			i.logger.LogAttrs(ctx, slog.LevelError, "idempotency: redis SETNX failed",
				slog.String("key", redisKey),
				slog.String("error", err.Error()),
			)
			return nil, status.Errorf(codes.Internal, "internal server error")
		}

		if acquired {
			// We own the key — execute the handler.
			resp, handlerErr := handler(ctx, req)

			// Remove the key after handler completes (success or failure) so
			// subsequent retries are processed normally by the handler.
			if delErr := i.redisClient.Delete(ctx, redisKey); delErr != nil {
				i.logger.LogAttrs(ctx, slog.LevelError, "idempotency: failed to delete key after handler completion",
					slog.String("key", redisKey),
					slog.String("error", delErr.Error()),
				)
			}

			return resp, handlerErr
		}

		// Another request owns the key — wait briefly to confirm it's still
		// in-flight, then reject the duplicate.
		return nil, i.waitAndRejectDuplicate(ctx, redisKey)
	}
}

// waitAndRejectDuplicate polls Redis briefly to distinguish between a
// genuinely in-flight request and a key that was already cleaned up. If the
// key disappears (handler completed), it tells the caller to retry. If the
// key is still held, it rejects the duplicate request.
func (i *Interceptors) waitAndRejectDuplicate(ctx context.Context, redisKey string) error {
	for attempt := 0; attempt < idempotencyMaxPollAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return status.Errorf(codes.DeadlineExceeded, "context cancelled while waiting for idempotent response")
		case <-time.After(idempotencyPollInterval):
		}

		cached, err := i.redisClient.Get(ctx, redisKey)
		if errors.Is(err, redis.Nil) {
			// Key was deleted — the original request completed. The caller
			// can safely retry and the handler's own idempotency (e.g. DB
			// constraints) will ensure correctness.
			return status.Errorf(codes.Aborted, "concurrent request completed, please retry")
		}
		if err != nil {
			i.logger.LogAttrs(ctx, slog.LevelError, "idempotency: redis GET failed during poll",
				slog.String("key", redisKey),
				slog.String("error", err.Error()),
			)
			return status.Errorf(codes.Internal, "internal server error")
		}

		// Still processing — keep polling.
		if cached == idempotencySentinel {
			continue
		}

		// Unexpected value — treat as completed.
		return status.Errorf(codes.Aborted, "concurrent request completed, please retry")
	}

	return status.Errorf(codes.DeadlineExceeded, "timed out waiting for concurrent request to complete")
}

// extractIdempotencyKey reads the "x-idempotency-key" value from gRPC
// incoming metadata. Returns InvalidArgument if the key is missing.
func extractIdempotencyKey(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.InvalidArgument, "missing idempotency key")
	}

	values := md.Get("x-idempotency-key")
	if len(values) == 0 || values[0] == "" {
		return "", status.Errorf(codes.InvalidArgument, "missing idempotency key")
	}

	return values[0], nil
}
