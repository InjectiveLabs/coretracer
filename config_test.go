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
	require.Equal(t, "staging", globalTags["env"], "Expected global tag 'env' to be 'staging'")
}

func TestGlobalTagsMap_NilConfig(t *testing.T) {
	var cfg *Config

	globalTags := cfg.GlobalTagsMap()

	require.Len(t, globalTags, 1, "Expected 1 global tag")
	require.Equal(t, "local", globalTags["env"], "Expected global tag 'env' to be 'local'")
}
