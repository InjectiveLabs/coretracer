package coretracer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestTraceError_StackTraceExcludesInternalFrames verifies that TraceError
// excludes internal coretracer frames from the stack trace
func TestTraceError_StackTraceExcludesInternalFrames(t *testing.T) {
	// Setup in-memory exporter to capture spans
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
	)
	otel.SetTracerProvider(tp)

	// Initialize tracer with test config
	cfg := &Config{
		EnvName:               "test",
		StuckFunctionWatchdog: false,
		Logger:                slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}

	tracer := newOtelTracer(cfg)
	defer tracer.Close()

	// Create a parent span first so TraceError has a span to attach to
	testTracer := otel.GetTracerProvider().Tracer("test")
	ctx, parentSpan := testTracer.Start(context.Background(), "parent-span")
	testError := errors.New("test error from user code")

	// Simulate user code calling TraceError
	userFunction1(tracer, ctx, testError)

	parentSpan.End()

	// Flush the tracer provider to export spans
	require.NoError(t, tp.ForceFlush(context.Background()))

	// Get the recorded spans
	spans := exporter.GetSpans()
	require.NotEmpty(t, spans, "Expected at least one span to be recorded")

	// Find the span with the error event
	var errorEvent sdktrace.Event
	var foundEvent bool
	for _, span := range spans {
		for _, event := range span.Events {
			if event.Name == "exception" {
				errorEvent = event
				foundEvent = true
				break
			}
		}
		if foundEvent {
			break
		}
	}

	require.True(t, foundEvent, "Expected to find an exception event")

	// Extract the stack trace attribute
	var stackTrace string
	for _, attr := range errorEvent.Attributes {
		if attr.Key == "exception.stacktrace" {
			stackTrace = attr.Value.AsString()
			break
		}
	}

	require.NotEmpty(t, stackTrace, "Expected stack trace to be present")

	// The stack trace should NOT contain internal coretracer functions
	require.NotContains(t, stackTrace, "github.com/InjectiveLabs/coretracer.(*otelTracer).TraceError",
		"Stack trace should not contain internal otelTracer.TraceError")
	require.NotContains(t, stackTrace, "github.com/InjectiveLabs/coretracer.TraceError",
		"Stack trace should not contain internal coretracer.TraceError wrapper")

	// The stack trace SHOULD contain user functions
	require.Contains(t, stackTrace, "github.com/InjectiveLabs/coretracer.userFunction1",
		"Stack trace should contain userFunction1")
	require.Contains(t, stackTrace, "github.com/InjectiveLabs/coretracer.userFunction2",
		"Stack trace should contain userFunction2")
	require.Contains(t, stackTrace, "github.com/InjectiveLabs/coretracer.userFunction3",
		"Stack trace should contain userFunction3 (the actual caller)")

	// Verify the first frame is from user code
	lines := strings.Split(stackTrace, "\n")
	firstFunctionLine := lines[0]
	require.Contains(t, firstFunctionLine, "github.com/InjectiveLabs/coretracer.userFunction3",
		"First stack frame should be from user code (userFunction3)")
}

// TestTraceError_StackTraceFormat verifies the format matches Go's standard stack trace
func TestTraceError_StackTraceFormat(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
	)
	otel.SetTracerProvider(tp)

	cfg := &Config{
		EnvName:               "test",
		StuckFunctionWatchdog: false,
		Logger:                slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	tracer := newOtelTracer(cfg)
	defer tracer.Close()

	// Create a parent span
	testTracer := otel.GetTracerProvider().Tracer("test")
	ctx, parentSpan := testTracer.Start(context.Background(), "parent-span")
	testError := errors.New("test error")

	userFunction3(tracer, ctx, testError)

	parentSpan.End()

	// Flush the tracer provider
	require.NoError(t, tp.ForceFlush(context.Background()))

	spans := exporter.GetSpans()
	require.NotEmpty(t, spans)

	var stackTrace string
	for _, span := range spans {
		for _, event := range span.Events {
			if event.Name == "exception" {
				for _, attr := range event.Attributes {
					if attr.Key == "exception.stacktrace" {
						stackTrace = attr.Value.AsString()
						break
					}
				}
			}
		}
	}

	require.NotEmpty(t, stackTrace)

	// Verify format: function\n\tfile:line\nfunction\n\tfile:line...
	lines := strings.Split(stackTrace, "\n")
	require.Greater(t, len(lines), 0, "Stack trace should have multiple lines")

	// Check alternating pattern: function, then file:line
	for i := 0; i < len(lines); i += 2 {
		functionLine := lines[i]

		// Function line should contain package path
		require.NotEmpty(t, functionLine, "Function line should not be empty")

		// Next line should be file path with line number
		if i+1 < len(lines) {
			fileLine := lines[i+1]
			require.True(t, strings.HasPrefix(fileLine, "\t"),
				fmt.Sprintf("File line should start with tab: %q", fileLine))
			require.Contains(t, fileLine, ":",
				fmt.Sprintf("File line should contain colon: %q", fileLine))
		}
	}
}

// TestTraceError_WithTags verifies tags are properly included
func TestTraceError_WithTags(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
	)
	otel.SetTracerProvider(tp)

	cfg := &Config{
		EnvName:               "test",
		StuckFunctionWatchdog: false,
		Logger:                slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	tracer := newOtelTracer(cfg)
	defer tracer.Close()

	// Create a parent span
	testTracer := otel.GetTracerProvider().Tracer("test")
	ctx, parentSpan := testTracer.Start(context.Background(), "parent-span")
	testError := errors.New("test error")
	testTags := NewTags().With("test.key", "test.value")

	tracer.TraceError(ctx, testError, testTags)

	parentSpan.End()

	// Flush the tracer provider
	require.NoError(t, tp.ForceFlush(context.Background()))

	spans := exporter.GetSpans()
	require.NotEmpty(t, spans)

	// Find the error event attributes
	var foundTag bool
	for _, span := range spans {
		for _, event := range span.Events {
			if event.Name == "exception" {
				for _, attr := range event.Attributes {
					if attr.Key == "test.key" && attr.Value.AsString() == "test.value" {
						foundTag = true
						break
					}
				}
			}
		}
	}

	require.True(t, foundTag, "Expected to find test tag in error event")
}

// Helper functions to simulate user code with nested calls
func userFunction1(tracer Tracer, ctx context.Context, err error) {
	userFunction2(tracer, ctx, err)
}

func userFunction2(tracer Tracer, ctx context.Context, err error) {
	userFunction3(tracer, ctx, err)
}

func userFunction3(tracer Tracer, ctx context.Context, err error) {
	// This is where TraceError is actually called
	tracer.TraceError(ctx, err)
}
