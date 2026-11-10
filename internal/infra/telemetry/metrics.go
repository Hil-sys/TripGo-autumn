package telemetry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	metricApi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type Metrics struct {
	HTTPDuration       metricApi.Float64Histogram
	HTTPActiveRequests metricApi.Int64UpDownCounter
	DBDuration         metricApi.Float64Histogram
	TripsCreated       metricApi.Int64Counter
	TripsCompleted     metricApi.Int64Counter
}

func InitMeter(ctx context.Context, endpoint string, serviceName string) (*Metrics, func(context.Context) error, error) {
	if endpoint == "" {
		return &Metrics{}, func(context.Context) error { return nil }, nil
	}

	hostPort := strings.TrimPrefix(endpoint, "http://")
	hostPort = strings.TrimPrefix(hostPort, "https://")
	hostPort = strings.TrimRight(hostPort, "/")

	exporter, err := otlpmetrichttp.New(
		ctx,
		otlpmetrichttp.WithEndpoint(hostPort),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create otlp metric exporter: %w", err)
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create metric resource: %w", err)
	}

	// Сбрасываем метрики в коллектор каждые 3 секунды для оперативного отображения
	reader := metric.NewPeriodicReader(
		exporter,
		metric.WithInterval(3*time.Second),
	)

	mp := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	meter := mp.Meter(serviceName)

	// 1. http.server.request.duration (histogram, секунды)
	httpDuration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metricApi.WithDescription("Measures the duration of inbound HTTP requests"),
		metricApi.WithUnit("s"),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create http.server.request.duration: %w", err)
	}

	// 2. http.server.active_requests (up-down counter, штуки)
	httpActiveRequests, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metricApi.WithDescription("Number of active HTTP server requests"),
		metricApi.WithUnit("{request}"),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create http.server.active_requests: %w", err)
	}

	// 3. db.client.operation.duration (histogram, секунды)
	dbDuration, err := meter.Float64Histogram(
		"db.client.operation.duration",
		metricApi.WithDescription("Duration of database client operations"),
		metricApi.WithUnit("s"),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create db.client.operation.duration: %w", err)
	}

	// 4. tripgo.trips.created (counter, штуки)
	tripsCreated, err := meter.Int64Counter(
		"tripgo.trips.created",
		metricApi.WithDescription("Total number of trips created"),
		metricApi.WithUnit("{trip}"),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create tripgo.trips.created: %w", err)
	}

	// 5. tripgo.trips.completed (counter, штуки)
	tripsCompleted, err := meter.Int64Counter(
		"tripgo.trips.completed",
		metricApi.WithDescription("Total number of trips completed"),
		metricApi.WithUnit("{trip}"),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create tripgo.trips.completed: %w", err)
	}

	metrics := &Metrics{
		HTTPDuration:       httpDuration,
		HTTPActiveRequests: httpActiveRequests,
		DBDuration:         dbDuration,
		TripsCreated:       tripsCreated,
		TripsCompleted:     tripsCompleted,
	}

	return metrics, mp.Shutdown, nil
}
