package config

import (
	"os"
	"strconv"
	"sync"
)

type Config struct {
	Port         int
	Username     string
	Password     string
	LogLevel     string
	LogTimestamp bool
}

var (
	cfg  *Config
	once sync.Once
)

func Get() *Config {
	once.Do(func() {
		cfg = load()
	})
	return cfg
}

func load() *Config {
	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			port = parsed
		}
	}

	logTimestamp := false
	if os.Getenv("BROKER_LOG_TIMESTAMP") == "true" {
		logTimestamp = true
	}

	return &Config{
		Port:         port,
		Username:     getEnvOrDefault("BROKER_USERNAME", ""),
		Password:     getEnvOrDefault("BROKER_PASSWORD", ""),
		LogLevel:     getEnvOrDefault("BROKER_LOG_LEVEL", "info"),
		LogTimestamp: logTimestamp,
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
