package observability

import (
	"os"
	"strings"
)

const (
	envAccessLogEnabled     = "ACCESS_LOG_ENABLED"
	envHTTPAccessLogEnabled = "HTTP_ACCESS_LOG_ENABLED"
	envGRPCAccessLogEnabled = "GRPC_ACCESS_LOG_ENABLED"
)

// HTTPAccessLogEnabled controls full HTTP access logs.
// Successful requests are skipped by default to avoid log I/O dominating benchmarks.
func HTTPAccessLogEnabled() bool {
	return envBool(envHTTPAccessLogEnabled, envBool(envAccessLogEnabled, false))
}

// GRPCAccessLogEnabled controls full gRPC access logs.
// Non-OK responses are still logged by interceptors even when this is false.
func GRPCAccessLogEnabled() bool {
	return envBool(envGRPCAccessLogEnabled, envBool(envAccessLogEnabled, false))
}

func envBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}
