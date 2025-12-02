// Package config handles environment-based configuration for Hermes.
package config

import (
	"errors"
	"log"
	"os"
	"strconv"
	"time"
)

// Config represents the complete Hermes configuration loaded from environment variables.
type Config struct {
	Server    ServerConfig
	Auth      AuthConfig
	Bootstrap BootstrapConfig
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Host           string
	Port           int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxHeaderBytes int
}

// AuthConfig contains authentication settings.
type AuthConfig struct {
	AegisURL     string
	AegisTimeout time.Duration
}

// BootstrapConfig contains initial admin user settings.
type BootstrapConfig struct {
	AdminUser     string
	AdminPassword string
}

// Load reads configuration from environment variables with sensible defaults.
// All environment variables use the HERMES_ prefix:
//   - HERMES_SERVER_HOST (default: "0.0.0.0")
//   - HERMES_SERVER_PORT (default: 8080)
//   - HERMES_AEGIS_URL (default: "http://localhost:3100/api")
//   - HERMES_ADMIN_USER (default: "hermes")
//   - HERMES_ADMIN_PASSWORD (default: "hermes123")
//
// Returns an error if validation fails (e.g., invalid port number).
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host:           getEnv("HERMES_SERVER_HOST", "0.0.0.0"),
			Port:           getEnvInt("HERMES_SERVER_PORT", 8080),
			ReadTimeout:    getEnvDuration("HERMES_SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:   getEnvDuration("HERMES_SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:    getEnvDuration("HERMES_SERVER_IDLE_TIMEOUT", 60*time.Second),
			MaxHeaderBytes: getEnvInt("HERMES_SERVER_MAX_HEADER_BYTES", 1048576), // 1MB
		},
		Auth: AuthConfig{
			AegisURL:     getEnv("HERMES_AEGIS_URL", "http://localhost:3100/api"),
			AegisTimeout: getEnvDuration("HERMES_AEGIS_TIMEOUT", 5*time.Second),
		},
		Bootstrap: BootstrapConfig{
			AdminUser:     getEnv("HERMES_ADMIN_USER", "hermes"),
			AdminPassword: getEnv("HERMES_ADMIN_PASSWORD", "hermes123"),
		},
	}

	// Validate configuration
	if err := validate(cfg); err != nil {
		log.Printf("Configuration validation failed: %v", err)
		return nil, errors.New("invalid configuration")
	}

	// Log loaded configuration
	log.Printf("Configuration loaded:")
	log.Printf("  Server: %s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("  Aegis URL: %s", cfg.Auth.AegisURL)
	log.Printf("  Bootstrap Admin: %s", cfg.Bootstrap.AdminUser)

	return cfg, nil
}

// validate checks if the configuration is valid.
func validate(cfg *Config) error {
	// Validate server port
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		log.Printf("Invalid server port: %d (must be 1-65535)", cfg.Server.Port)
		return errors.New("invalid server port")
	}

	// Validate timeouts
	if cfg.Server.ReadTimeout <= 0 {
		log.Printf("Invalid read timeout: %v (must be positive)", cfg.Server.ReadTimeout)
		return errors.New("invalid read timeout")
	}
	if cfg.Server.WriteTimeout <= 0 {
		log.Printf("Invalid write timeout: %v (must be positive)", cfg.Server.WriteTimeout)
		return errors.New("invalid write timeout")
	}

	return nil
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt retrieves an integer environment variable or returns a default value.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
		log.Printf("Warning: invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}

// getEnvDuration retrieves a duration environment variable or returns a default value.
// Accepts values like "30s", "5m", "1h"
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		log.Printf("Warning: invalid duration value for %s: %s, using default: %v", key, value, defaultValue)
	}
	return defaultValue
}

// GetLogLevel returns the configured log level from HERMES_LOG_LEVEL.
// Valid values: "debug", "info", "warn", "error"
// Default: "info"
func GetLogLevel() string {
	return getEnv("HERMES_LOG_LEVEL", "info")
}

// IsDebugMode returns true if debug mode is enabled (HERMES_LOG_LEVEL=debug).
func IsDebugMode() bool {
	return GetLogLevel() == "debug"
}
