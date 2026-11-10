package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/Hil-sys/TripGo-autumn/internal/config"
	"github.com/Hil-sys/TripGo-autumn/internal/generated"
	httpserver "github.com/Hil-sys/TripGo-autumn/internal/http"
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
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger := telemetry.NewLogger(cfg.LogLevel, serviceName)
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()


	traceShutdown, err := telemetry.InitTracer(ctx, cfg.OtelEndpoint, serviceName)
	if err != nil {
		logger.WarnContext(ctx, "failed to initialize tracer, continuing without tracing", "error", err)
	}
	defer func() {
		shutdownCtx, sCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer sCancel()
		if traceShutdown != nil {
			_ = traceShutdown(shutdownCtx)
		}
	}()

	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		logger.ErrorContext(ctx, "failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	txManager := transaction.NewTxManager(pool)

	repo := triprepo.NewRepository(txManager, cfg.Database.QueryTimeout)

	createUseCase := createuc.New(repo, txManager)
	getUseCase := getuc.New(repo)
	finishUseCase := finishuc.New(repo, txManager)

	serverAdapter := httpserver.NewServer(
		createhandler.NewHandler(createUseCase),
		gethandler.NewHandler(getUseCase),
		finishhandler.NewHandler(finishUseCase),
		systemhandler.NewHandler(repo),
	)

	r := chi.NewRouter()
	r.Use(otelhttp.NewMiddleware(serviceName))
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Recovery(logger))

	api.HandlerFromMux(serverAdapter, r)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.InfoContext(ctx, "starting http server", slog.String("addr", cfg.HTTPAddr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.ErrorContext(ctx, "http server crashed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down service gracefully")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", slog.Any("error", err))
	}

	logger.Info("service exited cleanly")
}
