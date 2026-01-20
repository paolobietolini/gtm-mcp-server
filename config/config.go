package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the GTM MCP Server.
type Config struct {
	// Server configuration
	Port    int
	BaseURL string

	// Google OAuth configuration
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURI  string

	// JWT configuration
	JWTSecret string

	// Logging
	LogLevel string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Port:              getEnvInt("PORT", 8081),
		BaseURL:           getEnv("BASE_URL", "http://localhost:8081"),
		GoogleClientID:    getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURI: getEnv("GOOGLE_REDIRECT_URI", ""),
		JWTSecret:         getEnv("JWT_SECRET", ""),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
	}

	// Validation is deferred to when auth is actually needed
	// This allows the server to start and respond to initialize/ping
	// even without OAuth credentials configured

	return cfg, nil
}

// ValidateAuth checks if OAuth credentials are configured.
func (c *Config) ValidateAuth() error {
	if c.GoogleClientID == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID is required for authentication")
	}
	if c.GoogleClientSecret == "" {
		return fmt.Errorf("GOOGLE_CLIENT_SECRET is required for authentication")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required for authentication")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
