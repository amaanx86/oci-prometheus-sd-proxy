// Package config loads and validates application configuration from a YAML file
// with environment variable overrides for all scalar values.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration struct.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Discovery DiscoveryConfig `yaml:"discovery"`
	Tenancies []TenancyConfig `yaml:"tenancies"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port  int    `yaml:"port"`
	Token string `yaml:"token"`
}

// DiscoveryConfig holds target discovery settings.
type DiscoveryConfig struct {
	TagKey          string        `yaml:"tag_key"`
	TagValue        string        `yaml:"tag_value"`
	LinuxPort       int           `yaml:"linux_port"`
	WindowsPort     int           `yaml:"windows_port"`
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	RateLimitRPS    float64       `yaml:"rate_limit_rps"`
}

// TenancyConfig holds OCI tenancy credentials and scope.
type TenancyConfig struct {
	Name           string   `yaml:"name"`
	Region         string   `yaml:"region"`
	TenancyID      string   `yaml:"tenancy_id"`
	UserID         string   `yaml:"user_id"`
	Fingerprint    string   `yaml:"fingerprint"`
	PrivateKeyPath string   `yaml:"private_key_path"`
	Passphrase     string   `yaml:"passphrase"`
	Compartments   []string `yaml:"compartments"`
}

func defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Discovery: DiscoveryConfig{
			TagKey:          "monitoring",
			TagValue:        "enabled",
			LinuxPort:       9100,
			WindowsPort:     9182,
			RefreshInterval: 5 * time.Minute,
			RateLimitRPS:    10.0,
		},
	}
}

// Load reads configuration from a YAML file and applies env var overrides.
// Priority order: defaults -> config file -> environment variables.
func Load() (*Config, error) {
	cfg := defaults()

	path := envStr("CONFIG_PATH", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read config file %q: %w", path, err)
	}
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config file %q: %w", path, err)
		}
	}

	// Scalar env var overrides
	if v := os.Getenv("SERVER_PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid SERVER_PORT %q: %w", v, err)
		}
		cfg.Server.Port = port
	}
	if v := os.Getenv("SERVER_TOKEN"); v != "" {
		cfg.Server.Token = v
	}
	if v := os.Getenv("DISCOVERY_TAG_KEY"); v != "" {
		cfg.Discovery.TagKey = v
	}
	if v := os.Getenv("DISCOVERY_TAG_VALUE"); v != "" {
		cfg.Discovery.TagValue = v
	}
	if v := os.Getenv("DISCOVERY_LINUX_PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid DISCOVERY_LINUX_PORT %q: %w", v, err)
		}
		cfg.Discovery.LinuxPort = port
	}
	if v := os.Getenv("DISCOVERY_WINDOWS_PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid DISCOVERY_WINDOWS_PORT %q: %w", v, err)
		}
		cfg.Discovery.WindowsPort = port
	}
	if v := os.Getenv("DISCOVERY_REFRESH_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid DISCOVERY_REFRESH_INTERVAL %q: %w", v, err)
		}
		cfg.Discovery.RefreshInterval = d
	}
	if v := os.Getenv("DISCOVERY_RATE_LIMIT_RPS"); v != "" {
		rps, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid DISCOVERY_RATE_LIMIT_RPS %q: %w", v, err)
		}
		cfg.Discovery.RateLimitRPS = rps
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.Server.Token == "" {
		return fmt.Errorf("server token is required - set SERVER_TOKEN env var or server.token in config.yaml")
	}
	if len(c.Tenancies) == 0 {
		return fmt.Errorf("at least one tenancy must be configured in config.yaml")
	}
	for i, t := range c.Tenancies {
		if t.Name == "" {
			return fmt.Errorf("tenancy[%d].name is required", i)
		}
		if t.TenancyID == "" {
			return fmt.Errorf("tenancy[%d] (%s): tenancy_id is required", i, t.Name)
		}
		if t.UserID == "" {
			return fmt.Errorf("tenancy[%d] (%s): user_id is required", i, t.Name)
		}
		if t.Region == "" {
			return fmt.Errorf("tenancy[%d] (%s): region is required", i, t.Name)
		}
		if t.Fingerprint == "" {
			return fmt.Errorf("tenancy[%d] (%s): fingerprint is required", i, t.Name)
		}
		if t.PrivateKeyPath == "" {
			return fmt.Errorf("tenancy[%d] (%s): private_key_path is required", i, t.Name)
		}
		// Note: empty compartments array is OK - will auto-discover all compartments in tenancy
	}
	return nil
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
