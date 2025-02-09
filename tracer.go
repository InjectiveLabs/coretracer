package coretracer

import (
	"context"
	"sync"
)

var (
	tracer    Tracer
	tracerMux = new(sync.RWMutex)

	config *Config
)

type SpanEnderFn func()

type Tracer interface {
	Trace(ctx *context.Context, tags ...Tags) SpanEnderFn
	TraceWithName(ctx *context.Context, name string, tags ...Tags) SpanEnderFn
	TraceError(ctx context.Context, err error)
	Traceless(tags ...Tags) SpanEnderFn
	TracelessWithName(ctx *context.Context, name string, tags ...Tags) SpanEnderFn

	WithTags(ctx context.Context, tags ...Tags)
	SetCallStackOffset(offset int)
	Close() error
}

func Enable(cfg *Config) {
	tracerMux.Lock()
	defer tracerMux.Unlock()
	tracer = newOtelTracer(cfg)
}

func Disable() {
	tracerMux.Lock()
	defer tracerMux.Unlock()
	tracer = nil
}

func DefaultTracer() Tracer {
	tracerMux.RLock()
	defer tracerMux.RUnlock()

	return tracer
}
