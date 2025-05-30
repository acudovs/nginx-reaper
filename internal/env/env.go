// Package env provides functions to work with environment variables.
package env

import (
	"nginx-reaper/internal/log"
	"os"
	"strconv"
	"time"
)

// GetDuration retrieves a time.Duration from the specified environment variable.
func GetDuration(envName string, defaultValue string) time.Duration {
	return parseValue(envName, defaultValue, time.ParseDuration)
}

// GetInt retrieves an integer from the specified environment variable.
func GetInt(envName string, defaultValue string) int {
	return parseValue(envName, defaultValue, strconv.Atoi)
}

// GetLogLevel retrieves a log.Level from the specified environment variable.
func GetLogLevel(envName string, defaultValue string) log.Level {
	return parseValue(envName, defaultValue, log.ParseLevel)
}

// GetString retrieves a string from the specified environment variable.
func GetString(envName string, defaultValue string) string {
	return parseValue(envName, defaultValue, func(s string) (string, error) { return s, nil })
}

// parseValue parses the environment variable value using the provided parser function.
// If the environment variable is not set or the value is invalid, the provided default value is returned.
// If the default value is invalid, the program panics.
func parseValue[T any](envName string, defaultValue string, parser func(string) (T, error)) T {
	def, err := parser(defaultValue)
	if err != nil {
		log.Panicf("Invalid default value, %v", err)
	}
	if envValue, ok := os.LookupEnv(envName); ok {
		value, err := parser(envValue)
		if err == nil {
			return value
		}
		log.Errorf("Invalid environment variable %v, %v, using default %v", envName, err, def)
	}
	return def
}
