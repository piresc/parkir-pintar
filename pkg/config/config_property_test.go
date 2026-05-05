package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: grpc-jwt-pkg-integration, Property 18: Config defaults for unset environment variables
// **Validates: Requirements 12.3**
//
// For any subset of gRPC environment variables that are set, the Config loader
// SHALL use the provided values for set variables and sensible defaults
// (port: 9090, dial timeout: 5s, keep-alive interval: 30s, keep-alive timeout: 10s)
// for unset variables.

// grpcEnvSubset represents a randomly chosen subset of gRPC env vars with random valid values.
type grpcEnvSubset struct {
	PortSet             bool
	Port                int
	DialTimeoutSet      bool
	DialTimeout         time.Duration
	KeepAliveTimeSet    bool
	KeepAliveTime       time.Duration
	KeepAliveTimeoutSet bool
	KeepAliveTimeout    time.Duration
}

func TestProperty18_ConfigDefaultsForUnsetEnvVars(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		subset := grpcEnvSubset{
			PortSet:             rapid.Bool().Draw(t, "portSet"),
			Port:                rapid.IntRange(1, 65535).Draw(t, "port"),
			DialTimeoutSet:      rapid.Bool().Draw(t, "dialTimeoutSet"),
			DialTimeout:         time.Duration(rapid.IntRange(1, 60).Draw(t, "dialTimeoutSec")) * time.Second,
			KeepAliveTimeSet:    rapid.Bool().Draw(t, "keepAliveTimeSet"),
			KeepAliveTime:       time.Duration(rapid.IntRange(1, 120).Draw(t, "keepAliveTimeSec")) * time.Second,
			KeepAliveTimeoutSet: rapid.Bool().Draw(t, "keepAliveTimeoutSet"),
			KeepAliveTimeout:    time.Duration(rapid.IntRange(1, 60).Draw(t, "keepAliveTimeoutSec")) * time.Second,
		}

		// Clear all gRPC env vars first
		grpcEnvVars := []string{
			"GRPC_SERVER_PORT", "GRPC_TLS_CERT_PATH", "GRPC_TLS_KEY_PATH",
			"GRPC_MAX_CONN_AGE", "GRPC_DIAL_TIMEOUT", "GRPC_KEEPALIVE_TIME",
			"GRPC_KEEPALIVE_TIMEOUT",
		}
		for _, v := range grpcEnvVars {
			os.Unsetenv(v)
		}

		// Also clear vars that affect validation so Load succeeds
		os.Unsetenv("APP_ENV")
		os.Setenv("APP_ENV", "local")
		// Ensure SERVER_PORT is valid so validation passes
		os.Setenv("SERVER_PORT", "8080")

		// Set only the chosen subset of gRPC env vars
		if subset.PortSet {
			os.Setenv("GRPC_SERVER_PORT", fmt.Sprintf("%d", subset.Port))
		}
		if subset.DialTimeoutSet {
			os.Setenv("GRPC_DIAL_TIMEOUT", subset.DialTimeout.String())
		}
		if subset.KeepAliveTimeSet {
			os.Setenv("GRPC_KEEPALIVE_TIME", subset.KeepAliveTime.String())
		}
		if subset.KeepAliveTimeoutSet {
			os.Setenv("GRPC_KEEPALIVE_TIMEOUT", subset.KeepAliveTimeout.String())
		}

		cfg, err := Load("")
		require.NoError(t, err)

		// Defaults
		const defaultPort = 9090
		defaultDialTimeout := 5 * time.Second
		defaultKeepAliveTime := 30 * time.Second
		defaultKeepAliveTimeout := 10 * time.Second

		// Verify: set vars use provided values, unset vars use defaults
		if subset.PortSet {
			assert.Equal(t, subset.Port, cfg.GRPC.Server.Port,
				"GRPC_SERVER_PORT was set; expected provided value")
		} else {
			assert.Equal(t, defaultPort, cfg.GRPC.Server.Port,
				"GRPC_SERVER_PORT was unset; expected default 9090")
		}

		if subset.DialTimeoutSet {
			assert.Equal(t, subset.DialTimeout, cfg.GRPC.Client.DialTimeout,
				"GRPC_DIAL_TIMEOUT was set; expected provided value")
		} else {
			assert.Equal(t, defaultDialTimeout, cfg.GRPC.Client.DialTimeout,
				"GRPC_DIAL_TIMEOUT was unset; expected default 5s")
		}

		if subset.KeepAliveTimeSet {
			assert.Equal(t, subset.KeepAliveTime, cfg.GRPC.Client.KeepAliveTime,
				"GRPC_KEEPALIVE_TIME was set; expected provided value")
		} else {
			assert.Equal(t, defaultKeepAliveTime, cfg.GRPC.Client.KeepAliveTime,
				"GRPC_KEEPALIVE_TIME was unset; expected default 30s")
		}

		if subset.KeepAliveTimeoutSet {
			assert.Equal(t, subset.KeepAliveTimeout, cfg.GRPC.Client.KeepAliveTimeout,
				"GRPC_KEEPALIVE_TIMEOUT was set; expected provided value")
		} else {
			assert.Equal(t, defaultKeepAliveTimeout, cfg.GRPC.Client.KeepAliveTimeout,
				"GRPC_KEEPALIVE_TIMEOUT was unset; expected default 10s")
		}

		// Cleanup
		for _, v := range grpcEnvVars {
			os.Unsetenv(v)
		}
		os.Unsetenv("SERVER_PORT")
	})
}
