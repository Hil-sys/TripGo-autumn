package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	LogLevel string
	HTTP     HTTPConfig
	Database DatabaseConfig
}

type HTTPConfig struct {
	Addr              string
	ShutdownTimeout   time.Duration
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	ConnectTimeout  time.Duration
	QueryTimeout    time.Duration
	MaxConnLifetime time.Duration
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("required environment variable DATABASE_URL is missing")
	}

	httpAddr := getEnvWithDefault("HTTP_ADDR", ":8080")
	logLevel := getEnvWithDefault("LOG_LEVEL", "info")

	// 1. Таймауты завершения и сервера
	shutdownTimeout, err := getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}
	httpReadTimeout, err := getEnvDuration("HTTP_READ_TIMEOUT", 5*time.Second)
	if err != nil {
		return nil, err
	}
	httpReadHeaderTimeout, err := getEnvDuration("HTTP_READ_HEADER_TIMEOUT", 2*time.Second)
	if err != nil {
		return nil, err
	}
	httpWriteTimeout, err := getEnvDuration("HTTP_WRITE_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}
	httpIdleTimeout, err := getEnvDuration("HTTP_IDLE_TIMEOUT", 15*time.Second)
	if err != nil {
		return nil, err
	}

	maxConns, err := getEnvInt32("DATABASE_MAX_CONNS", 10)
	if err != nil {
		return nil, err
	}

	minConns, err := getEnvInt32("DATABASE_MIN_CONNS", 2)
	if err != nil {
		return nil, err
	}

	connectTimeout, err := getEnvDuration("DATABASE_CONNECT_TIMEOUT", 5*time.Second)
	if err != nil {
		return nil, err
	}

	queryTimeout, err := getEnvDuration("DATABASE_QUERY_TIMEOUT", 3*time.Second)
	if err != nil {
		return nil, err
	}

	maxConnLifetime, err := getEnvDuration("DATABASE_MAX_CONN_LIFETIME", 30*time.Minute)
	if err != nil {
		return nil, err
	}

	return &Config{
		LogLevel: logLevel,
		HTTP: HTTPConfig{
			Addr:              httpAddr,
			ShutdownTimeout:   shutdownTimeout,
			ReadTimeout:       httpReadTimeout,
			ReadHeaderTimeout: httpReadHeaderTimeout,
			WriteTimeout:      httpWriteTimeout,
			IdleTimeout:       httpIdleTimeout,
		},
		Database: DatabaseConfig{
			URL:             dbURL,
			MaxConns:        maxConns,
			MinConns:        minConns,
			ConnectTimeout:  connectTimeout,
			QueryTimeout:    queryTimeout,
			MaxConnLifetime: maxConnLifetime,
		},
	}, nil
}

func getEnvWithDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) (time.Duration, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal, nil
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for environment variable %s: %w", key, err)
	}
	return d, nil
}

func getEnvInt32(key string, defaultVal int32) (int32, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal, nil
	}
	intVal, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid integer for environment variable %s: %w", key, err)
	}
	return int32(intVal), nil
}
