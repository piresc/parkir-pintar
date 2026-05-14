// Package circuitbreaker implements the circuit breaker pattern for wrapping
// external service calls (payment gateway, notification, etc.). It prevents
// cascade failures by failing fast when a downstream service is unhealthy.
//
// The circuit breaker has three states:
//   - Closed: all calls are forwarded; consecutive failures are tracked.
//   - Open: calls are rejected immediately with ErrCircuitOpen.
//   - HalfOpen: a limited number of probe calls are allowed through to test recovery.
//
// Thread-safe via sync.Mutex. Safe for concurrent use from multiple goroutines.
package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State represents the current state of the circuit breaker.
type State int

const (
	// StateClosed is the normal operating state. All calls are forwarded.
	StateClosed State = iota
	// StateOpen is the failing-fast state. Calls are rejected immediately.
	StateOpen
	// StateHalfOpen is the recovery-testing state. A limited number of probe
	// calls are allowed through.
	StateHalfOpen
)

// String returns a human-readable representation of the circuit breaker state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Sentinel errors returned by the circuit breaker.
var (
	// ErrCircuitOpen is returned when the circuit breaker is in the Open state
	// and rejects calls without forwarding them.
	ErrCircuitOpen = errors.New("circuit breaker open")
)

// Config holds the configuration parameters for a CircuitBreaker.
type Config struct {
	// FailureThreshold is the number of consecutive failures required to
	// transition from Closed to Open state.
	FailureThreshold int
	// OpenTimeout is the duration the circuit stays in Open state before
	// transitioning to HalfOpen.
	OpenTimeout time.Duration
	// HalfOpenMaxProbes is the maximum number of probe calls allowed in
	// HalfOpen state before a success transitions to Closed.
	HalfOpenMaxProbes int
}

// CircuitBreaker wraps external service calls with circuit breaker logic.
// It is safe for concurrent use from multiple goroutines.
type CircuitBreaker struct {
	mu               sync.Mutex
	state            State
	failureCount     int
	successCount     int
	lastFailureTime  time.Time
	failureThreshold int
	openTimeout      time.Duration
	halfOpenMax      int
	// now is a function returning the current time; overridable for testing.
	now func() time.Time
}

// New creates a new CircuitBreaker with the given configuration.
// If FailureThreshold is less than 1, it defaults to 5.
// If OpenTimeout is zero, it defaults to 30 seconds.
// If HalfOpenMaxProbes is less than 1, it defaults to 1.
func New(cfg Config) *CircuitBreaker {
	threshold := cfg.FailureThreshold
	if threshold < 1 {
		threshold = 5
	}

	timeout := cfg.OpenTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	halfOpenMax := cfg.HalfOpenMaxProbes
	if halfOpenMax < 1 {
		halfOpenMax = 1
	}

	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: threshold,
		openTimeout:      timeout,
		halfOpenMax:      halfOpenMax,
		now:              time.Now,
	}
}

// State returns the current state of the circuit breaker.
// It also evaluates whether an Open→HalfOpen transition should occur
// based on the elapsed time since the last failure.
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen && cb.now().Sub(cb.lastFailureTime) >= cb.openTimeout {
		cb.toHalfOpen()
	}

	return cb.state
}

// Execute runs fn if the circuit allows it.
// In Closed state, all calls are forwarded. Consecutive failures are tracked,
// and the circuit transitions to Open when the threshold is reached.
// In Open state, calls are rejected immediately with ErrCircuitOpen.
// In HalfOpen state, a limited number of probe calls are allowed through;
// a success transitions to Closed, a failure transitions back to Open.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()

	// Check for Open→HalfOpen transition based on timeout.
	if cb.state == StateOpen && cb.now().Sub(cb.lastFailureTime) >= cb.openTimeout {
		cb.toHalfOpen()
	}

	switch cb.state {
	case StateOpen:
		cb.mu.Unlock()
		return ErrCircuitOpen

	case StateHalfOpen:
		cb.mu.Unlock()
		err := fn()
		cb.mu.Lock()
		defer cb.mu.Unlock()

		if err != nil {
			cb.toOpen()
			return err
		}

		cb.successCount++
		if cb.successCount >= cb.halfOpenMax {
			cb.toClosed()
		}
		return nil

	case StateClosed:
		cb.mu.Unlock()
		err := fn()
		cb.mu.Lock()
		defer cb.mu.Unlock()

		if err != nil {
			cb.failureCount++
			cb.lastFailureTime = cb.now()
			if cb.failureCount >= cb.failureThreshold {
				cb.toOpen()
			}
			return err
		}

		cb.failureCount = 0
		return nil

	default:
		cb.mu.Unlock()
		return nil
	}
}

// toOpen transitions the circuit breaker to the Open state.
// Must be called with cb.mu held.
func (cb *CircuitBreaker) toOpen() {
	cb.state = StateOpen
	cb.lastFailureTime = cb.now()
	cb.successCount = 0
}

// toHalfOpen transitions the circuit breaker to the HalfOpen state.
// Must be called with cb.mu held.
func (cb *CircuitBreaker) toHalfOpen() {
	cb.state = StateHalfOpen
	cb.successCount = 0
	cb.failureCount = 0
}

// toClosed transitions the circuit breaker to the Closed state.
// Must be called with cb.mu held.
func (cb *CircuitBreaker) toClosed() {
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
}
