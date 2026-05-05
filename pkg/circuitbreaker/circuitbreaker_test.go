// Package circuitbreaker unit tests
//
// Best practices applied (from Go testing standards):
// - Descriptive names: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA (Arrange-Act-Assert) pattern
// - Table-driven tests for multiple scenarios
// - testify assertions for clear failure messages
// - Tests are fast, isolated, repeatable, and clear
// - Test both success and error/edge cases
// - Overridable `now` field used for deterministic time-dependent tests
// - Concurrent tests run with -race flag for data race detection
package circuitbreaker

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errTest is a sentinel error used across unit tests.
var errTest = errors.New("test error")

// newTestCB creates a CircuitBreaker with a controllable clock.
// Returns the CB and a function to advance time.
func newTestCB(threshold int, openTimeout time.Duration) (*CircuitBreaker, func(time.Duration)) {
	currentTime := time.Now()
	var mu sync.Mutex

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

	advance := func(d time.Duration) {
		mu.Lock()
		currentTime = currentTime.Add(d)
		mu.Unlock()
	}

	return cb, advance
}

// --- New constructor tests ---

func TestNew_ShouldApplyDefaults_WhenZeroConfig(t *testing.T) {
	// Arrange & Act
	cb := New(Config{})

	// Assert
	assert.Equal(t, StateClosed, cb.State())
	assert.Equal(t, 5, cb.failureThreshold)
	assert.Equal(t, 30*time.Second, cb.openTimeout)
	assert.Equal(t, 1, cb.halfOpenMax)
}

func TestNew_ShouldUseProvidedValues_WhenConfigSet(t *testing.T) {
	// Arrange & Act
	cb := New(Config{
		FailureThreshold:  3,
		OpenTimeout:       10 * time.Second,
		HalfOpenMaxProbes: 2,
	})

	// Assert
	assert.Equal(t, StateClosed, cb.State())
	assert.Equal(t, 3, cb.failureThreshold)
	assert.Equal(t, 10*time.Second, cb.openTimeout)
	assert.Equal(t, 2, cb.halfOpenMax)
}

// --- State string representation ---

func TestStateString_ShouldReturnReadableNames(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

// --- Closed state behavior (Requirements 8.1, 8.2) ---

func TestExecute_ShouldForwardCalls_WhenClosed(t *testing.T) {
	// Arrange
	cb, _ := newTestCB(3, time.Minute)
	called := false

	// Act
	err := cb.Execute(func() error {
		called = true
		return nil
	})

	// Assert
	require.NoError(t, err)
	assert.True(t, called, "function must be called in Closed state")
	assert.Equal(t, StateClosed, cb.State())
}

func TestExecute_ShouldReturnFnError_WhenClosedAndFnFails(t *testing.T) {
	// Arrange
	cb, _ := newTestCB(3, time.Minute)

	// Act
	err := cb.Execute(func() error { return errTest })

	// Assert
	assert.ErrorIs(t, err, errTest)
	assert.Equal(t, StateClosed, cb.State(), "single failure should not open circuit")
}

func TestExecute_ShouldResetFailureCount_WhenSuccessAfterFailures(t *testing.T) {
	// Arrange — threshold=3, cause 2 failures then 1 success
	cb, _ := newTestCB(3, time.Minute)

	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })

	// Act — success resets counter
	err := cb.Execute(func() error { return nil })
	require.NoError(t, err)

	// Now 2 more failures should NOT open (counter was reset)
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })

	// Assert
	assert.Equal(t, StateClosed, cb.State(), "counter should have been reset by success")
}

// --- Closed → Open transition (Requirement 8.3) ---

func TestExecute_ShouldTransitionToOpen_WhenConsecutiveFailuresReachThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold int
	}{
		{name: "threshold=1", threshold: 1},
		{name: "threshold=3", threshold: 3},
		{name: "threshold=5", threshold: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cb, _ := newTestCB(tt.threshold, time.Minute)

			// Act — cause exactly threshold consecutive failures
			for i := 0; i < tt.threshold; i++ {
				_ = cb.Execute(func() error { return errTest })
			}

			// Assert
			assert.Equal(t, StateOpen, cb.State())
		})
	}
}

// --- Open state behavior (Requirement 8.4) ---

func TestExecute_ShouldRejectWithoutCallingFn_WhenOpen(t *testing.T) {
	// Arrange — force Open state
	cb, _ := newTestCB(1, time.Minute)
	_ = cb.Execute(func() error { return errTest }) // triggers Open

	called := false

	// Act
	err := cb.Execute(func() error {
		called = true
		return nil
	})

	// Assert
	assert.ErrorIs(t, err, ErrCircuitOpen)
	assert.False(t, called, "function must NOT be called in Open state")
	assert.Equal(t, StateOpen, cb.State())
}

// --- Open → HalfOpen transition (Requirement 8.5) ---

func TestState_ShouldTransitionToHalfOpen_WhenOpenTimeoutElapses(t *testing.T) {
	// Arrange
	openTimeout := 100 * time.Millisecond
	cb, advance := newTestCB(1, openTimeout)
	_ = cb.Execute(func() error { return errTest }) // → Open
	assert.Equal(t, StateOpen, cb.State())

	// Act — advance past timeout
	advance(openTimeout + time.Millisecond)

	// Assert
	assert.Equal(t, StateHalfOpen, cb.State())
}

func TestState_ShouldRemainOpen_WhenTimeoutNotElapsed(t *testing.T) {
	// Arrange
	openTimeout := 100 * time.Millisecond
	cb, advance := newTestCB(1, openTimeout)
	_ = cb.Execute(func() error { return errTest }) // → Open

	// Act — advance less than timeout
	advance(openTimeout / 2)

	// Assert
	assert.Equal(t, StateOpen, cb.State())
}

// --- HalfOpen probe success path (Requirement 8.6) ---

func TestExecute_ShouldTransitionToClosed_WhenHalfOpenProbeSucceeds(t *testing.T) {
	// Arrange
	openTimeout := 50 * time.Millisecond
	cb, advance := newTestCB(1, openTimeout)
	_ = cb.Execute(func() error { return errTest }) // → Open
	advance(openTimeout + time.Millisecond)          // → HalfOpen
	assert.Equal(t, StateHalfOpen, cb.State())

	// Act — successful probe
	err := cb.Execute(func() error { return nil })

	// Assert
	require.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

// --- HalfOpen probe failure path (Requirement 8.6) ---

func TestExecute_ShouldTransitionBackToOpen_WhenHalfOpenProbeFails(t *testing.T) {
	// Arrange
	openTimeout := 50 * time.Millisecond
	cb, advance := newTestCB(1, openTimeout)
	_ = cb.Execute(func() error { return errTest }) // → Open
	advance(openTimeout + time.Millisecond)          // → HalfOpen
	assert.Equal(t, StateHalfOpen, cb.State())

	// Act — failed probe
	err := cb.Execute(func() error { return errTest })

	// Assert
	assert.ErrorIs(t, err, errTest)
	assert.Equal(t, StateOpen, cb.State())
}

// --- HalfOpen full cycle test ---

func TestExecute_ShouldCompleteFullCycle_WhenFailThenRecoverThenFailAgain(t *testing.T) {
	// Arrange
	openTimeout := 50 * time.Millisecond
	cb, advance := newTestCB(2, openTimeout)

	// Closed → Open (2 consecutive failures)
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })
	assert.Equal(t, StateOpen, cb.State())

	// Open → HalfOpen (timeout)
	advance(openTimeout + time.Millisecond)
	assert.Equal(t, StateHalfOpen, cb.State())

	// HalfOpen → Closed (probe success)
	err := cb.Execute(func() error { return nil })
	require.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())

	// Closed → Open again (2 more failures)
	_ = cb.Execute(func() error { return errTest })
	_ = cb.Execute(func() error { return errTest })
	assert.Equal(t, StateOpen, cb.State())

	// Open → HalfOpen → Open (probe failure)
	advance(openTimeout + time.Millisecond)
	assert.Equal(t, StateHalfOpen, cb.State())
	err = cb.Execute(func() error { return errTest })
	assert.ErrorIs(t, err, errTest)
	assert.Equal(t, StateOpen, cb.State())
}

// --- Configurable parameters (Requirement 8.7) ---

func TestNew_ShouldAcceptConfigurableParameters(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantThres int
		wantTO    time.Duration
		wantHO    int
	}{
		{
			name:      "custom values",
			cfg:       Config{FailureThreshold: 10, OpenTimeout: 5 * time.Second, HalfOpenMaxProbes: 3},
			wantThres: 10,
			wantTO:    5 * time.Second,
			wantHO:    3,
		},
		{
			name:      "minimum defaults applied",
			cfg:       Config{FailureThreshold: 0, OpenTimeout: 0, HalfOpenMaxProbes: 0},
			wantThres: 5,
			wantTO:    30 * time.Second,
			wantHO:    1,
		},
		{
			name:      "negative threshold defaults",
			cfg:       Config{FailureThreshold: -1, OpenTimeout: time.Second, HalfOpenMaxProbes: -1},
			wantThres: 5,
			wantTO:    time.Second,
			wantHO:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			cb := New(tt.cfg)

			// Assert
			assert.Equal(t, tt.wantThres, cb.failureThreshold)
			assert.Equal(t, tt.wantTO, cb.openTimeout)
			assert.Equal(t, tt.wantHO, cb.halfOpenMax)
		})
	}
}

// --- Concurrent access (Requirement 8.8) ---

func TestExecute_ShouldBeSafeForConcurrentUse(t *testing.T) {
	// Arrange
	cb, _ := newTestCB(5, 50*time.Millisecond)
	const goroutines = 20
	const callsPerGoroutine = 50

	var wg sync.WaitGroup
	var totalExecuted atomic.Int64

	// Act — concurrent mixed success/failure calls
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				_ = cb.Execute(func() error {
					totalExecuted.Add(1)
					if j%4 == 0 {
						return errTest
					}
					return nil
				})
			}
		}(i)
	}
	wg.Wait()

	// Assert — state must be valid, some calls must have executed
	state := cb.State()
	assert.True(t, state == StateClosed || state == StateOpen || state == StateHalfOpen,
		"state must be valid, got: %v", state)
	assert.Greater(t, totalExecuted.Load(), int64(0), "at least some calls must have been forwarded")
}

func TestExecute_ShouldNotRace_WhenConcurrentStateReadsAndWrites(t *testing.T) {
	// Arrange — this test is primarily validated by running with -race flag
	cb, advance := newTestCB(2, 20*time.Millisecond)

	var wg sync.WaitGroup

	// Writer goroutines: cause failures
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 30; j++ {
				_ = cb.Execute(func() error { return errTest })
			}
		}()
	}

	// Reader goroutines: read state
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 30; j++ {
				_ = cb.State()
			}
		}()
	}

	// Time advancer: trigger Open→HalfOpen transitions
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			advance(25 * time.Millisecond)
			_ = cb.Execute(func() error { return nil })
		}
	}()

	wg.Wait()

	// Assert — no panic, no race (validated by -race flag)
	state := cb.State()
	assert.True(t, state == StateClosed || state == StateOpen || state == StateHalfOpen)
}

// --- Execute triggers Open→HalfOpen transition internally ---

func TestExecute_ShouldTransitionToHalfOpen_WhenCalledAfterTimeout(t *testing.T) {
	// Arrange — verify Execute itself checks the timeout, not just State()
	openTimeout := 50 * time.Millisecond
	cb, advance := newTestCB(1, openTimeout)
	_ = cb.Execute(func() error { return errTest }) // → Open

	advance(openTimeout + time.Millisecond)

	// Act — Execute should detect timeout and allow probe
	called := false
	err := cb.Execute(func() error {
		called = true
		return nil
	})

	// Assert
	require.NoError(t, err)
	assert.True(t, called, "Execute should allow probe call after timeout")
	assert.Equal(t, StateClosed, cb.State(), "successful probe should transition to Closed")
}
