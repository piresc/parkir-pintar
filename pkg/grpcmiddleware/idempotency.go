package grpcmiddleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
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
			if handlerErr != nil {
				// Handler failed — remove the sentinel so retries can proceed.
				if delErr := i.redisClient.Delete(ctx, redisKey); delErr != nil {
					i.logger.LogAttrs(ctx, slog.LevelError, "idempotency: failed to delete sentinel after handler error",
						slog.String("key", redisKey),
						slog.String("error", delErr.Error()),
					)
				}
				return resp, handlerErr
			}

		// Serialize and replace sentinel with the actual response.
		var data []byte
		var marshalErr error
		if pm, ok := resp.(proto.Message); ok {
			data, marshalErr = protojson.Marshal(pm)
		} else {
			data, marshalErr = json.Marshal(resp)
		}
		if marshalErr != nil {
			i.logger.LogAttrs(ctx, slog.LevelError, "idempotency: failed to marshal response",
				slog.String("key", redisKey),
				slog.String("error", marshalErr.Error()),
			)
			return resp, nil
		}

			if setErr := i.redisClient.Set(ctx, redisKey, string(data), cfg.TTL); setErr != nil {
				i.logger.LogAttrs(ctx, slog.LevelError, "idempotency: redis SET failed",
					slog.String("key", redisKey),
					slog.String("error", setErr.Error()),
				)
			}

			return resp, nil
		}

		// Another request owns the key — poll until the response is ready.
		return i.pollForCachedResponse(ctx, redisKey)
	}
}

// pollForCachedResponse waits for the sentinel value to be replaced with the
// actual serialized response. It polls Redis at short intervals and returns
// the deserialized response once available.
func (i *Interceptors) pollForCachedResponse(ctx context.Context, redisKey string) (interface{}, error) {
	for attempt := 0; attempt < idempotencyMaxPollAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, status.Errorf(codes.DeadlineExceeded, "context cancelled while waiting for idempotent response")
		case <-time.After(idempotencyPollInterval):
		}

		cached, err := i.redisClient.Get(ctx, redisKey)
		if errors.Is(err, redis.Nil) {
			// Key was deleted (handler error) — let the caller retry.
			return nil, status.Errorf(codes.Aborted, "concurrent request failed, please retry")
		}
		if err != nil {
			i.logger.LogAttrs(ctx, slog.LevelError, "idempotency: redis GET failed during poll",
				slog.String("key", redisKey),
				slog.String("error", err.Error()),
			)
			return nil, status.Errorf(codes.Internal, "internal server error")
		}

		// Still processing — keep polling.
		if cached == idempotencySentinel {
			continue
		}

		// Real response available — deserialize into structpb.Struct which is
		// a valid proto.Message that gRPC can serialize back to the client.
		var respMap map[string]interface{}
		if unmarshalErr := json.Unmarshal([]byte(cached), &respMap); unmarshalErr != nil {
			i.logger.LogAttrs(ctx, slog.LevelError, "idempotency: failed to unmarshal cached response",
				slog.String("key", redisKey),
				slog.String("error", unmarshalErr.Error()),
			)
			return nil, status.Errorf(codes.Internal, "internal server error")
		}
		structResp, convErr := structpb.NewStruct(respMap)
		if convErr != nil {
			i.logger.LogAttrs(ctx, slog.LevelError, "idempotency: failed to convert to struct",
				slog.String("key", redisKey),
				slog.String("error", convErr.Error()),
			)
			return nil, status.Errorf(codes.Internal, "internal server error")
		}
		return structResp, nil
	}

	return nil, status.Errorf(codes.DeadlineExceeded, "timed out waiting for idempotent response")
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
