package otel

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/InjectiveLabs/coretracer"
)

func InitExporter(cfg *coretracer.Config) coretracer.ExporterShutdownFn {
	var secureOption otlptracegrpc.Option

	if cfg.CollectorSecureSSL {
		secureOption = otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))
	} else {
		secureOption = otlptracegrpc.WithInsecure()
	}

	clientOpts := []otlptracegrpc.Option{
		secureOption,
		otlptracegrpc.WithEndpoint(cfg.CollectorDSN),
	}

	if len(cfg.CollectorHeaders) > 0 {
		clientOpts = append(clientOpts, otlptracegrpc.WithHeaders(cfg.CollectorHeaders))
	}

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(clientOpts...),
	)
	if err != nil {
		slog.Warn("coretracer: otel exporter: failed to create exporter", "error", err)
		return emptyShutdownFn()
	}

	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", cfg.ServiceName),
			attribute.String("service.version", cfg.ServiceVersion),
			attribute.String("deployment.environment", cfg.EnvName),
			attribute.String("deployment.cluster_id", cfg.ClusterID),
		),
	)
	if err != nil {
		slog.Warn("coretracer: otel exporter: could not set resources", "error", err)
		return emptyShutdownFn()
	}

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resources),
	)

	otel.SetTracerProvider(traceProvider)

	return func(ctx context.Context) error {
		if err := traceProvider.ForceFlush(ctx); err != nil {
			slog.Warn("coretracer: otel exporter: failed to force flush traces", "error", err)
		}

		return exporter.Shutdown(ctx)
	}
}

func emptyShutdownFn() func(ctx context.Context) error {
	return func(ctx context.Context) error { return nil }
}
