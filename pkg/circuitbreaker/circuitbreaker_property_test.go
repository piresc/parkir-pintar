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

var errSimulated = errors.New("simulated failure")

func TestProperty8_CircuitBreakerStateMachine(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		threshold := rapid.IntRange(1, 5).Draw(t, "threshold")
		openTimeout := 50 * time.Millisecond

		cb := New(Config{
			FailureThreshold:  threshold,
			OpenTimeout:       openTimeout,
			HalfOpenMaxProbes: 1,
		})

		assert.Equal(t, StateClosed, cb.State(), "initial state must be Closed")

		callCount := 0
		_ = cb.Execute(func() error {
			callCount++
			return nil
		})
		assert.Equal(t, 1, callCount, "Closed state must forward calls")
		assert.Equal(t, StateClosed, cb.State(), "success in Closed keeps state Closed")

		for i := 0; i < threshold; i++ {
			err := cb.Execute(func() error {
				callCount++
				return errSimulated
			})
			assert.ErrorIs(t, err, errSimulated)
		}
		assert.Equal(t, StateOpen, cb.State(), "must be Open after threshold consecutive failures")

		callCountBefore := callCount
		err := cb.Execute(func() error {
			callCount++
			return nil
		})
		assert.ErrorIs(t, err, ErrCircuitOpen, "Open state must reject with ErrCircuitOpen")
		assert.Equal(t, callCountBefore, callCount, "Open state must not invoke the wrapped function")

		time.Sleep(openTimeout + 10*time.Millisecond)

		err = cb.Execute(func() error {
			return errSimulated
		})
		assert.ErrorIs(t, err, errSimulated)
		assert.Equal(t, StateOpen, cb.State(), "failed probe in HalfOpen must transition to Open")

		time.Sleep(openTimeout + 10*time.Millisecond)

		err = cb.Execute(func() error {
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, StateClosed, cb.State(), "successful probe in HalfOpen must transition to Closed")
	})
}

// For any number of concurrent goroutines executing calls through the circuit
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
