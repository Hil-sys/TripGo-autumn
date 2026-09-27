package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	LogLevel        string
	ShutdownTimeout time.Duration

	// Параметры пула pgxpool
	MaxConns        int32
	MinConns        int32
	ConnectTimeout  time.Duration
	QueryTimeout    time.Duration
	MaxConnLifetime time.Duration
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" { // если нет поля - падение
		return nil, fmt.Errorf("обязательная переменная окружения DATABASE_URL не задана")
	}

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	// парсим таймаут остановки
	shutdownStr := os.Getenv("SHUTDOWN_TIMEOUT")
	if shutdownStr == "" {
		shutdownStr = "10s"
	}
	shutdownTimeout, err := time.ParseDuration(shutdownStr)
	if err != nil {
		shutdownTimeout = 10 * time.Second
	}

	// парсим числовые параметры пула
	maxConns := int32(getEnvInt("DATABASE_MAX_CONNS", 10))
	minConns := int32(getEnvInt("DATABASE_MIN_CONNS", 2))

	return &Config{
		HTTPAddr:        httpAddr,
		DatabaseURL:     dbURL,
		LogLevel:        logLevel,
		ShutdownTimeout: shutdownTimeout,
		MaxConns:        maxConns,
		MinConns:        minConns,
	}, nil
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}
