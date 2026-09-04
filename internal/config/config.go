// Package config provides structured configuration loading from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all server configuration.
type Config struct {
	DatabaseURL string
	SSHHost     string
	SSHPort     int
	HostKeyPath string
	LogLevel    string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	port, err := strconv.Atoi(envOr("READIT_SSH_PORT", "2222"))
	if err != nil {
		return nil, fmt.Errorf("invalid READIT_SSH_PORT: %w", err)
	}

	cfg := &Config{
		DatabaseURL: envOr("READIT_DATABASE_URL", "postgres://readit:readit_dev@localhost:5432/readit?sslmode=disable"),
		SSHHost:     envOr("READIT_SSH_HOST", "0.0.0.0"),
		SSHPort:     port,
		HostKeyPath: envOr("READIT_HOST_KEY_PATH", ".ssh/host_key"),
		LogLevel:    envOr("READIT_LOG_LEVEL", "info"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("READIT_DATABASE_URL must not be empty")
	}

	return cfg, nil
}

// Address returns the SSH listen address as host:port.
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.SSHHost, c.SSHPort)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
