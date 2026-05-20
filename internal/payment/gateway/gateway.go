// - Use keyed fields in struct literals to prevent breakages during refactors
package gateway

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrGatewayFailure = errors.New("payment gateway failure")

//go:generate mockgen -destination=../mocks/mock_payment_gateway.go -package=mocks parkir-pintar/internal/payment/gateway PaymentGateway
type PaymentGateway interface {
	Charge(ctx context.Context, amount int64, method string) (transactionRef string, err error)
	Refund(ctx context.Context, transactionRef string) error
	GetStatus(ctx context.Context, transactionRef string) (string, error)
}

type StubGateway struct {
	ShouldFail bool
}

func NewStubGateway(shouldFail bool) *StubGateway {
	return &StubGateway{ShouldFail: shouldFail}
}

func (g *StubGateway) Charge(_ context.Context, _ int64, _ string) (string, error) {
	if g.ShouldFail {
		return "", fmt.Errorf("charge: %w", ErrGatewayFailure)
	}
	return fmt.Sprintf("txn-%s", uuid.New().String()), nil
}

func (g *StubGateway) Refund(_ context.Context, _ string) error {
	if g.ShouldFail {
		return fmt.Errorf("refund: %w", ErrGatewayFailure)
	}
	return nil
}

func (g *StubGateway) GetStatus(_ context.Context, _ string) (string, error) {
	if g.ShouldFail {
		return "", fmt.Errorf("get status: %w", ErrGatewayFailure)
	}
	return "success", nil
}
