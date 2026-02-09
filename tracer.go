package coretracer

import (
	"context"
	"log/slog"
	"sync"

	otel "go.opentelemetry.io/otel"
)

var (
	tracer             Tracer
	tracerMux          = new(sync.RWMutex)
	exporterShutdownFn ExporterShutdownFn

	config *Config
)

type (
	SpanEnderFn        func()
	ExporterShutdownFn func(ctx context.Context) error
)

type Tracer interface {
	Trace(ctx *context.Context, tags ...Tags) SpanEnderFn
	TraceWithName(ctx *context.Context, name string, tags ...Tags) SpanEnderFn
	TraceError(ctx context.Context, err error, tags ...Tags)
	Traceless(ctx *context.Context, tags ...Tags) SpanEnderFn
	TracelessWithName(ctx *context.Context, name string, tags ...Tags) SpanEnderFn

	WithTags(ctx context.Context, tags ...Tags)
	SetCallStackOffset(offset int)
	Close()
}

func Enable(cfg *Config, exporterInitFn func(cfg *Config) ExporterShutdownFn) {
	tracerMux.Lock()
	defer tracerMux.Unlock()

	exporterShutdownFn = exporterInitFn(cfg)

	if exporterShutdownFn == nil {
		slog.Warn("coretracer: failed to enable tracer")
		return
	}

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		cfg.Logger.Warn("coretracer: otel tracer error", "error", err)
	}))

	tracer = newOtelTracer(cfg)
	config = cfg
}

func Disable() {
	tracerMux.Lock()
	defer tracerMux.Unlock()
	if tracer != nil {
		tracer.Close()

		if err := exporterShutdownFn(context.Background()); err != nil {
			slog.Error("coretracer: failed to shutdown exporter", "error", err)
		}
	}

	tracer = nil
}

func DefaultTracer() Tracer {
	tracerMux.RLock()
	defer tracerMux.RUnlock()

	return tracer
}

func Trace(ctx *context.Context, tags ...Tags) SpanEnderFn {
	tracerMux.RLock()
	defer tracerMux.RUnlock()
	if tracer == nil {
		return func() {}
	}

	return tracer.Trace(ctx, tags...)
}

func TraceWithName(ctx *context.Context, name string, tags ...Tags) SpanEnderFn {
	tracerMux.RLock()
	defer tracerMux.RUnlock()
	if tracer == nil {
		return func() {}
	}

	return tracer.TraceWithName(ctx, name, tags...)
}

func TraceError(ctx context.Context, err error, tags ...Tags) {
	tracerMux.RLock()
	defer tracerMux.RUnlock()
	if tracer == nil {
		return
	}

	tracer.TraceError(ctx, err, tags...)
}

func Traceless(ctx *context.Context, tags ...Tags) SpanEnderFn {
	tracerMux.RLock()
	defer tracerMux.RUnlock()
	if tracer == nil {
		return func() {}
	}

	return tracer.Traceless(ctx, tags...)
}

func TracelessWithName(ctx *context.Context, name string, tags ...Tags) SpanEnderFn {
	tracerMux.RLock()
	defer tracerMux.RUnlock()
	if tracer == nil {
		return func() {}
	}

	return tracer.TracelessWithName(ctx, name, tags...)
}

func WithTags(ctx context.Context, tags ...Tags) {
	tracerMux.RLock()
	defer tracerMux.RUnlock()
	if tracer == nil {
		return
	}

	tracer.WithTags(ctx, tags...)
}

func SetCallStackOffset(offset int) {
	tracerMux.RLock()
	defer tracerMux.RUnlock()
	if tracer == nil {
		return
	}

	tracer.SetCallStackOffset(offset)
}

func Close() {
	tracerMux.Lock()
	defer tracerMux.Unlock()
	if tracer == nil {
		return
	}

	tracer.Close()
	tracer = nil

	if err := exporterShutdownFn(context.Background()); err != nil {
		slog.Error("coretracer: failed to shutdown exporter", "error", err)
	}
}
