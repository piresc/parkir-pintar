// Package saga provides a simple saga coordinator for orchestrating
// distributed transactions with compensation logic.
package saga

import (
	"context"
	"fmt"
)

// Step represents a single step in a saga with an execute action
// and a compensating action to undo it on failure.
type Step struct {
	Name       string
	Execute    func(ctx context.Context) error
	Compensate func(ctx context.Context) error
}

// Result holds the outcome of a saga execution, including which steps
// completed, which step failed, and any errors during compensation.
type Result struct {
	// CompletedSteps lists the names of steps that executed successfully.
	CompletedSteps []string

	// FailedStep is the name of the step that caused the saga to fail.
	// Empty if all steps succeeded.
	FailedStep string

	// Error is the error from the failed step, if any.
	Error error

	// CompensationErrors holds errors from compensation steps that failed.
	// Key is the step name, value is the compensation error.
	CompensationErrors map[string]error
}

// Saga orchestrates a sequence of steps with compensation on failure.
// If any step fails, previously completed steps are compensated in
// reverse order.
type Saga struct {
	Name  string
	Steps []Step
}

// NewSaga creates a new Saga with the given name.
func NewSaga(name string) *Saga {
	return &Saga{
		Name: name,
	}
}

// AddStep appends a step to the saga.
func (s *Saga) AddStep(step Step) *Saga {
	s.Steps = append(s.Steps, step)
	return s
}

// Execute runs all saga steps in order. If a step fails, it compensates
// all previously completed steps in reverse order. Context is propagated
// to all step functions for tracing and cancellation support.
func (s *Saga) Execute(ctx context.Context) *Result {
	result := &Result{
		CompletedSteps:     make([]string, 0, len(s.Steps)),
		CompensationErrors: make(map[string]error),
	}

	for i, step := range s.Steps {
		if err := step.Execute(ctx); err != nil {
			result.FailedStep = step.Name
			result.Error = fmt.Errorf("saga %q step %d (%s) failed: %w", s.Name, i, step.Name, err)

			// Compensate completed steps in reverse order
			s.compensate(ctx, result)
			return result
		}
		result.CompletedSteps = append(result.CompletedSteps, step.Name)
	}

	return result
}

// compensate runs compensation for all completed steps in reverse order.
func (s *Saga) compensate(ctx context.Context, result *Result) {
	for i := len(result.CompletedSteps) - 1; i >= 0; i-- {
		stepName := result.CompletedSteps[i]

		// Find the step by name to get its Compensate function
		for _, step := range s.Steps {
			if step.Name == stepName && step.Compensate != nil {
				if err := step.Compensate(ctx); err != nil {
					result.CompensationErrors[stepName] = fmt.Errorf(
						"compensation for step %q failed: %w", stepName, err,
					)
				}
				break
			}
		}
	}
}

// Success returns true if the saga completed without any step failures.
func (r *Result) Success() bool {
	return r.Error == nil
}

// HasCompensationErrors returns true if any compensation steps failed.
func (r *Result) HasCompensationErrors() bool {
	return len(r.CompensationErrors) > 0
}
