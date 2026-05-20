package idempotency_test

import (
	"context"
	"errors"
	"testing"

	"parkir-pintar/pkg/idempotency"
)

var errNotFound = errors.New("not found")

type record struct {
	ID   string
	Name string
}

func TestCheck_Found(t *testing.T) {
	cached := &record{ID: "abc", Name: "existing"}
	lookup := func(_ context.Context, _ string) (*record, error) {
		return cached, nil
	}

	res, err := idempotency.Check(context.Background(), "key-1", lookup, errNotFound, "test op")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Found {
		t.Fatal("expected Found=true")
	}
	if res.Record != cached {
		t.Fatalf("expected cached record, got %v", res.Record)
	}
}

func TestCheck_NotFound(t *testing.T) {
	lookup := func(_ context.Context, _ string) (*record, error) {
		return nil, errNotFound
	}

	res, err := idempotency.Check(context.Background(), "key-2", lookup, errNotFound, "test op")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Found {
		t.Fatal("expected Found=false")
	}
}

func TestCheck_UnexpectedError(t *testing.T) {
	dbErr := errors.New("connection refused")
	lookup := func(_ context.Context, _ string) (*record, error) {
		return nil, dbErr
	}

	_, err := idempotency.Check(context.Background(), "key-3", lookup, errNotFound, "test op")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped dbErr, got: %v", err)
	}
}

func TestCheck_NilRecordNoError(t *testing.T) {
	// Edge case: lookup returns (nil, nil) — treat as not found.
	lookup := func(_ context.Context, _ string) (*record, error) {
		return nil, nil
	}

	res, err := idempotency.Check(context.Background(), "key-4", lookup, errNotFound, "test op")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Found {
		t.Fatal("expected Found=false for nil record with nil error")
	}
}
