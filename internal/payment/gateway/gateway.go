// Package gateway provides the payment gateway abstraction and a stub
// implementation for testing. The stub gateway returns configurable
// success/failure responses without calling any external service.
//
// Best practices applied (from Go coding standards KB):
// - Keep interfaces small and define them where they're used
// - Document all exported functions and types with proper Godoc format
// - Use context.Context as first parameter for consistency
// - Use keyed fields in struct literals to prevent breakages during refactors
package gateway

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrGatewayFailure is returned when the payment gateway call fails.
var ErrGatewayFailure = errors.New("payment gateway failure")

// PaymentGateway defines the interface for external payment gateway operations.
type PaymentGateway interface {
	Charge(ctx context.Context, amount int64, method string) (transactionRef string, err error)
	Refund(ctx context.Context, transactionRef string) error
	GetStatus(ctx context.Context, transactionRef string) (string, error)
}

// StubGateway is a configurable stub implementation of PaymentGateway for testing.
type StubGateway struct {
	ShouldFail bool
}

// NewStubGateway creates a new StubGateway with the given failure configuration.
func NewStubGateway(shouldFail bool) *StubGateway {
	return &StubGateway{ShouldFail: shouldFail}
}

// Charge simulates a payment charge. When ShouldFail is false, it returns
// a transaction reference in the format "txn-{uuid}". When ShouldFail is
// true, it returns ErrGatewayFailure.
func (g *StubGateway) Charge(_ context.Context, _ int64, _ string) (string, error) {
	if g.ShouldFail {
		return "", fmt.Errorf("charge: %w", ErrGatewayFailure)
	}
	return fmt.Sprintf("txn-%s", uuid.New().String()), nil
}

// Refund simulates a payment refund. When ShouldFail is false, it returns nil.
// When ShouldFail is true, it returns ErrGatewayFailure.
func (g *StubGateway) Refund(_ context.Context, _ string) error {
	if g.ShouldFail {
		return fmt.Errorf("refund: %w", ErrGatewayFailure)
	}
	return nil
}

// GetStatus simulates a payment status query. When ShouldFail is false, it
// returns "success". When ShouldFail is true, it returns ErrGatewayFailure.
func (g *StubGateway) GetStatus(_ context.Context, _ string) (string, error) {
	if g.ShouldFail {
		return "", fmt.Errorf("get status: %w", ErrGatewayFailure)
	}
	return "success", nil
}
