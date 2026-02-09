package coretracer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.NotNil(t, cfg, "Expected non-nil config")
	require.Equal(t, "local", cfg.EnvName, "Expected EnvName to be 'local'")
	require.Equal(t, 5*time.Minute, cfg.StuckFunctionTimeout, "Expected StuckFunctionTimeout to be 5 minutes")
}

func TestValidateConfig(t *testing.T) {
	cfg := &Config{
		EnvName:              "production",
		StuckFunctionTimeout: 10 * time.Second,
	}

	validatedCfg := validateConfig(cfg)

	require.Equal(t, "production", validatedCfg.EnvName, "Expected EnvName to be 'production'")
	require.Equal(t, 10*time.Second, validatedCfg.StuckFunctionTimeout, "Expected StuckFunctionTimeout to be 10 seconds")

	// Test with nil config
	validatedCfg = validateConfig(nil)

	require.Equal(t, "local", validatedCfg.EnvName, "Expected EnvName to be 'local'")
	require.Equal(t, 5*time.Minute, validatedCfg.StuckFunctionTimeout, "Expected StuckFunctionTimeout to be 5 minutes")
}

func TestGlobalTagsMap(t *testing.T) {
	cfg := &Config{
		EnvName: "staging",
	}

	globalTags := cfg.GlobalTagsMap()

	require.Len(t, globalTags, 1, "Expected 1 global tag")
	require.Equal(t, "staging", globalTags["deployment.environment"], "Expected global tag 'deployment.environment' to be 'staging'")
}

func TestGlobalTagsMap_NilConfig(t *testing.T) {
	var cfg *Config

	globalTags := cfg.GlobalTagsMap()

	require.Len(t, globalTags, 4, "Expected 4 global tags")
	require.Equal(t, "local", globalTags["deployment.environment"], "Expected global tag 'deployment.environment' to be 'local'")
	require.Equal(t, "unknown", globalTags["service.name"], "Expected global tag 'service.name' to be 'unknown'")
	require.Equal(t, "dev", globalTags["service.version"], "Expected global tag 'service.version' to be 'dev'")
	require.Equal(t, "svc-us-east", globalTags["deployment.cluster_id"], "Expected global tag 'deployment.cluster_id' to be 'svc-us-east'")
}
