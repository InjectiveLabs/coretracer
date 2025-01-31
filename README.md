# CoreTracer

Welcome to CoreTracer! A package for tracing Go applications, re-invented with all recent insights on metrics collection and tracing.
It is a spiritual successor of https://github.com/InjectiveLabs/metrics, but we omit the "metrics" part to fully focus on tracing.

Certain concepts are borrowed from the `metrics` package, but its API is redesigned to have very minimal surface and internal processing
to leverage most of the [OpenTraicing](https://opentracing.io).

CoreTracer is built around `go.opentelemetry.io/otel` package and its `Tracer` interface. Any Otel-compatible pipeline can be used to collect and aggregate the traces collected by this tool.

We anticipate to use the following OTel receivers:

- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
- [Uptrace](https://uptrace.dev/)
- [Datadog](https://docs.datadoghq.com/tracing/)

There are cerain ways to transform traces into metrics, a great article on this topic:
https://newrelic.com/blog/nerdlog/transforming-traces

[<img src="https://newrelic.com/sites/default/files/styles/1800w/public/2024-03/connector.webp?itok=fkc8qH1S" alt="Transforming Traces into Metrics" width="600" />](https://newrelic.com/blog/nerdlog/transforming-traces)

## Future Goals

- Forward traces to Uptrace, Datadog, etc
- Transform traces into metrics
- Export metrics to InfluxDB, Grafana

## Basic Usage

```go
import "github.com/InjectiveLabs/coretracer"


type MyService struct {
    svcTags coretracer.Tags
}

func NewMyService() *MyService {
    return &MyService{
        svcTags: coretracer.Tags{
            "svc": "myService",
        },
    }
}

func (s *MyService) SomeFunc(ctx context.Context) error {
    defer coretracer.Trace(&ctx, s.svcTags)()

    // do a subcall
    s.SomeOtherFunc(ctx)

    err := db.Query(ctx, "SELECT * FROM users")
    if err != nil {
        coretracer.TraceError(ctx, err)
        return err
    }
    
    return nil
}

func (s *MyService) SomeOtherFunc(ctx context.Context) {
    defer coretracer.Trace(&ctx, s.svcTags)()
    // do something
}
```

### Key points here:

- `coretracer.Trace` is used to initiate a span within a service method
- `coretracer.TraceError` is used to end span, add the error and mark span as failed
- `coretracer.Tags` is used to add tags to the trace span

The line `defer coretracer.Trace(&ctx)()` unfolds into the following runtime actions:
* `Trace()` returns a `type SpanEnder func()` that ends span with a success and records the duration.
* The end function is called when the method returns, so it's `defer`red.
* We pass a pointer to the context inteface (`*context.Context`), so the context can be updated with the span data in-place.
* The context is enriched with the span reference, so we can call `coretracer.TraceError` on the same context later.
* Ending the span with `TraceError` will mark it as failed and add the error to the span, but also tombstone the context.
* If the span ended with an error (and context is tombstoned), execution of the deferred `SpanEnder` function will be no-op.
* If user calls `TraceError` on a tombstoned context, it will emit a warning with a stacktrace in the logs, considered a programming error.
* Context could be a Cosmos SDK context, in that case the real Go context will be replaced in-place for `ctx.Context()`.
* Passing `nil` context will emit a warning with a stacktrace in the logs, considered a programming error. Falls back to `Traceless` (see below).
* We avoid panics to make sure a smooth transition from the old `metrics` package.

### The trick with context in-place update:
```go
ctx := context.Background()
fmt.Printf("pointer to context: %p\n", &ctx)
defer coretracer.Trace(&ctx)()
fmt.Printf("pointer to a new context: %p\n", &ctx)
```

It allows to keep a single return value of `coretracer.Trace` that can be used to end the span only.

### Summary about the basic usage

* API surface must be minimal, user must only care about when methiod begins and when method fails.
* Only 1 call to coretracer is needed to measure method duration and status.
* Additional 1 call per erroring branch of code.
* User needs to pass the context to be able to trace the sequence of calls.

## Usage with additional tags

`coretracer.Tags` provides a way to add additional tags to the span. Note that tags for the span are managed before span is created.

```go
svcTags := coretracer.Tags{
    "svc": "myService",
}

additonalTags := svcTags.WithTags(coretracer.Tags{ 
    "block_height": 3245674,
})

defer coretracer.Trace(&ctx, additonalTags)()
```

Sometimes tags only known at certain point of the span execution, so we can use `WithTags` to add them.

```go
svcTags := coretracer.Tags{
    "svc": "myService",
}

defer coretracer.Trace(&ctx, svcTags)()

// much later

additonalTags := coretracer.Tags{ 
    "block_height": 3245674,
}

// coretracer.WithTags will fetch the span from the context and add tags to it
coretracer.WithTags(ctx, additonalTags)
```

## Usage with Closures (anonymous functions)

Sometimes, especially with Goroutines, we need to trace anonymous functions. There is a helper function `coretracer.TraceWithName` that can be used to trace a function and give it a name explicitly.

```go
func SomeFunc(ctx context.Context) {
    defer coretracer.Trace(&ctx, "SomeFunc")()

    // run a goroutine
    go func() {
        defer coretracer.TraceWithName(&ctx, "SomeOtherFuncGoroutine")()
        // do something
        // ctx here holds the span of SomeOtherFuncGoroutine
    }()

    // ctx here holds the span of SomeFunc
    coretracer.WithTags(ctx, coretracer.Tags{ 
        "is_goroutine": false,
    })
}
```

The trickiest of all tricks here is that the root context is captured as a value, so the reference to the context inside the goroutine closure will update only that child context with a value. The parent context of `SomeFunc` will hold the original value of the `SomeFunc` span.

While this sounds scary, a typical usage pattern is to just avoid overthinking and keep placing the `coretracer.Trace*` calls in the beginning of the most functions.

## Usage without Context

While spans are generated from the context, it's possible to create a span without it. This is still useful because the actual call stack can be deducted from the stack dump. This is the same as getting the function name from the latest stack frame, but instead, we get all the parent function names as well and construct a trace path out of many virtual spans.

Unfortunately, there is no way to get the timing of the parent spans. So, only the latest span will have the duration. We assume that there aren't many functions that lack a context.

```go
import "github.com/InjectiveLabs/coretracer"


type MyService struct {
    svcTags coretracer.Tags
}

func NewMyService() *MyService {
    return &MyService{
        svcTags: coretracer.Tags{
            "svc": "myService",
        },
    }
}

func (s *MyService) SomeFuncWithoutContext() error {
    defer coretracer.Traceless(s.svcTags)()

    // do a subcall
    s.SomeOtherFuncWithoutContext()
    
    return nil
}


func (s *MyService) SomeOtherFuncWithoutContext() {
    defer coretracer.Traceless(s.svcTags)()
    // do something
}
```

## Usage with No Context, No Name

For the sake of API completeness, there is a `TracelessWithName`. But we discourage doing that.

```go
func (s *MyService) SomeFuncWithoutContext() error {
    defer coretracer.Traceless(nil, s.svcTags)()

    // run a goroutine
    go func() {
        ctx := context.Background()
        defer coretracer.TracelessWithName(&ctx, "SomeOtherFuncGoroutine", s.svcTags)()

        // do stuff
        coretracer.WithTags(ctx, coretracer.Tags{ 
            "is_goroutine": true,
        })
    }()
    
    return nil
}
```

Traceless won't care about the context that is `nil`. It will just create a span without prior context. If the context is set, it will be used to:
* Fetch an existing span from the context, if span exists, `coretracer.Traceless` will act the same way as `coretracer.Trace`.
* If span doesn't exist, it will create a new span sequence from the runtime call stack.
* If the context is nil, it will not update the context with the span reference. Otherwise current span is set in the context.

## Summary

* `coretracer.Trace` is used to trace a method with a context.
* `coretracer.Traceless` is used to trace a method without a context.
* `coretracer.TraceWithName` is used to trace an anonymous closure with a given name.
* `coretracer.TracelessWithName` is used to trace an anonymous closure without a context, with a given name.
* `coretracer.Tags` is used to add tags to the span.
* `coretracer.WithTags` can add more tags to the existing span.
* `coretracer.TraceError` is used to end span, set the error and mark span as failed.

All tracing functions are here to collect as much info about call stack, associate tags and measure timing of function execution.
They're designed to have very little overhead in terms of line code and runtime performance.

If tracing is disabled, all tracing functions are no-op and should not affect the runtime performance. Including the tags management - tag joins are heavy on GC if used in hot paths. So `WithTags` will no-op.

When enbled, the tracing info will be processed with OTal config and client and will be sent to the collecting backend.

## Tracing Config for OTel

```go
// TODO
// Endpoints, Enabled, Env vars, default buffer size
```

## License

MIT
