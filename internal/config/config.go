// LifeLogger LL-Journal
// https://api.lifelogger.life
// company: Tellurian Corp (https://www.telluriancorp.com)
// created in: December 2025

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds application configuration
type Config struct {
	Host        string `json:"host"`
	Port        uint16 `json:"port"`
	DatabaseURL string `json:"database_url"`
	S3Endpoint  string `json:"s3_endpoint"`
	S3Bucket   string `json:"s3_bucket"`
	S3AccessKey string `json:"s3_access_key"`
	S3SecretKey string `json:"s3_secret_key"`
	GitRoot     string `json:"git_root"`
	LogLevel    string `json:"log_level"`
}

// Default returns default configuration
func Default() *Config {
	return &Config{
		Host:        "0.0.0.0",
		Port:        9002,
		DatabaseURL: "",
		S3Endpoint:  "",
		S3Bucket:    "lifelogger-journals",
		S3AccessKey: "",
		S3SecretKey: "",
		GitRoot:     "/var/lib/ll-journal/git",
		LogLevel:    "info",
	}
}

// Load loads configuration with priority: .env (dev only) -> env vars -> JSON file
func Load() (*Config, error) {
	config := Default()

	// Priority 1: Try to load from .env file (only in development, not in production)
	envMode := os.Getenv("ENV")
	if envMode == "" {
		envMode = os.Getenv("APP_ENV")
	}
	if envMode == "" {
		envMode = "development" // Default to development if not set
	}

	if envMode != "production" {
		if _, err := os.Stat(".env"); err == nil {
			if err := godotenv.Load(); err != nil {
				return nil, fmt.Errorf("failed to load .env file: %w", err)
			}
		}
	}

	// Priority 2: Load from environment variables
	config.loadFromEnv()

	// Priority 3: Try to load from JSON config file (if exists)
	if _, err := os.Stat("config.json"); err == nil {
		jsonConfig, err := LoadFromJSON("config.json")
		if err == nil {
			config.mergeFromJSON(jsonConfig)
		}
	}

	return config, nil
}

// loadFromEnv loads configuration from environment variables
func (c *Config) loadFromEnv() {
	if host := os.Getenv("LL_JOURNAL_HOST"); host != "" {
		c.Host = host
	}

	if portStr := os.Getenv("LL_JOURNAL_PORT"); portStr != "" {
		if port, err := strconv.ParseUint(portStr, 10, 16); err == nil {
			c.Port = uint16(port)
		}
	}

	if dbURL := os.Getenv("LL_JOURNAL_DATABASE_URL"); dbURL != "" {
		c.DatabaseURL = dbURL
	}

	if endpoint := os.Getenv("LL_JOURNAL_S3_ENDPOINT"); endpoint != "" {
		c.S3Endpoint = endpoint
	}

	if bucket := os.Getenv("LL_JOURNAL_S3_BUCKET"); bucket != "" {
		c.S3Bucket = bucket
	}

	if key := os.Getenv("LL_JOURNAL_S3_ACCESS_KEY"); key != "" {
		c.S3AccessKey = key
	}

	if secret := os.Getenv("LL_JOURNAL_S3_SECRET_KEY"); secret != "" {
		c.S3SecretKey = secret
	}

	if root := os.Getenv("LL_JOURNAL_GIT_ROOT"); root != "" {
		c.GitRoot = root
	}

	if level := os.Getenv("LL_JOURNAL_LOG_LEVEL"); level != "" {
		c.LogLevel = level
	}
}

// LoadFromJSON loads configuration from JSON file
func LoadFromJSON(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// mergeFromJSON merges configuration from JSON, only setting values that weren't set from env vars
func (c *Config) mergeFromJSON(jsonConfig *Config) {
	if os.Getenv("LL_JOURNAL_HOST") == "" && jsonConfig.Host != "" {
		c.Host = jsonConfig.Host
	}

	if os.Getenv("LL_JOURNAL_PORT") == "" && jsonConfig.Port != 0 {
		c.Port = jsonConfig.Port
	}

	if os.Getenv("LL_JOURNAL_DATABASE_URL") == "" && jsonConfig.DatabaseURL != "" {
		c.DatabaseURL = jsonConfig.DatabaseURL
	}

	if os.Getenv("LL_JOURNAL_S3_ENDPOINT") == "" && jsonConfig.S3Endpoint != "" {
		c.S3Endpoint = jsonConfig.S3Endpoint
	}

	if os.Getenv("LL_JOURNAL_S3_BUCKET") == "" && jsonConfig.S3Bucket != "" {
		c.S3Bucket = jsonConfig.S3Bucket
	}

	if os.Getenv("LL_JOURNAL_S3_ACCESS_KEY") == "" && jsonConfig.S3AccessKey != "" {
		c.S3AccessKey = jsonConfig.S3AccessKey
	}

	if os.Getenv("LL_JOURNAL_S3_SECRET_KEY") == "" && jsonConfig.S3SecretKey != "" {
		c.S3SecretKey = jsonConfig.S3SecretKey
	}

	if os.Getenv("LL_JOURNAL_GIT_ROOT") == "" && jsonConfig.GitRoot != "" {
		c.GitRoot = jsonConfig.GitRoot
	}

	if os.Getenv("LL_JOURNAL_LOG_LEVEL") == "" && jsonConfig.LogLevel != "" {
		c.LogLevel = jsonConfig.LogLevel
	}
}

// SocketAddr returns the socket address string
func (c *Config) SocketAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
