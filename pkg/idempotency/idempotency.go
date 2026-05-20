// Package idempotency provides a reusable utility for the common
// idempotency-check pattern used across domain usecases. The pattern is:
// look up an existing record by idempotency key; if found, return it (cached
// result); if the error is "not found", signal the caller to proceed with
// creation; otherwise propagate the unexpected error.
package idempotency

import (
	"context"
	"errors"
	"fmt"
)

// LookupFunc is a function that retrieves an existing record by idempotency key.
// It should return (nil, ErrNotFound) when no record exists, or (record, nil)
// when a cached result is available.
type LookupFunc[T any] func(ctx context.Context, key string) (*T, error)

// Result holds the outcome of an idempotency check.
type Result[T any] struct {
	// Found is true when a cached record was returned by the lookup.
	Found bool
	// Record is the cached record (non-nil only when Found is true).
	Record *T
}

// Check performs the standard idempotency lookup pattern:
//   - Calls lookupFn with the given idempotency key.
//   - If a record is found (no error, non-nil), returns Result{Found: true, Record: record}.
//   - If the error matches errNotFound, returns Result{Found: false} so the caller proceeds.
//   - Otherwise wraps and returns the unexpected error with the provided operation context.
//
// Usage:
//
//	res, err := idempotency.Check(ctx, req.IdempotencyKey, uc.repo.GetByIdempotencyKey, repository.ErrNotFound, "process payment")
//	if err != nil {
//	    return nil, err
//	}
//	if res.Found {
//	    return res.Record, nil
//	}
//	// ... proceed with creation
func Check[T any](ctx context.Context, key string, lookupFn LookupFunc[T], errNotFound error, operation string) (Result[T], error) {
	existing, err := lookupFn(ctx, key)
	if err == nil && existing != nil {
		return Result[T]{Found: true, Record: existing}, nil
	}
	if err != nil && !errors.Is(err, errNotFound) {
		return Result[T]{}, fmt.Errorf("%s idempotency check: %w", operation, err)
	}
	return Result[T]{}, nil
}
