package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	ErrMissingEnvironmentVariable = errors.New("missing environment variable")
	ErrInvalidEnvironmentVariable = errors.New("invalid environment variable")
)

// Config contains the application configuration.
type Config struct {
	Environment string
	HTTP        HTTPConfig
	Log         LogConfig
}

// HTTPConfig with params
type HTTPConfig struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	IdleTimeout     time.Duration
}

// LogConfig with level
type LogConfig struct {
	Level string
}

// Load reads and validates configuration from environment variables.
func Load() (Config, error) {
	environment, err := requiredEnv("APP_ENV")
	if err != nil {
		return Config{}, err
	}

	address, err := requiredEnv("HTTP_ADDR")
	if err != nil {
		return Config{}, err
	}
	if err := validateHTTPAddress("HTTP_ADDR", address); err != nil {
		return Config{}, err
	}

	readTimeout, err := requiredDuration("HTTP_READ_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := requiredDuration("HTTP_WRITE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := requiredDuration("HTTP_SHUTDOWN_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	idleTimeout, err := requiredDuration("HTTP_IDLE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	logLevel, err := requiredEnv("LOG_LEVEL")
	if err != nil {
		return Config{}, err
	}
	logLevel, err = validateLogLevel("LOG_LEVEL", logLevel)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Environment: environment,
		HTTP: HTTPConfig{
			Address:         address,
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			ShutdownTimeout: shutdownTimeout,
			IdleTimeout:     idleTimeout,
		},
		Log: LogConfig{
			Level: logLevel,
		},
	}, nil
}

func requiredEnv(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return "", fmt.Errorf("%w: %s", ErrMissingEnvironmentVariable, key)
	}

	return value, nil
}

func requiredDuration(key string) (time.Duration, error) {
	value, err := requiredEnv(key)
	if err != nil {
		return 0, err
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf(
			"%w: %s=%q: %v",
			ErrInvalidEnvironmentVariable,
			key,
			value,
			err,
		)
	}
	if duration <= 0 {
		return 0, fmt.Errorf(
			"%w: %s=%q: duration must be positive",
			ErrInvalidEnvironmentVariable,
			key,
			value,
		)
	}

	return duration, nil
}

func validateHTTPAddress(key, value string) error {
	_, port, err := net.SplitHostPort(value)
	if err != nil {
		return fmt.Errorf(
			"%w: %s=%q: %v",
			ErrInvalidEnvironmentVariable,
			key,
			value,
			err,
		)
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf(
			"%w: %s=%q: port must be a number between 1 and 65535",
			ErrInvalidEnvironmentVariable,
			key,
			value,
		)
	}

	return nil
}

func validateLogLevel(key, value string) (string, error) {
	level := strings.ToLower(value)
	switch level {
	case "debug", "info", "warn", "error":
		return level, nil
	default:
		return "", fmt.Errorf(
			"%w: %s=%q: expected debug, info, warn, or error",
			ErrInvalidEnvironmentVariable,
			key,
			value,
		)
	}
}
