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

func TestNew_Defaults(t *testing.T) {
	cb := New(Config{})
	assert.Equal(t, StateClosed, cb.State())
}

func TestNew_CustomConfig(t *testing.T) {
	cb := New(Config{
		FailureThreshold:  3,
		OpenTimeout:       10 * time.Second,
		HalfOpenMaxProbes: 2,
	})
	assert.Equal(t, StateClosed, cb.State())
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.state.String())
	}
}

func TestExecute_Success(t *testing.T) {
	cb := New(Config{FailureThreshold: 3})
	err := cb.Execute(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestExecute_FailureBelowThreshold(t *testing.T) {
	cb := New(Config{FailureThreshold: 3})
	testErr := errors.New("fail")

	// 2 failures (below threshold of 3)
	for i := 0; i < 2; i++ {
		err := cb.Execute(func() error { return testErr })
		assert.ErrorIs(t, err, testErr)
	}
	assert.Equal(t, StateClosed, cb.State())
}

func TestExecute_OpensAfterThreshold(t *testing.T) {
	cb := New(Config{FailureThreshold: 3, OpenTimeout: 1 * time.Second})
	testErr := errors.New("fail")

	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error { return testErr })
	}
	assert.Equal(t, StateOpen, cb.State())

	// Next call should return ErrCircuitOpen
	err := cb.Execute(func() error { return nil })
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

func TestExecute_SuccessResetsCount(t *testing.T) {
	cb := New(Config{FailureThreshold: 3})
	testErr := errors.New("fail")

	// 2 failures then 1 success
	_ = cb.Execute(func() error { return testErr })
	_ = cb.Execute(func() error { return testErr })
	_ = cb.Execute(func() error { return nil })

	// 2 more failures should NOT open (count was reset)
	_ = cb.Execute(func() error { return testErr })
	_ = cb.Execute(func() error { return testErr })
	assert.Equal(t, StateClosed, cb.State())
}

func TestExecute_HalfOpenRecovery(t *testing.T) {
	cb := New(Config{FailureThreshold: 2, OpenTimeout: 50 * time.Millisecond, HalfOpenMaxProbes: 1})
	testErr := errors.New("fail")

	// Trip the breaker
	_ = cb.Execute(func() error { return testErr })
	_ = cb.Execute(func() error { return testErr })
	assert.Equal(t, StateOpen, cb.State())

	// Wait for timeout to transition to half-open
	time.Sleep(60 * time.Millisecond)

	// Probe succeeds → should close
	err := cb.Execute(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestExecute_HalfOpenFailure(t *testing.T) {
	cb := New(Config{FailureThreshold: 2, OpenTimeout: 50 * time.Millisecond, HalfOpenMaxProbes: 1})
	testErr := errors.New("fail")

	// Trip the breaker
	_ = cb.Execute(func() error { return testErr })
	_ = cb.Execute(func() error { return testErr })
	assert.Equal(t, StateOpen, cb.State())

	// Wait for half-open
	time.Sleep(60 * time.Millisecond)

	// Probe fails → back to open
	_ = cb.Execute(func() error { return testErr })
	assert.Equal(t, StateOpen, cb.State())
}

func TestExecute_ConcurrentSafety(t *testing.T) {
	cb := New(Config{FailureThreshold: 100, OpenTimeout: 1 * time.Second})
	var wg sync.WaitGroup
	var successCount atomic.Int64

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cb.Execute(func() error { return nil })
			if err == nil {
				successCount.Add(1)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, int64(50), successCount.Load())
	assert.Equal(t, StateClosed, cb.State())
}

func TestErrCircuitOpen_IsDetectable(t *testing.T) {
	cb := New(Config{FailureThreshold: 1, OpenTimeout: 1 * time.Second})
	_ = cb.Execute(func() error { return errors.New("fail") })

	err := cb.Execute(func() error { return nil })
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCircuitOpen))
}
