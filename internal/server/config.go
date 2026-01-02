package server

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string
	Host string

	DatabaseURL string

	JWTSecret     string
	JWTExpiryDays int

	LogLevel string
	LogFile  string

	WSPingInterval time.Duration
	WSPongTimeout  time.Duration
}

func LoadConfig() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	config := &Config{}

	config.Port = getEnvWithDefault("TASKTIME_PORT", "8080")
	config.Host = getEnvWithDefault("TASKTIME_HOST", "0.0.0.0")

	config.DatabaseURL = os.Getenv("NEON_DATABASE_URL")
	if config.DatabaseURL == "" {
		config.DatabaseURL = os.Getenv("DATABASE_URL")
		if config.DatabaseURL == "" {
			return nil, fmt.Errorf("NEON_DATABASE_URL environment variable is required")
		}
	}

	config.JWTSecret = os.Getenv("JWT_SECRET")
	if config.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	if len(config.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters long, got %d characters", len(config.JWTSecret))
	}

	jwtExpiryStr := getEnvWithDefault("JWT_EXPIRY_DAYS", "7")
	jwtExpiry, err := strconv.Atoi(jwtExpiryStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_DAYS value: %s", jwtExpiryStr)
	}
	config.JWTExpiryDays = jwtExpiry

	config.LogLevel = getEnvWithDefault("LOG_LEVEL", "info")
	config.LogFile = os.Getenv("LOG_FILE") // Can be empty for stdout logging

	wsPingIntervalStr := getEnvWithDefault("WS_PING_INTERVAL", "30s")
	wsPingInterval, err := time.ParseDuration(wsPingIntervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid WS_PING_INTERVAL value: %s", wsPingIntervalStr)
	}
	config.WSPingInterval = wsPingInterval

	wsPongTimeoutStr := getEnvWithDefault("WS_PONG_TIMEOUT", "60s")
	wsPongTimeout, err := time.ParseDuration(wsPongTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid WS_PONG_TIMEOUT value: %s", wsPongTimeoutStr)
	}
	config.WSPongTimeout = wsPongTimeout

	return config, nil
}

func (c *Config) ValidateConfig() error {
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid LOG_LEVEL: %s (must be one of: debug, info, warn, error)", c.LogLevel)
	}

	// Validate JWT expiry days
	if c.JWTExpiryDays <= 0 {
		return fmt.Errorf("JWT_EXPIRY_DAYS must be positive, got %d", c.JWTExpiryDays)
	}

	// Validate WebSocket timeouts
	if c.WSPingInterval <= 0 {
		return fmt.Errorf("WS_PING_INTERVAL must be positive, got %v", c.WSPingInterval)
	}
	if c.WSPongTimeout <= 0 {
		return fmt.Errorf("WS_PONG_TIMEOUT must be positive, got %v", c.WSPongTimeout)
	}
	if c.WSPongTimeout <= c.WSPingInterval {
		return fmt.Errorf("WS_PONG_TIMEOUT (%v) must be greater than WS_PING_INTERVAL (%v)", c.WSPongTimeout, c.WSPingInterval)
	}

	return nil
}

// getEnvWithDefault returns the value of the environment variable or the default value if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
