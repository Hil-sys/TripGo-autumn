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
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// 1. Загружаем конфигурацию из окружения
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка инициализации конфигурации: %v", err)
	}
	log.Printf("Конфигурация успешно загружена. Сервер запустится на %s", cfg.HTTPAddr)

	// В этой лабораторной пул БД мы подключим на следующем шаге,
	// поэтому пока делаем заглушку для проверки доступности базы.
	dbConnected := true

	// 2. Инициализируем роутер chi
	r := chi.NewRouter()

	// Добавляем базовые мидлвари для безопасности и логирования
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// 3. Реализуем системные ручки /health и /ready (Пункт 5.7 Бизнес-правил)
	// /health отвечает 200, пока процесс жив, и НЕ ходит в БД
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"OK"}`))
	})

	// /ready проверяет доступность БД и отвечает 503, если она лежит
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !dbConnected {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"Disrupted", "database":"unavailable"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"Ready", "database":"connected"}`))
	})

	// 4. Настраиваем HTTP-сервер со всеми обязательными таймаутами (Пункт 3 ТЗ)
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       15 * time.Second,
	}

	// 5. Запуск Graceful Shutdown (Сквозное требование №3)
	// Канал для перехвата системных сигналов завершения работы
	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)

	// Запускаем HTTP-сервер в отдельной горутине, чтобы он не блокировал основной поток
	go func() {
		log.Printf("Запуск HTTP-сервера на %s...", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	// Основной поток блокируется здесь и ждет сигналов SIGINT (Ctrl+C) или SIGTERM (от Docker)
	sig := <-shutdownSignals
	log.Printf("Получен системный сигнал %v. Начинаем Graceful Shutdown...", sig)

	// Выделяем сервису ограниченный бюджет времени на вежливую остановку (SHUTDOWN_TIMEOUT)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	// Метод Shutdown закроет слушающие порты, перестанет принимать новые запросы
	// и дождется завершения текущих активных вызовов
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Принудительная остановка по таймауту: %v", err)
	} else {
		log.Println("HTTP-сервер успешно и вежливо остановлен.")
	}

	// Здесь на следующих шагах мы будем закрывать пул подключений к БД: pool.Close()
	log.Println("Сервис полностью завершил работу.")
}
