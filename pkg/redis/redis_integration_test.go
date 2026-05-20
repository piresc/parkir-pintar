package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMiniredis(t *testing.T) (*Client, *miniredis.Miniredis, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	rc := &Client{client: client}

	cleanup := func() {
		_ = client.Close()
		mr.Close()
	}
	return rc, mr, cleanup
}

func TestSetGet_ShouldRoundtrip_WhenValueIsStored(t *testing.T) {
	rc, _, cleanup := setupMiniredis(t)
	defer cleanup()
	ctx := context.Background()

	err := rc.Set(ctx, "greeting", "hello", 5*time.Minute)
	require.NoError(t, err)

	val, err := rc.Get(ctx, "greeting")

	require.NoError(t, err)
	assert.Equal(t, "hello", val)
}

func TestGet_ShouldReturnError_WhenKeyDoesNotExist(t *testing.T) {
	rc, _, cleanup := setupMiniredis(t)
	defer cleanup()
	ctx := context.Background()

	_, err := rc.Get(ctx, "nonexistent")

	assert.ErrorIs(t, err, redis.Nil)
}

func TestDelete_ShouldRemoveKey_WhenKeyExists(t *testing.T) {
	rc, _, cleanup := setupMiniredis(t)
	defer cleanup()
	ctx := context.Background()

	err := rc.Set(ctx, "to-delete", "value", 0)
	require.NoError(t, err)

	err = rc.Delete(ctx, "to-delete")
	require.NoError(t, err)

	_, err = rc.Get(ctx, "to-delete")

	assert.ErrorIs(t, err, redis.Nil)
}

func TestHMSetHGetAll_ShouldRoundtrip_WhenHashFieldsAreSet(t *testing.T) {
	rc, _, cleanup := setupMiniredis(t)
	defer cleanup()
	ctx := context.Background()

	fields := map[string]interface{}{
		"name":  "John",
		"email": "john@example.com",
		"age":   "30",
	}

	err := rc.HMSet(ctx, "user:1", fields)
	require.NoError(t, err)

	result, err := rc.HGetAll(ctx, "user:1")

	require.NoError(t, err)
	assert.Equal(t, "John", result["name"])
	assert.Equal(t, "john@example.com", result["email"])
	assert.Equal(t, "30", result["age"])
}
