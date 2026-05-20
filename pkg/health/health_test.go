package health

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockChecker struct {
	name string
	err  error
}

func (m *mockChecker) Check(_ context.Context) error {
	return m.err
}

func (m *mockChecker) Name() string {
	return m.name
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewService_ShouldReturnService_WhenLoggerProvided(t *testing.T) {
	logger := newTestLogger()

	svc := NewService(logger)

	require.NotNil(t, svc)
	assert.NotNil(t, svc.checkers)
	assert.Equal(t, 0, len(svc.checkers))
}

func TestAddChecker_ShouldRegisterChecker_WhenValidCheckerProvided(t *testing.T) {
	svc := NewService(newTestLogger())
	checker := &mockChecker{name: "test-db", err: nil}

	svc.AddChecker("database", checker)

	assert.Equal(t, 1, len(svc.checkers))
	assert.Equal(t, checker, svc.checkers["database"])
}

func TestCheckAll_ShouldReturnHealthy_WhenAllCheckersPass(t *testing.T) {
	svc := NewService(newTestLogger())
	svc.AddChecker("postgres", &mockChecker{name: "postgres", err: nil})
	svc.AddChecker("redis", &mockChecker{name: "redis", err: nil})
	ctx := context.Background()

	result, err := svc.CheckAll(ctx)

	require.NoError(t, err)
	assert.Equal(t, "healthy", result["status"])
	deps, ok := result["dependencies"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 2, len(deps))
}

func TestCheckAll_ShouldReturnUnhealthy_WhenAnyCheckerFails(t *testing.T) {
	svc := NewService(newTestLogger())
	svc.AddChecker("postgres", &mockChecker{name: "postgres", err: nil})
	svc.AddChecker("redis", &mockChecker{name: "redis", err: errors.New("connection refused")})
	ctx := context.Background()

	result, err := svc.CheckAll(ctx)

	require.NoError(t, err)
	assert.Equal(t, "unhealthy", result["status"])
	deps, ok := result["dependencies"].(map[string]interface{})
	require.True(t, ok)

	redisDep, ok := deps["redis"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "unhealthy", redisDep["status"])
	assert.Equal(t, "connection refused", redisDep["error"])

	pgDep, ok := deps["postgres"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "healthy", pgDep["status"])
}

func TestCheckAll_ShouldReturnHealthy_WhenNoCheckersRegistered(t *testing.T) {
	svc := NewService(newTestLogger())
	ctx := context.Background()

	result, err := svc.CheckAll(ctx)

	require.NoError(t, err)
	assert.Equal(t, "healthy", result["status"])
}

func TestCheckAll_ShouldReturnError_WhenContextCancelled(t *testing.T) {
	svc := NewService(newTestLogger())
	svc.AddChecker("postgres", &mockChecker{name: "postgres", err: nil})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := svc.CheckAll(ctx)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "health check cancelled")
}

func TestCheckAll_ShouldRecordDuration_WhenCheckerRuns(t *testing.T) {
	svc := NewService(newTestLogger())
	svc.AddChecker("fast-check", &mockChecker{name: "fast-check", err: nil})
	ctx := context.Background()

	result, err := svc.CheckAll(ctx)

	require.NoError(t, err)
	deps, ok := result["dependencies"].(map[string]interface{})
	require.True(t, ok)
	dep, ok := deps["fast-check"].(map[string]interface{})
	require.True(t, ok)
	_, hasDuration := dep["duration_ms"]
	assert.True(t, hasDuration, "dependency result should include duration_ms")
}

func TestCheckAll_ShouldNotBlockOtherCheckers_WhenOneCheckerFails(t *testing.T) {
	svc := NewService(newTestLogger())
	svc.AddChecker("failing", &mockChecker{name: "failing", err: errors.New("down")})
	svc.AddChecker("passing", &mockChecker{name: "passing", err: nil})
	svc.AddChecker("also-passing", &mockChecker{name: "also-passing", err: nil})
	ctx := context.Background()

	result, err := svc.CheckAll(ctx)

	require.NoError(t, err)
	deps, ok := result["dependencies"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 3, len(deps))
}
