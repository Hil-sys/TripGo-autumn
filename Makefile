ifneq (,$(wildcard .env))
    include .env
    export
endif

DATABASE_URL ?= postgres://tripgo:tripgo@localhost:5432/tripgo?sslmode=disable
MIGRATIONS_DIR ?= migrations

.PHONY: all build run migrate migrate-down generate test lint clean

all: lint test build

# Сборка бинарника сервиса
build:
	go build -v -o bin/trip-service ./cmd/trip-service

# Локальный запуск сервиса
run:
	go run ./cmd/trip-service

# Применение миграций базы данных
migrate:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

# Откат последней миграции
migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

# Вся кодогенерация проекта (OpenAPI и mockgen)
generate:
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml
	go generate ./...

# Запуск unit-тестов с детектором гонок и повторным прогоном
test:
	go test -v -race -count=2 ./...

# Статический анализ кода через golangci-lint
lint:
	golangci-lint run ./...

# Очистка артефактов сборки
clean:
	rm -rf bin/
	