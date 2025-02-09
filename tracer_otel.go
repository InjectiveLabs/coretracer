package coretracer

import (
	"context"
	"fmt"
	"runtime"
	"time"

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
func (t *otelTracer) Close() error {
	return nil
}

// Trace implements Tracer.
func (t *otelTracer) Trace(ctx *context.Context, tags ...Tags) SpanEnderFn {
	defer func() {
		t.logger.Println("[PANIC] Trace() panicked - this is a bug")
		recover()
	}()

	frame := t.stackCache.GetCaller()
	funcName := stackcache.FuncName(frame.Function)

	return t.traceStart(ctx, funcName, false, tags)
}

// TraceError implements Tracer.
func (t *otelTracer) TraceError(ctx context.Context, err error) {
	defer func() {
		t.logger.Println("[PANIC] TraceError() panicked - this is a bug")
		recover()
	}()

	if ctx == nil {
		ctx = context.Background()
	}

	span := oteltracer.SpanFromContext(ctx)
	if span == nil {
		t.logger.Println("[WARN] no span found in context - TraceError() with invalid context")
		return
	}

	_, alreadyTombstoned := t.checkAndSetTombstoneSpanCtxError(ctx)
	if !alreadyTombstoned && span.IsRecording() {
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
}

// TraceWithName implements Tracer.
func (t *otelTracer) TraceWithName(ctx *context.Context, name string, tags ...Tags) SpanEnderFn {
	return t.traceStart(ctx, name, false, tags)
}

// Traceless implements Tracer.
func (t *otelTracer) Traceless(ctx *context.Context, tags ...Tags) SpanEnderFn {
	frame := t.stackCache.GetCaller()
	funcName := stackcache.FuncName(frame.Function)

	return t.traceStart(ctx, funcName, true, tags)
}

// TracelessWithName implements Tracer.
func (t *otelTracer) TracelessWithName(ctx *context.Context, name string, tags ...Tags) SpanEnderFn {
	return t.traceStart(ctx, name, true, tags)
}

func (t *otelTracer) traceStart(ctx *context.Context, funcName string, virtualTrace bool, tags []Tags) SpanEnderFn {
	if ctx == nil {
		emptyCtx := context.Background()
		ctx = &emptyCtx

		virtualTrace = true
	}

	allTags := NewTags().Union(tags...)
	attributes := make([]otelattribute.KeyValue, 0, 10)
	allTags.Range(func(k string, v any) bool {
		attributes = append(attributes, anyToOtalAttribute(k, v))
		return true
	})

	var parentSpans []oteltracer.Span
	parentSpansEndFn := func(spansToEnd []oteltracer.Span) {}

	if virtualTrace {
		now := time.Now().UTC()
		frames := t.stackCache.GetStackFrames()

		*ctx, parentSpans = t.callStackFramesToSpans(*ctx, now, frames, attributes)

		parentSpansEndFn = func(spansToEnd []oteltracer.Span) {
			if len(spansToEnd) == 0 {
				return
			}

			// iterate in reverse order to end the spans in the correct order
			for i := len(spansToEnd) - 1; i >= 0; i-- {
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

	// set the modified context in-place
	*ctx = modifiedContext

	return func() {
		var alreadyTombstoned bool

		// pointer to ctx is captured by closure, yet we can update the original value in-place
		*ctx, alreadyTombstoned = t.checkAndSetTombstoneSpanCtxDone(*ctx)

		if !alreadyTombstoned && span.IsRecording() {
			span.SetStatus(otelcodes.Ok, "")
			span.End()
		}

		parentSpansEndFn(parentSpans)
	}
}

func (t *otelTracer) callStackFramesToSpans(
	ctx context.Context,
	timestamp time.Time,
	frames []runtime.Frame,
	attributes []otelattribute.KeyValue,
) (context.Context, []oteltracer.Span) {
	spans := make([]oteltracer.Span, 0, len(frames))

	var newSpan oteltracer.Span
	for _, frame := range frames {
		ctx, newSpan = t.tracer.Start(
			ctx,
			stackcache.FuncName(frame.Function),
			oteltracer.WithAttributes(attributes...),
			oteltracer.WithTimestamp(timestamp),
		)

		spans = append(spans, newSpan)
	}

	return ctx, spans
}

// WithTags implements Tracer.
func (t *otelTracer) WithTags(ctx context.Context, tags ...Tags) {
	panic("unimplemented")
}

// SetCallStackOffset implements Tracer.
func (t *otelTracer) SetCallStackOffset(offset int) {
	if offset < 0 {
		offset = 0
	}

	t.callStackOffset = offset
}

func Close() {
	tracerMux.RLock()
	defer tracerMux.RUnlock()

	if tracer == nil {
		return
	}

	tracer.Close()
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
	if isTombstoneSpanCtxDone(ctx) || isTombstoneSpanCtxError(ctx) || isTombstoneSpanCtxStuck(ctx) {
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
