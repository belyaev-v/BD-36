package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	RSS           []string      `json:"rss"`
	RequestPeriod time.Duration `json:"request_period"`
	DatabaseURL   string        `json:"database_url"`
	APIHost       string        `json:"api_host"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var raw struct {
		RSS           []string `json:"rss"`
		RequestPeriod int      `json:"request_period"`
		DatabaseURL   string   `json:"database_url"`
		APIHost       string   `json:"api_host"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	if len(raw.RSS) == 0 {
		return nil, fmt.Errorf("config.rss must contain at least one feed URL")
	}

	if raw.RequestPeriod <= 0 {
		return nil, fmt.Errorf("config.request_period must be greater than zero")
	}

	if raw.DatabaseURL == "" {
		return nil, fmt.Errorf("config.database_url must be provided")
	}

	cfg := &Config{
		RSS:           raw.RSS,
		RequestPeriod: time.Duration(raw.RequestPeriod) * time.Minute,
		DatabaseURL:   raw.DatabaseURL,
		APIHost:       raw.APIHost,
	}

	if cfg.APIHost == "" {
		cfg.APIHost = ":8080"
	}

	return cfg, nil
}
