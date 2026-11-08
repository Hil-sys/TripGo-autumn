package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Hil-sys/TripGo-autumn/internal/config"
	"github.com/Hil-sys/TripGo-autumn/internal/handler"
	"github.com/Hil-sys/TripGo-autumn/internal/repository"
	"github.com/Hil-sys/TripGo-autumn/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	api "github.com/Hil-sys/TripGo-autumn/internal/api"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка конфигурации: %v", err)
	}

	// Вся инициализация и валидация пула спрятана в модуль repository
	pool, err := repository.NewPool(context.Background(), cfg.Database)
	if err != nil {
		log.Fatalf("Ошибка инициализации базы данных: %v", err)
	}
	defer pool.Close()
	log.Println("Успешное подключение к PostgreSQL (Ping OK)")

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	txManager := repository.NewTxManager(pool)
	tripRepo := repository.NewTripRepository(txManager, cfg.Database.QueryTimeout)

	tripUsecase := usecase.NewTripUsecase(tripRepo, txManager)
	tripHandler := handler.NewTripHandler(tripUsecase)

	// Регистрация маршрутов OpenAPI с обработкой ошибок RFC 9457
	api.HandlerWithOptions(tripHandler, api.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusBadRequest)

			detail := err.Error()
			problem := api.Problem{
				Type:     "https://tripgo.example/problems/invalid_request",
				Title:    "Bad Request",
				Status:   int32(http.StatusBadRequest),
				Detail:   &detail,
				Instance: &r.URL.Path,
				Code:     "invalid_request",
			}

			_ = json.NewEncoder(w).Encode(problem)
		},
	})

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           r,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:	   cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Запуск HTTP-сервера на %s...", cfg.HTTP.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	sig := <-shutdownSignals
	log.Printf("Получен сигнал %v. Начинаем Graceful Shutdown...", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Принудительная остановка: %v", err)
	} else {
		log.Println("HTTP-сервер успешно остановлен.")
	}

	log.Println("Закрываем пул подключений к PostgreSQL...")
	pool.Close()
	log.Println("Сервис полностью завершил работу.")
}