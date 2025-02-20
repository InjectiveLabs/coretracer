package otel

import (
	"context"
	"log"

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

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			secureOption,
			otlptracegrpc.WithEndpoint(cfg.CollectorDSN),
		),
	)
	if err != nil {
		log.Printf("[WARN] Otel Exporter: Failed to create exporter: %v", err)
		return emptyShutdownFn()
	}

	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", cfg.ServiceName),
			attribute.String("service.version", cfg.ServiceVersion),
			attribute.String("deployment.environment", cfg.EnvName),
			attribute.String("deployment.chain_id", cfg.ChainID),
		),
	)
	if err != nil {
		log.Printf("[WARN] Otel Exporter: Could not set resources: %v", err)
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
			log.Printf("[WARN] Otel Exporter: Failed to force flush traces: %v", err)
		}

		return exporter.Shutdown(ctx)
	}
}

func emptyShutdownFn() func(ctx context.Context) error {
	return func(ctx context.Context) error { return nil }
}
