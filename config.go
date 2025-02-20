package coretracer

import (
	"log"
	"time"
)

type Config struct {
	Enabled               bool
	EnvName               string
	ServiceName           string
	ServiceVersion        string
	ChainID               string
	CollectorDSN          string
	CollectorSecureSSL    bool
	StuckFunctionWatchdog bool
	StuckFunctionTimeout  time.Duration
	Logger                BasicLogger
}

type BasicLogger interface {
	Printf(format string, v ...any)
	Println(v ...any)
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

	if len(c.ChainID) > 0 {
		globalTags["deployment.chain_id"] = c.ChainID
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

	if len(cfg.ChainID) == 0 {
		cfg.ChainID = "localnet-1337"
	}

	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}

	return cfg
}
