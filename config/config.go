package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Url               string `env:"TS3_URL,required"`
	ApiKey            string `env:"TS3_API_KEY,required"`
	VServerId         int    `env:"TS3_VSERVER_ID"               envDefault:"1"`
	IdleTime          int    `env:"TS3_IDLE_TIME"                envDefault:"60"` // in minutes
	IdleChannelId     int    `env:"TS3_IDLE_CHANNEL_ID,required"`
	IdleCheckInterval int    `env:"TS3_IDLE_CHECK_INTERVAL"      envDefault:"5"`  // in minutes
	RequestTimeout    int    `env:"TS3_REQUEST_TIMEOUT"          envDefault:"15"` // in seconds
	MessageTemplate   string `env:"TS3_MESSAGE_TEMPLATE"         envDefault:"User %s was moved to Idle Channel"`
}

func New() (*Config, error) {
	cfg := &Config{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	cfg.Url = strings.TrimSuffix(cfg.Url, "/")

	// A non-positive interval would panic time.NewTicker; catch it at startup.
	if cfg.IdleCheckInterval <= 0 {
		return nil, fmt.Errorf(
			"TS3_IDLE_CHECK_INTERVAL must be > 0, got %d",
			cfg.IdleCheckInterval,
		)
	}
	if cfg.IdleTime < 0 {
		return nil, fmt.Errorf(
			"TS3_IDLE_TIME must be >= 0, got %d", cfg.IdleTime,
		)
	}
	if cfg.RequestTimeout < 0 {
		return nil, fmt.Errorf(
			"TS3_REQUEST_TIMEOUT must be >= 0, got %d", cfg.RequestTimeout,
		)
	}

	return cfg, nil
}

// IdleThreshold is how long a client may be idle before it is moved.
func (c *Config) IdleThreshold() time.Duration {
	return time.Duration(c.IdleTime) * time.Minute
}

// TickInterval is how often the idle sweep runs.
func (c *Config) TickInterval() time.Duration {
	return time.Duration(c.IdleCheckInterval) * time.Minute
}

// RequestTimeoutDuration is the per-request HTTP timeout.
func (c *Config) RequestTimeoutDuration() time.Duration {
	return time.Duration(c.RequestTimeout) * time.Second
}
