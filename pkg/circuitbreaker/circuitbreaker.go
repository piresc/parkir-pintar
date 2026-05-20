// service calls. It prevents cascade failures by failing fast when a
package circuitbreaker

import (
	"errors"
	"time"

	"github.com/sony/gobreaker/v2"
)

type State int

const (
	StateClosed   State = State(gobreaker.StateClosed)
	StateHalfOpen State = State(gobreaker.StateHalfOpen)
	StateOpen     State = State(gobreaker.StateOpen)
)

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

var ErrCircuitOpen = gobreaker.ErrOpenState

type Config struct {
	Name              string
	FailureThreshold  int
	OpenTimeout       time.Duration
	HalfOpenMaxProbes int
	Interval          time.Duration
}

type CircuitBreaker struct {
	cb *gobreaker.CircuitBreaker[any]
}

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
	if cfg.Interval <= 0 {
		cfg.Interval = 60 * time.Second
	}

	settings := gobreaker.Settings{
		Name:        cfg.Name,
		Interval:    cfg.Interval,
		Timeout:     cfg.OpenTimeout,
		MaxRequests: uint32(cfg.HalfOpenMaxProbes),
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return int(counts.ConsecutiveFailures) >= cfg.FailureThreshold
		},
	}

	return &CircuitBreaker{
		cb: gobreaker.NewCircuitBreaker[any](settings),
	}
}

func (c *CircuitBreaker) Execute(fn func() error) error {
	_, err := c.cb.Execute(func() (any, error) {
		return nil, fn()
	})
	// Map gobreaker error to our domain error.
	if errors.Is(err, gobreaker.ErrTooManyRequests) {
		return ErrCircuitOpen
	}
	return err
}

func (c *CircuitBreaker) State() State {
	return State(c.cb.State())
}
