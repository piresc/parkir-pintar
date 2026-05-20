package health

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const (
	StatusHealthy   = "healthy"
	StatusUnhealthy = "unhealthy"
	KeyStatus       = "status"
)

type Checker interface {
	Check(ctx context.Context) error
	Name() string
}

type Service struct {
	checkers map[string]Checker
	logger   *slog.Logger
	mu       sync.RWMutex
}

func NewService(logger *slog.Logger) *Service {
	return &Service{
		checkers: make(map[string]Checker),
		logger:   logger,
	}
}

func (s *Service) AddChecker(name string, checker Checker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkers[name] = checker
}

func (s *Service) CheckAll(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]interface{})
	dependencies := make(map[string]interface{})
	allHealthy := true

	for name, checker := range s.checkers {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("health check cancelled: %w", err)
		}

		depResult := s.runChecker(ctx, name, checker)
		dependencies[name] = depResult

		if depResult[KeyStatus] != StatusHealthy {
			allHealthy = false
		}
	}

	if allHealthy {
		result[KeyStatus] = StatusHealthy
	} else {
		result[KeyStatus] = StatusUnhealthy
	}
	result["dependencies"] = dependencies

	return result, nil
}

func (s *Service) runChecker(ctx context.Context, name string, checker Checker) map[string]interface{} {
	start := time.Now()

	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := checker.Check(checkCtx)
	duration := time.Since(start)

	depResult := map[string]interface{}{
		"duration_ms": duration.Milliseconds(),
	}

	if err != nil {
		depResult[KeyStatus] = StatusUnhealthy
		depResult["error"] = err.Error()
		s.logger.Error("health check failed",
			slog.String("checker", name),
			slog.String("error", err.Error()),
			slog.Int64("duration_ms", duration.Milliseconds()))
	} else {
		depResult[KeyStatus] = StatusHealthy
	}

	return depResult
}
