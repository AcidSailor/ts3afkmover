package configs

import (
    "github.com/caarlos0/env/v11"
    "strings"
)

type Config struct {
    Url               string `env:"TS3_URL,required"`
    ApiKey            string `env:"TS3_API_KEY,required"`
    VServerId         int    `env:"TS3_VSERVER_ID" envDefault:"1"`
    IdleTime          int    `env:"TS3_IDLE_TIME" envDefault:"60"` // in minutes
    IdleChannelId     int    `env:"TS3_IDLE_CHANNEL_ID,required"`
    IdleCheckInterval int    `env:"TS3_IDLE_CHECK_INTERVAL" envDefault:"5"` // in minutes
    RequestTimeout    int    `env:"TS3_REQUEST_TIMEOUT" envDefault:"15"`    // in seconds
    MessageTemplate   string `env:"TS3_MESSAGE_TEMPLATE" envDefault:"User %s was moved to Idle Channel"`
}

func NewConfig() (*Config, error) {
    config := &Config{}
    err := env.Parse(config)

    if err != nil {
        return nil, err
    }

    url := strings.TrimSuffix(config.Url, "/")
    config.Url = url

    return config, nil
}
