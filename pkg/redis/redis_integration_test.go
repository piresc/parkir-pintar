// Package redis provides a Redis client with connection pooling and data structure support.
//
// Best practices applied (from coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - miniredis/v2 for in-memory Redis testing without real server
// - go-redis/redis/v8 client pointing to miniredis addr
// - Each test gets a fresh miniredis instance for isolation
// - Test serialization and TTL behavior
package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMiniredis creates a miniredis server and a RedisClient pointing to it.
func setupMiniredis(t *testing.T) (*RedisClient, *miniredis.Miniredis, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	rc := &RedisClient{client: client}

	cleanup := func() {
		_ = client.Close()
		mr.Close()
	}
	return rc, mr, cleanup
}

func TestSetGet_ShouldRoundtrip_WhenValueIsStored(t *testing.T) {
	// Arrange
	rc, _, cleanup := setupMiniredis(t)
	defer cleanup()
	ctx := context.Background()

	// Act
	err := rc.Set(ctx, "greeting", "hello", 5*time.Minute)
	require.NoError(t, err)

	val, err := rc.Get(ctx, "greeting")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "hello", val)
}

func TestGet_ShouldReturnError_WhenKeyDoesNotExist(t *testing.T) {
	// Arrange
	rc, _, cleanup := setupMiniredis(t)
	defer cleanup()
	ctx := context.Background()

	// Act
	_, err := rc.Get(ctx, "nonexistent")

	// Assert
	assert.ErrorIs(t, err, redis.Nil)
}

func TestDelete_ShouldRemoveKey_WhenKeyExists(t *testing.T) {
	// Arrange
	rc, _, cleanup := setupMiniredis(t)
	defer cleanup()
	ctx := context.Background()

	err := rc.Set(ctx, "to-delete", "value", 0)
	require.NoError(t, err)

	// Act
	err = rc.Delete(ctx, "to-delete")
	require.NoError(t, err)

	_, err = rc.Get(ctx, "to-delete")

	// Assert
	assert.ErrorIs(t, err, redis.Nil)
}

func TestHMSetHGetAll_ShouldRoundtrip_WhenHashFieldsAreSet(t *testing.T) {
	// Arrange
	rc, _, cleanup := setupMiniredis(t)
	defer cleanup()
	ctx := context.Background()

	fields := map[string]interface{}{
		"name":  "John",
		"email": "john@example.com",
		"age":   "30",
	}

	// Act
	err := rc.HMSet(ctx, "user:1", fields)
	require.NoError(t, err)

	result, err := rc.HGetAll(ctx, "user:1")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "John", result["name"])
	assert.Equal(t, "john@example.com", result["email"])
	assert.Equal(t, "30", result["age"])
}
