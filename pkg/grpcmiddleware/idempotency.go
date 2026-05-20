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
	// handler is executing. Concurrent requests that see this value know to
	idempotencySentinel = "processing"

	idempotencyPollInterval = 10 * time.Millisecond

	// idempotencyMaxPollAttempts caps the number of poll iterations to avoid
	idempotencyMaxPollAttempts = 300 // 300 × 10ms = 3s max wait
)

type IdempotencyConfig struct {
	TTL time.Duration
	Methods []string
}

// and replaces the sentinel with the serialized response. Concurrent
func (i *Interceptors) IdempotencyUnaryInterceptor(cfg IdempotencyConfig) grpc.UnaryServerInterceptor {
	enforced := toSet(cfg.Methods)

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if i.redisClient == nil {
			return handler(ctx, req)
		}

		if _, ok := enforced[info.FullMethod]; !ok {
			return handler(ctx, req)
		}

		idempotencyKey, err := extractIdempotencyKey(ctx)
		if err != nil {
			return nil, err
		}

		redisKey := fmt.Sprintf("idempotency:%s:%s", info.FullMethod, idempotencyKey)

		acquired, err := i.redisClient.SetNX(ctx, redisKey, idempotencySentinel, cfg.TTL)
		if err != nil {
			i.logger.LogAttrs(ctx, slog.LevelError, "idempotency: redis SETNX failed",
				slog.String("key", redisKey),
				slog.String("error", err.Error()),
			)
			return nil, status.Errorf(codes.Internal, "internal server error")
		}

		if acquired {
			resp, handlerErr := handler(ctx, req)

			if delErr := i.redisClient.Delete(ctx, redisKey); delErr != nil {
				i.logger.LogAttrs(ctx, slog.LevelError, "idempotency: failed to delete key after handler completion",
					slog.String("key", redisKey),
					slog.String("error", delErr.Error()),
				)
			}

			return resp, handlerErr
		}

		return nil, i.waitAndRejectDuplicate(ctx, redisKey)
	}
}

func (i *Interceptors) waitAndRejectDuplicate(ctx context.Context, redisKey string) error {
	for attempt := 0; attempt < idempotencyMaxPollAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return status.Errorf(codes.DeadlineExceeded, "context cancelled while waiting for idempotent response")
		case <-time.After(idempotencyPollInterval):
		}

		cached, err := i.redisClient.Get(ctx, redisKey)
		if errors.Is(err, redis.Nil) {
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

		if cached == idempotencySentinel {
			continue
		}

		return status.Errorf(codes.Aborted, "concurrent request completed, please retry")
	}

	return status.Errorf(codes.DeadlineExceeded, "timed out waiting for concurrent request to complete")
}

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
