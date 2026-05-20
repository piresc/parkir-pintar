// creation; otherwise propagate the unexpected error.
package idempotency

import (
	"context"
	"errors"
	"fmt"
)

type LookupFunc[T any] func(ctx context.Context, key string) (*T, error)

type Result[T any] struct {
	Found  bool
	Record *T
}

// - Otherwise wraps and returns the unexpected error with the provided operation context.
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
