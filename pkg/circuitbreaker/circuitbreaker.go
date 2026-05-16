// Package circuitbreaker provides a circuit breaker for wrapping external
// service calls. It prevents cascade failures by failing fast when a
// downstream service is unhealthy.
//
// This package is a thin wrapper around github.com/sony/gobreaker/v2.
package circuitbreaker

import (
	"errors"
	"time"

	"github.com/sony/gobreaker/v2"
)

// State represents the circuit breaker state.
type State int

const (
	StateClosed   State = State(gobreaker.StateClosed)
	StateHalfOpen State = State(gobreaker.StateHalfOpen)
	StateOpen     State = State(gobreaker.StateOpen)
)

// String returns the human-readable state name.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when the circuit breaker is in the Open state.
var ErrCircuitOpen = gobreaker.ErrOpenState

// Config holds the circuit breaker configuration.
type Config struct {
	// FailureThreshold is the number of consecutive failures before opening.
	FailureThreshold int
	// OpenTimeout is how long the circuit stays open before transitioning to half-open.
	OpenTimeout time.Duration
	// HalfOpenMaxProbes is the number of probe calls allowed in half-open state.
	HalfOpenMaxProbes int
}

// CircuitBreaker wraps sony/gobreaker with a simplified API.
type CircuitBreaker struct {
	cb *gobreaker.CircuitBreaker[any]
}

// New creates a CircuitBreaker with the given configuration.
// Defaults: FailureThreshold=5, OpenTimeout=30s, HalfOpenMaxProbes=1.
func New(cfg Config) *CircuitBreaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.OpenTimeout <= 0 {
		cfg.OpenTimeout = 30 * time.Second
	}
	if cfg.HalfOpenMaxProbes <= 0 {
		cfg.HalfOpenMaxProbes = 1
	}

	settings := gobreaker.Settings{
		MaxRequests: uint32(cfg.HalfOpenMaxProbes),
		Interval:    0, // don't reset counts periodically
		Timeout:     cfg.OpenTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return int(counts.ConsecutiveFailures) >= cfg.FailureThreshold
		},
	}

	return &CircuitBreaker{
		cb: gobreaker.NewCircuitBreaker[any](settings),
	}
}

// Execute runs the given function through the circuit breaker.
// Returns ErrCircuitOpen (gobreaker.ErrOpenState) if the circuit is open.
func (c *CircuitBreaker) Execute(fn func() error) error {
	_, err := c.cb.Execute(func() (any, error) {
		return nil, fn()
	})
	// Map gobreaker's ErrTooManyRequests (half-open limit) to ErrCircuitOpen
	// for backward compatibility.
	if errors.Is(err, gobreaker.ErrTooManyRequests) {
		return ErrCircuitOpen
	}
	return err
}

// State returns the current state of the circuit breaker.
func (c *CircuitBreaker) State() State {
	return State(c.cb.State())
}
