package coretracer

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	otel "go.opentelemetry.io/otel"
	otelattribute "go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	oteltracer "go.opentelemetry.io/otel/trace"

	"github.com/InjectiveLabs/coretracer/stackcache"
)

const defaultStackSearchOffset = 1

var _ Tracer = (*otelTracer)(nil)

// captureErrorStackTrace captures the stack trace for error reporting,
// skipping the specified number of frames
func captureErrorStackTrace(skip int) string {
	const maxDepth = 32
	pcs := make([]uintptr, maxDepth)
	n := runtime.Callers(skip, pcs)

	if n == 0 {
		return ""
	}

	frames := runtime.CallersFrames(pcs[:n])
	var result strings.Builder

	for {
		frame, more := frames.Next()
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(frame.Function)
		result.WriteString("\n\t")
		result.WriteString(frame.File)
		result.WriteString(":")
		result.WriteString(fmt.Sprintf("%d", frame.Line))

		if !more {
			break
		}
	}

	return result.String()
}

func newOtelTracer(cfg *Config) Tracer {
	cfg = validateConfig(cfg)

	t := &otelTracer{
		config:          cfg,
		logger:          cfg.Logger,
		callStackOffset: 0,
		tracer:          otel.GetTracerProvider().Tracer("coretracer"),
	}

	t.stackCache = stackcache.New(
		defaultStackSearchOffset,
		t.callStackOffset,
		"github.com/InjectiveLabs/coretracer",
	)

	return t
}

type otelTracer struct {
	config          *Config
	callStackOffset int
	tracer          oteltracer.Tracer
	logger          BasicLogger
	stackCache      stackcache.StackCache
}

// Close implements Tracer.
func (t *otelTracer) Close() {
	t.tracer = nil
}

// Trace implements Tracer.
func (t *otelTracer) Trace(ctx *context.Context, tags ...Tags) SpanEnderFn {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Error("coretracer: Trace() panicked - this is a bug", "panic", r)
			t.logger.Error("coretracer: stack trace", "stack", string(debug.Stack()))
		}
	}()

	frame := t.stackCache.GetCaller()
	funcName := stackcache.FuncName(frame.Function)

	return t.traceStart(ctx, funcName, false, tags)
}

// TraceError implements Tracer.
func (t *otelTracer) TraceError(ctx context.Context, err error, tags ...Tags) {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Error("coretracer: TraceError() panicked - this is a bug", "panic", r)
			t.logger.Error("coretracer: stack trace", "stack", string(debug.Stack()))
		}
	}()

	if err == nil {
		t.logger.Debug("coretracer: TraceError() called with nil error")
		return
	} else if ctx == nil {
		ctx = context.Background()
	}

	var isNewSpan bool
	span := oteltracer.SpanFromContext(ctx)

	if span == nil {
		// Create a new virtual span if no span exists
		frame := t.stackCache.GetCaller()
		funcName := stackcache.FuncName(frame.Function)

		t.logger.Debug("coretracer: TracelessError starts from", "function", funcName)

		ctxPtr := &ctx
		_ = t.traceStart(ctxPtr, funcName, true, tags)
		span = oteltracer.SpanFromContext(*ctxPtr)
		isNewSpan = true
	} else if !span.IsRecording() {
		return
	}

	span.SetStatus(otelcodes.Error, err.Error())
	errorOpts := []oteltracer.EventOption{
		// do not include stack trace provided by OpenTelemetry SDK,
		// we'll set our own.
	}

	// isNewSpan already includes these tags
	if len(tags) > 0 && !isNewSpan {
		// Merge tags into attributes
		allTags := NewTags().Union(tags...)
		attributes := make([]otelattribute.KeyValue, 0, len(tags))
		allTags.Range(func(k string, v any) bool {
			attributes = append(attributes, anyToOtalAttribute(k, v))
			return true
		})
		errorOpts = append(errorOpts, oteltracer.WithAttributes(attributes...))
	}

	// Capture and trim stack trace to exclude internal frames
	// Skip frames: runtime.Callers(0), captureErrorStackTrace(1), TraceError-otelTracer(2)
	stackTrace := captureErrorStackTrace(3)
	if stackTrace != "" {
		errorOpts = append(errorOpts, oteltracer.WithAttributes(
			otelattribute.String("exception.stacktrace", stackTrace),
		))
	}

	span.RecordError(err, errorOpts...)

	span.End()
}

// TraceWithName implements Tracer.
func (t *otelTracer) TraceWithName(ctx *context.Context, name string, tags ...Tags) SpanEnderFn {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Error("coretracer: TraceWithName() panicked - this is a bug", "panic", r)
			t.logger.Error("coretracer: stack trace", "stack", string(debug.Stack()))
		}
	}()

	return t.traceStart(ctx, name, false, tags)
}

// Traceless implements Tracer.
func (t *otelTracer) Traceless(ctx *context.Context, tags ...Tags) SpanEnderFn {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Error("coretracer: Traceless() panicked - this is a bug", "panic", r)
			t.logger.Error("coretracer: stack trace", "stack", string(debug.Stack()))
		}
	}()

	frame := t.stackCache.GetCaller()
	funcName := stackcache.FuncName(frame.Function)

	t.logger.Debug("coretracer: Traceless() starts from", "function", funcName)

	return t.traceStart(ctx, funcName, true, tags)
}

// TracelessWithName implements Tracer.
func (t *otelTracer) TracelessWithName(ctx *context.Context, name string, tags ...Tags) SpanEnderFn {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Error("coretracer: TracelessWithName() panicked - this is a bug", "panic", r)
			t.logger.Error("coretracer: stack trace", "stack", string(debug.Stack()))
		}
	}()

	return t.traceStart(ctx, name, true, tags)
}

func (t *otelTracer) traceStart(ctx *context.Context, funcName string, virtualTrace bool, tags []Tags) SpanEnderFn {
	if ctx == nil {
		emptyCtx := context.Background()
		ctx = &emptyCtx

		virtualTrace = true
	}

	allTags := NewTags().Union(tags...)
	attributes := make([]otelattribute.KeyValue, 0, len(tags))
	allTags.Range(func(k string, v any) bool {
		attributes = append(attributes, anyToOtalAttribute(k, v))
		return true
	})

	var parentSpans []oteltracer.Span
	parentSpansEndFn := func(spansToEnd []oteltracer.Span) {}

	if virtualTrace {
		now := time.Now().UTC()
		frames := t.stackCache.GetStackFrames()

		*ctx, parentSpans = t.callStackFramesToSpans(now, frames, attributes)

		parentSpansEndFn = func(spansToEnd []oteltracer.Span) {
			if len(spansToEnd) == 0 {
				return
			}

			// iterate in reverse order to end the spans in the correct order
			for i := 0; i < len(spansToEnd); i++ {
				spansToEnd[i].End(oteltracer.WithTimestamp(now))
			}
		}
	}

	// this the final span
	modifiedContext, span := t.tracer.Start(
		*ctx,
		funcName,
		oteltracer.WithAttributes(attributes...),
	)

	doneC := make(chan struct{}, 1)

	if t.config.StuckFunctionWatchdog {
		go func(name string, start time.Time) {
			timeout := time.NewTimer(t.config.StuckFunctionTimeout)
			defer timeout.Stop()

			select {
			case <-doneC:
				return
			case <-timeout.C:
				if !span.IsRecording() {
					return
				}

				err := fmt.Errorf("detected stuck function: %s stuck for %v", name, time.Since(start))
				span.RecordError(
					err, oteltracer.WithStackTrace(true),
				)

				span.SetAttributes(otelattribute.String("exception.type", "stuck"))
				span.SetStatus(otelcodes.Error, "stuck")
			}
		}(funcName, time.Now().UTC())
	}

	// set the modified context in-place
	*ctx = modifiedContext

	return func() {
		close(doneC)

		if span.IsRecording() {
			span.SetStatus(otelcodes.Ok, "")
			span.End()
		}

		parentSpansEndFn(parentSpans)
	}
}

func (t *otelTracer) callStackFramesToSpans(
	timestamp time.Time,
	frames []runtime.Frame,
	attributes []otelattribute.KeyValue,
) (context.Context, []oteltracer.Span) {
	if len(frames) <= 2 {
		return context.Background(), nil
	}

	spans := make([]oteltracer.Span, 0, len(frames)-2)

	var newSpan oteltracer.Span
	ctx := context.Background()

	for i := len(frames) - 2; i > 0; i-- {
		if frames[i].Function == "runtime.main" ||
			frames[i].Function == "main.main" {
			continue
		}

		opts := []oteltracer.SpanStartOption{
			oteltracer.WithAttributes(attributes...),
			oteltracer.WithTimestamp(timestamp),
		}

		if newSpan == nil {
			opts = append(opts, oteltracer.WithNewRoot())
		}

		ctx, newSpan = t.tracer.Start(
			ctx,
			stackcache.FuncName(frames[i].Function),
			opts...,
		)

		spans = append(spans, newSpan)
	}

	return ctx, spans
}

// WithTags implements Tracer.
func (t *otelTracer) WithTags(ctx context.Context, tags ...Tags) {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Error("coretracer: WithTags() panicked - this is a bug", "panic", r)
			t.logger.Error("coretracer: stack trace", "stack", string(debug.Stack()))
		}
	}()

	span := oteltracer.SpanFromContext(ctx)
	if span == nil {
		t.logger.Warn("coretracer: no span found in context - WithTags() with invalid context")
		return
	}

	allTags := NewTags().Union(tags...)
	attributes := make([]otelattribute.KeyValue, 0, len(tags))
	allTags.Range(func(k string, v any) bool {
		attributes = append(attributes, anyToOtalAttribute(k, v))
		return true
	})

	span.SetAttributes(attributes...)
}

// SetCallStackOffset implements Tracer.
func (t *otelTracer) SetCallStackOffset(offset int) {
	if offset < 0 {
		offset = 0
	}

	t.callStackOffset = offset
}

func anyToOtalAttribute(k string, v any) otelattribute.KeyValue {
	if v == nil {
		return otelattribute.String(k, "")
	}

	switch v := v.(type) {
	case string:
		return otelattribute.String(k, v)
	case int:
		return otelattribute.Int(k, v)
	case int64:
		return otelattribute.Int64(k, v)
	case float64:
		return otelattribute.Float64(k, v)
	case bool:
		return otelattribute.Bool(k, v)
	case []string:
		return otelattribute.StringSlice(k, v)
	case []int:
		return otelattribute.IntSlice(k, v)
	case []int64:
		return otelattribute.Int64Slice(k, v)
	case []float64:
		return otelattribute.Float64Slice(k, v)
	case []bool:
		return otelattribute.BoolSlice(k, v)
	case *string:
		return otelattribute.String(k, *v)
	case *int:
		return otelattribute.Int(k, *v)
	case *int64:
		return otelattribute.Int64(k, *v)
	case *float64:
		return otelattribute.Float64(k, *v)
	case *bool:
		return otelattribute.Bool(k, *v)
	case *[]string:
		return otelattribute.StringSlice(k, *v)
	case *[]int:
		return otelattribute.IntSlice(k, *v)
	case *[]int64:
		return otelattribute.Int64Slice(k, *v)
	case *[]float64:
		return otelattribute.Float64Slice(k, *v)
	case *[]bool:
		return otelattribute.BoolSlice(k, *v)
	}

	if stringer, ok := v.(fmt.Stringer); ok {
		return otelattribute.Stringer(k, stringer)
	}

	return otelattribute.String(k, fmt.Sprintf("%v", v))
}
