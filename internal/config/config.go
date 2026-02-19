package config

import (
	"fmt"
	"os"
)

type Config struct {
	RCONHost     string
	RCONPort     string
	RCONPassword string
	ListenAddr   string
	LogLevel     string
}

func Load() (*Config, error) {
	cfg := &Config{
		RCONHost:     envOrDefault("PZMONITOR_RCON_HOST", "127.0.0.1"),
		RCONPort:     envOrDefault("PZMONITOR_RCON_PORT", "27015"),
		RCONPassword: os.Getenv("PZMONITOR_RCON_PASSWORD"),
		ListenAddr:   envOrDefault("PZMONITOR_LISTEN_ADDR", ":9101"),
		LogLevel:     envOrDefault("PZMONITOR_LOG_LEVEL", "info"),
	}

	if cfg.RCONPassword == "" {
		return nil, fmt.Errorf("PZMONITOR_RCON_PASSWORD is required")
	}

	return cfg, nil
}

func (c *Config) RCONAddr() string {
	return c.RCONHost + ":" + c.RCONPort
}

func envOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
