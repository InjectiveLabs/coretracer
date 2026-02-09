package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/InjectiveLabs/coretracer"
	otel "github.com/InjectiveLabs/coretracer/exporters/otel"
	"github.com/xlab/closer"
)

func main() {
	log.Println("[Hello!] We expect some SigNoz OTEL collector listening on DSN localhost:4317")

	coretracer.Enable(&coretracer.Config{
		ServiceName:           "example",
		ServiceVersion:        "1.0.0",
		EnvName:               "dev",
		CollectorDSN:          "localhost:4317",
		ClusterID:             "svc-us-east",
		StuckFunctionTimeout:  10 * time.Second,
		StuckFunctionWatchdog: true,
	}, otel.InitExporter)

	defer closer.Close()
	closer.Bind(func() {
		coretracer.Close()
	})

	time.Sleep(1 * time.Second)

	svc := &MyService{
		svcTags: coretracer.NewTags(map[string]any{
			"svc": "myService",
		}),
	}

	doneC := make(chan any, 2)
	go func() {
		doneC <- svc.ExampleWithContext8(context.Background())
	}()
	go func() {
		doneC <- svc.ExampleWithContext9(context.Background())
	}()

	svc.ExampleWithContext1(context.Background())
	svc.ExampleWithContext2(context.Background())
	svc.ExampleWithContext3(context.Background())
	svc.ExampleWithContext4(context.Background())
	svc.ExampleWithContext5(context.Background())
	svc.ExampleWithContext6(context.Background())
	svc.ExampleTraceless1(context.Background())

	// wait for the two stuck functions to finish
	<-doneC
	<-doneC

	fmt.Println(">>> check SigNoz http://localhost:3301/")
}

type MyService struct {
	svcTags coretracer.Tags
}

func (s *MyService) ExampleWithContext1(ctx context.Context) {
	defer coretracer.Trace(&ctx, s.svcTags)()

	func(ctx context.Context) {
		defer coretracer.Trace(&ctx, s.svcTags)()

		func(ctx context.Context) {
			defer coretracer.Trace(&ctx, s.svcTags)()

			func(ctx context.Context) {
				defer coretracer.Trace(&ctx, s.svcTags)()

				log.Println("Running a func that just happened")
				time.Sleep(1 * time.Second)
			}(ctx)
		}(ctx)
	}(ctx)
}

func (s *MyService) ExampleWithContext2(ctx context.Context) {
	defer coretracer.Trace(&ctx, s.svcTags)()

	log.Println("Running a func that just happened but errored")
	time.Sleep(1 * time.Second)

	coretracer.TraceError(ctx, errors.New("some error"))
}

func (s *MyService) ExampleWithContext3(ctx context.Context) {
	defer coretracer.Trace(&ctx, s.svcTags)()

	log.Println("Running a func that has a goroutine that just happened")

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func(ctx context.Context) {
		defer coretracer.TraceWithName(&ctx, "kawabanga_goroutine", s.svcTags)()
		defer wg.Done()

		time.Sleep(1 * time.Second)
		log.Println("Goroutine kawabanga just happened")
	}(ctx)
	wg.Wait()
}

func (s *MyService) ExampleWithContext4(ctx context.Context) {
	defer coretracer.Trace(&ctx, s.svcTags)()

	log.Println("Running a func that has a goroutine that happened and errored")

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func(ctx context.Context) {
		defer coretracer.TraceWithName(&ctx, "kawabanga_goroutine", s.svcTags)()
		defer wg.Done()

		time.Sleep(1 * time.Second)
		log.Println("Goroutine kawabanga errored")

		coretracer.TraceError(ctx, errors.New("some error"))
	}(ctx)
	wg.Wait()
}

func (s *MyService) ExampleWithContext5(ctx context.Context) {
	defer coretracer.Trace(&ctx, s.svcTags)()

	log.Println("Running a func that has a goroutine that happened and errored, then func errored")

	ohNo := false

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func(ctx context.Context) {
		defer coretracer.TraceWithName(&ctx, "kawabanga2_goroutine", s.svcTags)()
		defer wg.Done()

		time.Sleep(1 * time.Second)
		log.Println("Goroutine kawabanga2 errored")

		coretracer.TraceError(ctx, errors.New("some error 2"))
		ohNo = true
	}(ctx)
	wg.Wait()

	if ohNo {
		coretracer.TraceError(ctx, errors.New("goroutine errorred"))
	}
}

func (s *MyService) ExampleWithContext6(ctx context.Context) {
	defer coretracer.Trace(&ctx, s.svcTags)()

	log.Println("Running a func that just happened and spawned other func")
	time.Sleep(1 * time.Second)

	s.exampleWithContext7(ctx)
}

func (s *MyService) exampleWithContext7(ctx context.Context) {
	defer coretracer.Trace(&ctx, s.svcTags)()

	log.Println("Spawned exampleWithContext7 from ExampleWithContext6")
	time.Sleep(1 * time.Second)

	// attach some cool tags!
	coretracer.WithTags(ctx, coretracer.NewTag("is_spawned", true))
}

func (s *MyService) ExampleWithContext8(ctx context.Context) any {
	defer coretracer.Trace(&ctx, s.svcTags)()

	log.Println("Running a func that will be stuck")
	time.Sleep(11 * time.Second)

	log.Println("Func has been stuck for 11 seconds but now errored")
	coretracer.TraceError(ctx, errors.New("some error 3"))

	return nil
}

func (s *MyService) ExampleWithContext9(ctx context.Context) any {
	defer coretracer.Trace(&ctx, s.svcTags)()

	log.Println("Running a func that will be stuck")
	time.Sleep(11 * time.Second)

	log.Println("Func has been stuck for 11 seconds but now OK")

	return nil
}

func (s *MyService) ExampleTraceless1(ctx context.Context) {
	defer coretracer.Trace(&ctx, s.svcTags)()

	func() {
		defer coretracer.Traceless(nil, s.svcTags)()

		func() {
			defer coretracer.Traceless(nil, s.svcTags)()

			func() {
				defer coretracer.Traceless(nil, s.svcTags)()

				log.Println("Look mum no context!")
				time.Sleep(1 * time.Second)
			}()
		}()
	}()
}
