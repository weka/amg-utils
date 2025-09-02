package config

import (
	"fmt"
	"regexp"

	"github.com/spf13/viper"
)

// Config represents the daemon configuration
type Config struct {
	TestTime    string `mapstructure:"test_time"`
	WebPort     int    `mapstructure:"web_port"`
	ResultsPath string `mapstructure:"results_path"`
}

// LoadConfig loads configuration from viper
func LoadConfig() (*Config, error) {
	var cfg Config

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate time format (HH:MM)
	timeRegex := regexp.MustCompile(`^([01]?[0-9]|2[0-3]):[0-5][0-9]$`)
	if !timeRegex.MatchString(c.TestTime) {
		return fmt.Errorf("invalid test_time format '%s', expected HH:MM", c.TestTime)
	}

	// Validate port range
	if c.WebPort < 1 || c.WebPort > 65535 {
		return fmt.Errorf("invalid web_port %d, must be between 1-65535", c.WebPort)
	}

	// Results path cannot be empty
	if c.ResultsPath == "" {
		return fmt.Errorf("results_path cannot be empty")
	}

	return nil
}
