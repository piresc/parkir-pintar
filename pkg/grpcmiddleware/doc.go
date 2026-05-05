// Package grpcmiddleware provides gRPC server interceptors for authentication,
// recovery, tracing, rate limiting, logging, and idempotency. All interceptors
// are accessed via the [Interceptors] struct, which holds shared dependencies
// (jwtSecret, logger, tracer, redisClient), mirroring the pattern used by
// pkg/middleware for HTTP middleware.
//
// # Recommended Interceptor Chain Order
//
// The interceptors should be wired outermost-first in this order:
//
//	Recovery → Tracing → Logging → RateLimit → Auth → Idempotency → Handler
//
// Recovery is outermost so that panics in any inner interceptor or handler are
// caught. Tracing wraps everything to capture the full request duration.
// Logging records the final status code. RateLimit rejects excess traffic
// before auth work is done. Auth validates the JWT. Idempotency is innermost
// (before the handler) so it only applies to authenticated, rate-limited
// requests.
//
// # Example Setup
//
// Create an [Interceptors] instance and wire the chain using
// [grpc.ChainUnaryInterceptor] and [grpc.ChainStreamInterceptor]:
//
//	interceptors := grpcmiddleware.NewInterceptors(jwtSecret, logger, tracer, redisClient)
//
//	rateLimitCfg := grpcmiddleware.RateLimitConfig{
//	    RequestsPerSecond: 100,
//	    BurstSize:         200,
//	    CleanupInterval:   time.Minute,
//	}
//
//	idempotencyCfg := grpcmiddleware.IdempotencyConfig{
//	    TTL:     24 * time.Hour,
//	    Methods: []string{"/parking.ReservationService/CreateReservation"},
//	}
//
//	publicMethods := []string{"/grpc.health.v1.Health/Check"}
//
//	srv := grpc.NewServer(
//	    grpc.ChainUnaryInterceptor(
//	        interceptors.RecoveryUnaryInterceptor(),
//	        interceptors.TracingUnaryInterceptor(),
//	        interceptors.LoggingUnaryInterceptor(),
//	        interceptors.RateLimitUnaryInterceptor(rateLimitCfg),
//	        interceptors.AuthUnaryInterceptor(publicMethods),
//	        interceptors.IdempotencyUnaryInterceptor(idempotencyCfg),
//	    ),
//	    grpc.ChainStreamInterceptor(
//	        interceptors.RecoveryStreamInterceptor(),
//	        interceptors.TracingStreamInterceptor(),
//	        interceptors.LoggingStreamInterceptor(),
//	        interceptors.AuthStreamInterceptor(publicMethods),
//	    ),
//	)
package grpcmiddleware
