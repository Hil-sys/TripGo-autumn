package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

type ContextHandler struct {
	slog.Handler
}

func (h *ContextHandler) Handle(ctx context.Context, record slog.Record) error {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		record.AddAttrs(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, record)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithGroup(name)}
}

func NewLogger(levelStr, serviceName string) *slog.Logger {
	var level slog.Level
	switch levelStr {
	case "DEBUG", "debug":
		level = slog.LevelDebug
	case "WARN", "warn":
		level = slog.LevelWarn
	case "ERROR", "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, opts)
	ctxHandler := &ContextHandler{Handler: jsonHandler}

	return slog.New(ctxHandler).With(slog.String("service", serviceName))
}
