package telemetry

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func samplerFromEnv() sdktrace.Sampler {
	name := strings.ToLower(strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER")))

	ratio := func() float64 {
		arg := strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER_ARG"))
		if v, err := strconv.ParseFloat(arg, 64); err == nil && v >= 0 && v <= 1 {
			return v
		}
		return 1.0
	}

	switch name {
	case "always_off":
		return sdktrace.NeverSample()
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(ratio())
	case "parentbased_always_on":
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample())
	case "parentbased_traceidratio":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio()))
	default: // "always_on" или пусто
		return sdktrace.AlwaysSample()
	}
}

// InitTracer инициализирует OpenTelemetry TracerProvider через OTLP HTTP-экспортер.
func InitTracer(ctx context.Context, endpoint, serviceName string) (func(context.Context) error, error) {
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	// Отрезаем схемы http:// и https://, так как WithEndpoint ожидает чистый host:port
	hostPort := strings.TrimPrefix(endpoint, "http://")
	hostPort = strings.TrimPrefix(hostPort, "https://")
	hostPort = strings.TrimRight(hostPort, "/")

	exporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpoint(hostPort),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp http trace exporter: %w", err)
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(samplerFromEnv()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}
