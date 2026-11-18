package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/Hil-sys/TripGo-autumn/internal/config"
	domain "github.com/Hil-sys/TripGo-autumn/internal/domain/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/generated"
	httpserver "github.com/Hil-sys/TripGo-autumn/internal/http"
	"github.com/Hil-sys/TripGo-autumn/internal/http/domainerror"
	"github.com/Hil-sys/TripGo-autumn/internal/http/middleware"
	systemhandler "github.com/Hil-sys/TripGo-autumn/internal/http/system"
	createhandler "github.com/Hil-sys/TripGo-autumn/internal/http/trip/create"
	finishhandler "github.com/Hil-sys/TripGo-autumn/internal/http/trip/finish"
	gethandler "github.com/Hil-sys/TripGo-autumn/internal/http/trip/get"
	"github.com/Hil-sys/TripGo-autumn/internal/infra/postgres"
	"github.com/Hil-sys/TripGo-autumn/internal/infra/postgres/transaction"
	"github.com/Hil-sys/TripGo-autumn/internal/infra/telemetry"
	triprepo "github.com/Hil-sys/TripGo-autumn/internal/repository/trip"
	createuc "github.com/Hil-sys/TripGo-autumn/internal/usecase/trip/create"
	finishuc "github.com/Hil-sys/TripGo-autumn/internal/usecase/trip/finish"
	getuc "github.com/Hil-sys/TripGo-autumn/internal/usecase/trip/get"
)

const serviceName = "trip-service"

func main() {
	if err := run(); err != nil {
		slog.Error("application exited with error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	logger := telemetry.NewLogger(cfg.LogLevel, serviceName)
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 1. Инициализация OpenTelemetry Tracing
	traceShutdown, err := telemetry.InitTracer(ctx, cfg.OtelEndpoint, serviceName)
	if err != nil {
		logger.WarnContext(ctx, "failed to initialize tracer", slog.Any("error", err))
	}
	defer func() {
		shutdownCtx, sCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer sCancel()
		if traceShutdown != nil {
			_ = traceShutdown(shutdownCtx)
		}
	}()

	// 2. Инициализация OpenTelemetry Metrics
	metrics, meterShutdown, err := telemetry.InitMeter(ctx, cfg.OtelEndpoint, serviceName)
	if err != nil {
		logger.WarnContext(ctx, "failed to initialize meter", slog.Any("error", err))
	}
	defer func() {
		shutdownCtx, sCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer sCancel()
		if meterShutdown != nil {
			_ = meterShutdown(shutdownCtx)
		}
	}()

	// 3. Инфраструктура базы данных и транзакций
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	txManager := transaction.NewTxManager(pool)

	// 4. Репозитории и сценарии бизнес-логики (Use Cases)
	repo := triprepo.NewRepository(txManager, cfg.Database.QueryTimeout)

	createUseCase := createuc.New(repo, txManager, logger)
	getUseCase := getuc.New(repo)
	finishUseCase := finishuc.New(repo, txManager, logger)

	// 5. HTTP-обработчики и адаптер ServerInterface
	serverAdapter := httpserver.NewServer(
		createhandler.NewHandler(createUseCase, metrics, logger),
		gethandler.NewHandler(getUseCase, logger),
		finishhandler.NewHandler(finishUseCase, metrics, logger),
		systemhandler.NewHandler(repo),
	)

	// 6. Роутер и Middleware цепочка
	r := chi.NewRouter()
	r.Use(middleware.Recovery(logger))
	r.Use(otelhttp.NewMiddleware(serviceName))
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Metrics(metrics))

	api.HandlerWithOptions(serverAdapter, api.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			field := "tripId"
			var invalidParamErr *api.InvalidParamFormatError
			if errors.As(err, &invalidParamErr) {
				field = invalidParamErr.ParamName
			}
			status, prob := domainerror.Problem(
				domain.ValidationError{Field: field, Reason: "must be a valid UUID"},
				r.URL.Path,
			)
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(prob)
		},
	})

	// 7. Конфигурация HTTP-сервера с таймаутами
	httpServer := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           r,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.InfoContext(ctx, "starting http server", slog.String("addr", cfg.HTTP.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("http server crashed: %w", err)
	case <-ctx.Done():
		logger.Info("shutting down service gracefully")
	}

	// 8. Graceful shutdown по SHUTDOWN_TIMEOUT
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", slog.Any("error", err))
	}

	logger.Info("service exited cleanly")
	return nil
}
