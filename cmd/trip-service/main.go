package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Hil-sys/TripGo-autumn/internal/config"
	"github.com/Hil-sys/TripGo-autumn/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка инициализации конфигурации: %v", err)
	}

	// 2. Настраиваем пул подключений к PostgreSQL (Пункт 3.6 ТЗ)
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Некорректная строка подключения к БД: %v", err)
	}

	// Выставляем лимиты и таймауты пула из конфига
	poolConfig.MaxConns = cfg.DatabaseMaxConns
	poolConfig.MinConns = cfg.DatabaseMinConns
	poolConfig.MaxConnLifetime = cfg.DatabaseMaxConnLifetime

	// Создаем контекст для подключения с таймаутом по ТЗ
	initCtx, initCancel := context.WithTimeout(context.Background(), cfg.DatabaseConnectTimeout)
	defer initCancel()

	// Инициализируем пул
	pool, err := pgxpool.NewWithConfig(initCtx, poolConfig)
	if err != nil {
		log.Fatalf("Не удалось создать пул подключений к БД: %v", err)
	}

	// Обязательный стартовый Ping. Если база недоступна — сервис НЕ стартует (Требование ТЗ!)
	if err := pool.Ping(initCtx); err != nil {
		log.Fatalf("База данных недоступна на старте приложения: %v", err)
	}
	log.Println("Успешное подключение к PostgreSQL (Ping OK)")

	// 3. Инициализируем роутер chi
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Системная ручка /health (отвечает 200, пока жив процесс, в БД НЕ ходит)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"OK"}`))
	})

	// Системная ручка /ready (честно проверяет доступность БД через быстрый Ping)
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Для проверки готовности выделяем жесткий таймаут из конфига
		pingCtx, pingCancel := context.WithTimeout(r.Context(), cfg.DatabaseQueryTimeout)
		defer pingCancel()

		if err := pool.Ping(pingCtx); err != nil {
			log.Printf("Пинг базы данных провален внутри /ready: %v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"Disrupted", "database":"unavailable"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"Ready", "database":"connected"}`))
	})

	// 4. Настраиваем HTTP-сервер
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       15 * time.Second,
	}

	// 5. Запуск Graceful Shutdown
	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Запуск HTTP-сервера на %s...", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	sig := <-shutdownSignals
	log.Printf("Получен системный сигнал %v. Начинаем Graceful Shutdown...", sig)

	// Бюджет времени на остановку сервера
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	// Вежливо тушим веб-сервер
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Принудительная остановка сервера по таймауту: %v", err)
	} else {
		log.Println("HTTP-сервер успешно и вежливо остановлен.")
	}

	// ЗАКРЫВАЕМ ПУЛ БД ПОСЛЕ ОСТАНОВКИ СЕРВЕРА (Сквозное требование №3)
	log.Println("Закрываем пул подключений к PostgreSQL...")
	pool.Close()

	log.Println("Сервис полностью завершил работу.")
}
