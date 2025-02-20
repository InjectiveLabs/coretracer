package coretracer

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	otel "go.opentelemetry.io/otel"
	otelattribute "go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	oteltracer "go.opentelemetry.io/otel/trace"

	"github.com/InjectiveLabs/coretracer/stackcache"
)

const defaultStackSearchOffset = 1

var _ Tracer = (*otelTracer)(nil)

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
			t.logger.Printf("[PANIC] Trace() panicked - this is a bug: %v", r)
			t.logger.Println(string(debug.Stack()))
		}
	}()

	frame := t.stackCache.GetCaller()
	funcName := stackcache.FuncName(frame.Function)

	return t.traceStart(ctx, funcName, false, tags)
}

// TraceError implements Tracer.
func (t *otelTracer) TraceError(ctx context.Context, err error) {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Printf("[PANIC] TraceError() panicked - this is a bug: %v", r)
			t.logger.Println(string(debug.Stack()))
		}
	}()

	if ctx == nil {
		ctx = context.Background()
	}

	span := oteltracer.SpanFromContext(ctx)
	if span == nil {
		t.logger.Println("[WARN] no span found in context - TraceError() with invalid context")
		return
	}

	if !span.IsRecording() {
		return
	}

	if err != nil {
		span.SetStatus(otelcodes.Error, err.Error())
	} else {
		// prevent panic from a programming error when a wrong err value is passed
		span.SetStatus(otelcodes.Error, "")
	}

	span.RecordError(
		err, oteltracer.WithStackTrace(true),
	)

	span.End()
}

// TraceWithName implements Tracer.
func (t *otelTracer) TraceWithName(ctx *context.Context, name string, tags ...Tags) SpanEnderFn {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Printf("[PANIC] TraceWithName() panicked - this is a bug: %v", r)
			t.logger.Println(string(debug.Stack()))
		}
	}()

	return t.traceStart(ctx, name, false, tags)
}

// Traceless implements Tracer.
func (t *otelTracer) Traceless(ctx *context.Context, tags ...Tags) SpanEnderFn {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Printf("[PANIC] Traceless() panicked - this is a bug: %v", r)
			t.logger.Println(string(debug.Stack()))
		}
	}()

	frame := t.stackCache.GetCaller()
	funcName := stackcache.FuncName(frame.Function)

	t.logger.Println("[DEBUG] Traceless() starts from", funcName)

	return t.traceStart(ctx, funcName, true, tags)
}

// TracelessWithName implements Tracer.
func (t *otelTracer) TracelessWithName(ctx *context.Context, name string, tags ...Tags) SpanEnderFn {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Printf("[PANIC] TracelessWithName() panicked - this is a bug: %v", r)
			t.logger.Println(string(debug.Stack()))
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
			t.logger.Printf("[PANIC] WithTags() panicked - this is a bug: %v", r)
			t.logger.Println(string(debug.Stack()))
		}
	}()

	span := oteltracer.SpanFromContext(ctx)
	if span == nil {
		t.logger.Println("[WARN] no span found in context - WithTags() with invalid context")
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

type (
	stacktracerSpanDone  struct{}
	stacktracerSpanError struct{}
	stacktracerSpanStuck struct{}
)

func tombstoneSpanCtxDone(ctx context.Context) context.Context {
	return context.WithValue(ctx, stacktracerSpanDone{}, true)
}

func tombstoneSpanCtxError(ctx context.Context) context.Context {
	return context.WithValue(ctx, stacktracerSpanError{}, true)
}

func tombstoneSpanCtxStuck(ctx context.Context) context.Context {
	return context.WithValue(ctx, stacktracerSpanStuck{}, true)
}

func isTombstoneSpanCtxDone(ctx context.Context) bool {
	return ctx.Value(stacktracerSpanDone{}) != nil
}

func isTombstoneSpanCtxError(ctx context.Context) bool {
	return ctx.Value(stacktracerSpanError{}) != nil
}

func isTombstoneSpanCtxStuck(ctx context.Context) bool {
	return ctx.Value(stacktracerSpanStuck{}) != nil
}

func (t *otelTracer) checkAndSetTombstoneSpanCtxDone(ctx context.Context) (context.Context, bool) {
	if isTombstoneSpanCtxError(ctx) || isTombstoneSpanCtxStuck(ctx) {
		// already tombstoned
		return ctx, false
	}

	return tombstoneSpanCtxDone(ctx), true
}

func (t *otelTracer) checkAndSetTombstoneSpanCtxError(ctx context.Context) (context.Context, bool) {
	if isTombstoneSpanCtxStuck(ctx) {
		// already tombstoned as stuck
		return ctx, false
	} else if isTombstoneSpanCtxDone(ctx) {
		// already tombstoned as done - this is a bug
		t.logger.Println("[WARN] span context is already tombstoned as done - this is a bug")
		return ctx, false
	} else if isTombstoneSpanCtxError(ctx) {
		// already tombstoned as error - this is a bug
		t.logger.Println("[WARN] span context is already tombstoned as error - skipping double tombstoning")
		return ctx, false
	}

	return tombstoneSpanCtxError(ctx), true
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
