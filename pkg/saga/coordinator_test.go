package saga

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaga_ShouldSucceed_WhenAllStepsPass(t *testing.T) {
	var executed []string

	saga := NewSaga("test-saga")
	saga.AddStep(Step{
		Name:       "step1",
		Execute:    func(ctx context.Context) error { executed = append(executed, "exec1"); return nil },
		Compensate: func(ctx context.Context) error { return nil },
	})
	saga.AddStep(Step{
		Name:       "step2",
		Execute:    func(ctx context.Context) error { executed = append(executed, "exec2"); return nil },
		Compensate: func(ctx context.Context) error { return nil },
	})
	saga.AddStep(Step{
		Name:       "step3",
		Execute:    func(ctx context.Context) error { executed = append(executed, "exec3"); return nil },
		Compensate: func(ctx context.Context) error { return nil },
	})

	result := saga.Execute(context.Background())

	require.True(t, result.Success())
	assert.Equal(t, []string{"step1", "step2", "step3"}, result.CompletedSteps)
	assert.Empty(t, result.FailedStep)
	assert.NoError(t, result.Error)
	assert.False(t, result.HasCompensationErrors())
	assert.Equal(t, []string{"exec1", "exec2", "exec3"}, executed)
}

func TestSaga_ShouldCompensate_WhenStep3Fails(t *testing.T) {
	var compensated []string

	saga := NewSaga("test-saga")
	saga.AddStep(Step{
		Name:       "step1",
		Execute:    func(ctx context.Context) error { return nil },
		Compensate: func(ctx context.Context) error { compensated = append(compensated, "comp1"); return nil },
	})
	saga.AddStep(Step{
		Name:       "step2",
		Execute:    func(ctx context.Context) error { return nil },
		Compensate: func(ctx context.Context) error { compensated = append(compensated, "comp2"); return nil },
	})
	saga.AddStep(Step{
		Name:       "step3",
		Execute:    func(ctx context.Context) error { return errors.New("step3 failed") },
		Compensate: func(ctx context.Context) error { compensated = append(compensated, "comp3"); return nil },
	})

	result := saga.Execute(context.Background())

	require.False(t, result.Success())
	assert.Equal(t, "step3", result.FailedStep)
	assert.ErrorContains(t, result.Error, "step3 failed")
	assert.Equal(t, []string{"step1", "step2"}, result.CompletedSteps)
	// Compensations run in reverse: step2 then step1
	assert.Equal(t, []string{"comp2", "comp1"}, compensated)
	assert.False(t, result.HasCompensationErrors())
}

func TestSaga_ShouldRecordCompensationErrors_WhenCompensationFails(t *testing.T) {
	saga := NewSaga("test-saga")
	saga.AddStep(Step{
		Name:       "step1",
		Execute:    func(ctx context.Context) error { return nil },
		Compensate: func(ctx context.Context) error { return errors.New("comp1 failed") },
	})
	saga.AddStep(Step{
		Name:       "step2",
		Execute:    func(ctx context.Context) error { return nil },
		Compensate: func(ctx context.Context) error { return nil },
	})
	saga.AddStep(Step{
		Name:       "step3",
		Execute:    func(ctx context.Context) error { return errors.New("step3 boom") },
		Compensate: func(ctx context.Context) error { return nil },
	})

	result := saga.Execute(context.Background())

	require.False(t, result.Success())
	assert.Equal(t, "step3", result.FailedStep)
	assert.True(t, result.HasCompensationErrors())
	assert.Contains(t, result.CompensationErrors, "step1")
	assert.ErrorContains(t, result.CompensationErrors["step1"], "comp1 failed")
	// step2 compensation succeeded, so it shouldn't be in errors
	assert.NotContains(t, result.CompensationErrors, "step2")
}

func TestSaga_ShouldPropagateContext(t *testing.T) {
	type ctxKey string
	key := ctxKey("trace_id")

	var capturedValue string

	saga := NewSaga("ctx-saga")
	saga.AddStep(Step{
		Name: "step1",
		Execute: func(ctx context.Context) error {
			capturedValue = ctx.Value(key).(string)
			return nil
		},
		Compensate: func(ctx context.Context) error { return nil },
	})

	ctx := context.WithValue(context.Background(), key, "abc-123")
	result := saga.Execute(ctx)

	require.True(t, result.Success())
	assert.Equal(t, "abc-123", capturedValue)
}
