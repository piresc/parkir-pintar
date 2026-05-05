package circuitbreaker

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// errSimulated is a sentinel error used by test call sequences.
var errSimulated = errors.New("simulated failure")

// Feature: grpc-jwt-pkg-integration, Property 8: Circuit breaker state machine
// **Validates: Requirements 8.1, 8.2, 8.3, 8.4, 8.5, 8.6**
//
// For any failure threshold T and any sequence of success/failure call results:
// (a) in Closed state, all calls are forwarded;
// (b) after T consecutive failures, the state transitions to Open;
// (c) in Open state, calls are rejected immediately without invoking the wrapped function;
// (d) after the open timeout, the state transitions to Half-Open;
// (e) in Half-Open, a successful probe transitions to Closed and a failed probe
//     transitions back to Open.
func TestProperty8_CircuitBreakerStateMachine(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		threshold := rapid.IntRange(1, 10).Draw(t, "threshold")
		openTimeoutMs := rapid.IntRange(1, 100).Draw(t, "openTimeoutMs")
		openTimeout := time.Duration(openTimeoutMs) * time.Millisecond

		// Use a controllable clock so we can simulate time passing.
		currentTime := time.Now()
		var mu sync.Mutex
		advanceTime := func(d time.Duration) {
			mu.Lock()
			currentTime = currentTime.Add(d)
			mu.Unlock()
		}

		cb := New(Config{
			FailureThreshold:  threshold,
			OpenTimeout:       openTimeout,
			HalfOpenMaxProbes: 1,
		})
		cb.now = func() time.Time {
			mu.Lock()
			defer mu.Unlock()
			return currentTime
		}

		// (a) In Closed state, all calls are forwarded.
		assert.Equal(t, StateClosed, cb.State(), "initial state must be Closed")

		callCount := 0
		_ = cb.Execute(func() error {
			callCount++
			return nil
		})
		assert.Equal(t, 1, callCount, "Closed state must forward calls")
		assert.Equal(t, StateClosed, cb.State(), "success in Closed keeps state Closed")

		// (b) After T consecutive failures, transition to Open.
		for i := 0; i < threshold; i++ {
			err := cb.Execute(func() error {
				callCount++
				return errSimulated
			})
			assert.ErrorIs(t, err, errSimulated)
		}
		assert.Equal(t, StateOpen, cb.State(), "must be Open after threshold consecutive failures")

		// (c) In Open state, calls are rejected without invoking fn.
		callCountBefore := callCount
		err := cb.Execute(func() error {
			callCount++
			return nil
		})
		assert.ErrorIs(t, err, ErrCircuitOpen, "Open state must reject with ErrCircuitOpen")
		assert.Equal(t, callCountBefore, callCount, "Open state must not invoke the wrapped function")

		// (d) After open timeout, transition to Half-Open.
		advanceTime(openTimeout + time.Millisecond)
		assert.Equal(t, StateHalfOpen, cb.State(), "must transition to HalfOpen after timeout")

		// (e-1) In Half-Open, a failed probe transitions back to Open.
		err = cb.Execute(func() error {
			return errSimulated
		})
		assert.ErrorIs(t, err, errSimulated)
		assert.Equal(t, StateOpen, cb.State(), "failed probe in HalfOpen must transition to Open")

		// Advance time again to get back to HalfOpen.
		advanceTime(openTimeout + time.Millisecond)
		assert.Equal(t, StateHalfOpen, cb.State(), "must transition to HalfOpen again after timeout")

		// (e-2) In Half-Open, a successful probe transitions to Closed.
		err = cb.Execute(func() error {
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, StateClosed, cb.State(), "successful probe in HalfOpen must transition to Closed")
	})
}

// Feature: grpc-jwt-pkg-integration, Property 9: Circuit breaker concurrency safety
// **Validates: Requirements 8.8**
//
// For any number of concurrent goroutines executing calls through the circuit
// breaker, the state transitions SHALL remain consistent and no data races
// SHALL occur.
func TestProperty9_CircuitBreakerConcurrencySafety(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		threshold := rapid.IntRange(1, 10).Draw(t, "threshold")
		goroutines := rapid.IntRange(2, 50).Draw(t, "goroutines")

		cb := New(Config{
			FailureThreshold:  threshold,
			OpenTimeout:       50 * time.Millisecond,
			HalfOpenMaxProbes: 1,
		})

		var wg sync.WaitGroup
		var executed atomic.Int64

		wg.Add(goroutines)
		for i := 0; i < goroutines; i++ {
			go func() {
				defer wg.Done()
				// Each goroutine executes a mix of success and failure calls.
				for j := 0; j < 20; j++ {
					_ = cb.Execute(func() error {
						executed.Add(1)
						if j%3 == 0 {
							return errSimulated
						}
						return nil
					})
				}
			}()
		}
		wg.Wait()

		// The state must be one of the three valid states.
		state := cb.State()
		assert.True(t, state == StateClosed || state == StateOpen || state == StateHalfOpen,
			"state must be a valid circuit breaker state, got: %v", state)

		// At least some calls must have been executed (not all rejected).
		assert.Greater(t, executed.Load(), int64(0),
			"at least some calls must have been forwarded")
	})
}
