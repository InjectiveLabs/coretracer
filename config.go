package coretracer

import (
	"log/slog"
	"time"
)

type Config struct {
	Enabled               bool
	EnvName               string
	ServiceName           string
	ServiceVersion        string
	ClusterID             string
	CollectorDSN          string
	CollectorSecureSSL    bool
	CollectorHeaders      map[string]string
	StuckFunctionWatchdog bool
	StuckFunctionTimeout  time.Duration
	Logger                BasicLogger
}

type BasicLogger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// GlobalTagsMap returns an unsafe map of global tags.
// Allows to convert the global tags into Tags type.
// These tags are global and added to all traces.
func (c *Config) GlobalTagsMap() map[string]string {
	if c == nil {
		c = DefaultConfig()
	}

	globalTags := make(map[string]string, 1)

	if len(c.EnvName) > 0 {
		globalTags["deployment.environment"] = c.EnvName
	}

	if len(c.ClusterID) > 0 {
		globalTags["deployment.cluster_id"] = c.ClusterID
	}

	if len(c.ServiceName) > 0 {
		globalTags["service.name"] = c.ServiceName
	}

	if len(c.ServiceVersion) > 0 {
		globalTags["service.version"] = c.ServiceVersion
	}

	return globalTags
}

// DefaultConfig returns a default config with sane defaults.
func DefaultConfig() *Config {
	return validateConfig(nil)
}

func validateConfig(cfg *Config) *Config {
	if cfg == nil {
		cfg = &Config{}
	}

	if cfg.StuckFunctionTimeout < time.Second {
		cfg.StuckFunctionTimeout = 5 * time.Minute
	}

	if len(cfg.EnvName) == 0 {
		cfg.EnvName = "local"
	}

	if len(cfg.ServiceName) == 0 {
		cfg.ServiceName = "unknown"
	}

	if len(cfg.ServiceVersion) == 0 {
		cfg.ServiceVersion = "dev"
	}

	if len(cfg.ClusterID) == 0 {
		cfg.ClusterID = "svc-us-east"
	}

	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return cfg
}
