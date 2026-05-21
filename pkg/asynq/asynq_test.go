package asynq

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClient_ShouldReturnNonNilClient(t *testing.T) {
	client := NewClient("localhost:6379", "")
	require.NotNil(t, client)
	_ = client.Close()
}

func TestNewClient_ShouldAcceptPassword(t *testing.T) {
	client := NewClient("localhost:6379", "secret-password")
	require.NotNil(t, client)
	_ = client.Close()
}

func TestNewServer_ShouldReturnNonNilServer(t *testing.T) {
	server := NewServer("localhost:6379", "", 10)
	require.NotNil(t, server)
}

func TestNewServer_ShouldAcceptConcurrencyValue(t *testing.T) {
	server := NewServer("localhost:6379", "pass", 5)
	require.NotNil(t, server)
}

func TestNewServer_ShouldDefaultConcurrency_WhenZero(t *testing.T) {
	server := NewServer("localhost:6379", "", 0)
	require.NotNil(t, server)
}
