package config

import (
	"os"
	"strconv"
)

// getEnv retrieves environment variable or returns default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves environment variable as int or returns default
func getEnvAsInt(key string, defaultValue int) int {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

// getEnvAsInt64 retrieves environment variable as int64 or returns default
func getEnvAsInt64(key string, defaultValue int64) int64 {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
			return value
		}
	}
	return defaultValue
}

// getEnvAsUint64 retrieves environment variable as uint64 or returns default
func getEnvAsUint64(key string, defaultValue uint64) uint64 {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := strconv.ParseUint(valueStr, 10, 64); err == nil {
			return value
		}
	}
	return defaultValue
}
