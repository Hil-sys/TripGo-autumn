package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr              string
	DatabaseURL           string
	LogLevel              string
	ShutdownTimeout       time.Duration
	HTTPReadTimeout       time.Duration
	HTTPReadHeaderTimeout time.Duration
	HTTPWriteTimeout      time.Duration
	HTTPIdleTimeout       time.Duration
	DatabaseMaxConns      int32
	DatabaseMinConns      int32
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
	DatabaseMaxConnLifetime time.Duration
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("required environment variable DATABASE_URL is missing")
	}

	httpAddr := getEnvWithDefault("HTTP_ADDR", ":8080")
	logLevel := getEnvWithDefault("LOG_LEVEL", "info")

	shutdownTimeout, err := getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}

	// HTTP-таймауты сервера
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

	// Параметры пула БД и таймауты
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
		HTTPAddr:                httpAddr,
		DatabaseURL:             dbURL,
		LogLevel:                logLevel,
		ShutdownTimeout:         shutdownTimeout,
		HTTPReadTimeout:         httpReadTimeout,
		HTTPReadHeaderTimeout:   httpReadHeaderTimeout,
		HTTPWriteTimeout:        httpWriteTimeout,
		HTTPIdleTimeout:         httpIdleTimeout,
		DatabaseMaxConns:        maxConns,
		DatabaseMinConns:        minConns,
		DatabaseConnectTimeout:  connectTimeout,
		DatabaseQueryTimeout:    queryTimeout,
		DatabaseMaxConnLifetime: maxConnLifetime,
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
