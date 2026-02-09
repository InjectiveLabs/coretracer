package stackcache

import (
	"testing"

	testingwrap "github.com/InjectiveLabs/coretracer/stackcache/internal/testingwrap"
	"github.com/stretchr/testify/require"
)

func TestGetCaller(t *testing.T) {
	wrappedCall1(t, func(t *testing.T) {
		cache := New(0, 1, "runtime")
		frame := cache.GetCaller()
		require.NotNil(t, frame)
		require.NotEmpty(t, frame.Function)
	})
}

func TestGetStackFrames(t *testing.T) {
	cache := New(0, 1, "runtime")

	wrappedCall1(t, func(t *testing.T) {
		frames := cache.GetStackFrames()
		require.NotNil(t, frames)
		require.NotEmpty(t, frames)
	})
}

func TestGetCallerWithSkip(t *testing.T) {
	wrappedCall1(t, func(t *testing.T) {
		cache := New(0, 2, "runtime")
		frame := cache.GetCaller()
		require.NotNil(t, frame)
		require.NotEmpty(t, frame.Function)

		expectedFrame := "github.com/InjectiveLabs/coretracer/stackcache.wrappedCall3"
		require.Equal(t, expectedFrame, frame.Function)
	})
}

func TestGetStackFramesWithSkip(t *testing.T) {
	wrappedCall1(t, func(t *testing.T) {
		cache := New(0, 1, "runtime")
		frames := cache.GetStackFrames()
		require.NotNil(t, frames)
		require.NotEmpty(t, frames)

		expectedStack := []string{
			"github.com/InjectiveLabs/coretracer/stackcache.TestGetStackFramesWithSkip.func1",
			"github.com/InjectiveLabs/coretracer/stackcache.wrappedCall3",
			"github.com/InjectiveLabs/coretracer/stackcache.wrappedCall2",
			"github.com/InjectiveLabs/coretracer/stackcache.wrappedCall1",
			"github.com/InjectiveLabs/coretracer/stackcache.TestGetStackFramesWithSkip",
			"testing.tRunner",
		}

		for i, frame := range frames {
			require.Equal(t, expectedStack[i], frame.Function)
		}
	})
}

func TestGetCallerWithBreakpointPackage(t *testing.T) {
	cache := New(0, 0, "github.com/InjectiveLabs/coretracer/stackcache/internal/testingwrap")

	wrappedCall1(t, func(t *testing.T) {
		frame := testingwrap.GetCaller(cache)
		require.NotNil(t, frame)
		require.NotEmpty(t, frame.Function)

		expectedFrame := "github.com/InjectiveLabs/coretracer/stackcache.TestGetCallerWithBreakpointPackage.func1"
		require.Equal(t, expectedFrame, frame.Function)
	})
}

func TestGetStackFramesWithBreakpointPackage(t *testing.T) {
	cache := New(0, 0, "github.com/InjectiveLabs/coretracer/stackcache/internal/testingwrap")

	wrappedCall1(t, func(t *testing.T) {
		frames := testingwrap.GetStackFrames(cache)
		require.NotNil(t, frames)
		require.NotEmpty(t, frames)

		expectedStack := []string{
			"github.com/InjectiveLabs/coretracer/stackcache.TestGetStackFramesWithBreakpointPackage.func1",
			"github.com/InjectiveLabs/coretracer/stackcache.wrappedCall3",
			"github.com/InjectiveLabs/coretracer/stackcache.wrappedCall2",
			"github.com/InjectiveLabs/coretracer/stackcache.wrappedCall1",
			"github.com/InjectiveLabs/coretracer/stackcache.TestGetStackFramesWithBreakpointPackage",
			"testing.tRunner",
		}

		for i, frame := range frames {
			require.Equal(t, expectedStack[i], frame.Function)
		}
	})
}

func wrappedCall1(t *testing.T, testFunc func(t *testing.T)) {
	wrappedCall2(t, testFunc)
}

func wrappedCall2(t *testing.T, testFunc func(t *testing.T)) {
	wrappedCall3(t, testFunc)
}

func wrappedCall3(t *testing.T, testFunc func(t *testing.T)) {
	testFunc(t)
}

func TestPackageName(t *testing.T) {
	pkg := PackageName("github.com/InjectiveLabs/coretracer/stackcache.TestGetPackageName")
	require.Equal(t, "github.com/InjectiveLabs/coretracer/stackcache", pkg)
}

func TestGetStackFramesWithLongStackTrace(t *testing.T) {
	cache := New(0, 2, "runtime")

	// Simulate a long stack trace by calling nested functions
	nestedCall1(t, cache, func(t *testing.T, cache StackCache) {
		frames := cache.GetStackFrames()
		require.NotNil(t, frames)
		require.NotEmpty(t, frames)
		require.Equal(t, 12, len(frames))

		require.Equal(t, "github.com/InjectiveLabs/coretracer/stackcache.nestedCall10", frames[0].Function)
		require.Equal(t, "github.com/InjectiveLabs/coretracer/stackcache.nestedCall9", frames[1].Function)
		require.Equal(t, "github.com/InjectiveLabs/coretracer/stackcache.nestedCall8", frames[2].Function)
		require.Equal(t, "github.com/InjectiveLabs/coretracer/stackcache.nestedCall7", frames[3].Function)
		require.Equal(t, "github.com/InjectiveLabs/coretracer/stackcache.nestedCall6", frames[4].Function)
		require.Equal(t, "github.com/InjectiveLabs/coretracer/stackcache.nestedCall5", frames[5].Function)
		require.Equal(t, "github.com/InjectiveLabs/coretracer/stackcache.nestedCall4", frames[6].Function)
		require.Equal(t, "github.com/InjectiveLabs/coretracer/stackcache.nestedCall3", frames[7].Function)
		require.Equal(t, "github.com/InjectiveLabs/coretracer/stackcache.nestedCall2", frames[8].Function)
		require.Equal(t, "github.com/InjectiveLabs/coretracer/stackcache.nestedCall1", frames[9].Function)
		require.Equal(t, "github.com/InjectiveLabs/coretracer/stackcache.TestGetStackFramesWithLongStackTrace", frames[10].Function)
		require.Equal(t, "testing.tRunner", frames[len(frames)-1].Function)
	})
}

func nestedCall1(t *testing.T, cache StackCache, testFunc func(t *testing.T, cache StackCache)) {
	nestedCall2(t, cache, testFunc)
}

func nestedCall2(t *testing.T, cache StackCache, testFunc func(t *testing.T, cache StackCache)) {
	nestedCall3(t, cache, testFunc)
}

func nestedCall3(t *testing.T, cache StackCache, testFunc func(t *testing.T, cache StackCache)) {
	nestedCall4(t, cache, testFunc)
}

func nestedCall4(t *testing.T, cache StackCache, testFunc func(t *testing.T, cache StackCache)) {
	nestedCall5(t, cache, testFunc)
}

func nestedCall5(t *testing.T, cache StackCache, testFunc func(t *testing.T, cache StackCache)) {
	nestedCall6(t, cache, testFunc)
}

func nestedCall6(t *testing.T, cache StackCache, testFunc func(t *testing.T, cache StackCache)) {
	nestedCall7(t, cache, testFunc)
}

func nestedCall7(t *testing.T, cache StackCache, testFunc func(t *testing.T, cache StackCache)) {
	nestedCall8(t, cache, testFunc)
}

func nestedCall8(t *testing.T, cache StackCache, testFunc func(t *testing.T, cache StackCache)) {
	nestedCall9(t, cache, testFunc)
}

func nestedCall9(t *testing.T, cache StackCache, testFunc func(t *testing.T, cache StackCache)) {
	nestedCall10(t, cache, testFunc)
}

func nestedCall10(t *testing.T, cache StackCache, testFunc func(t *testing.T, cache StackCache)) {
	testFunc(t, cache)
}

func BenchmarkGetStackFramesWithLongStackTrace(b *testing.B) {
	cache := New(0, 2, "runtime")

	for i := 0; i < b.N; i++ {
		bNestedCall1(b, cache, func(b *testing.B, cache StackCache) {
			frames := cache.GetStackFrames()

			if len(frames) != 13 {
				for _, frame := range frames {
					b.Log(frame.Function)
				}

				b.Fatalf("ERROR: expected 13 frames, got %d", len(frames))
			}
		})
	}
}

func bNestedCall1(b *testing.B, cache StackCache, testFunc func(b *testing.B, cache StackCache)) {
	bNestedCall2(b, cache, testFunc)
}

func bNestedCall2(b *testing.B, cache StackCache, testFunc func(b *testing.B, cache StackCache)) {
	bNestedCall3(b, cache, testFunc)
}

func bNestedCall3(b *testing.B, cache StackCache, testFunc func(b *testing.B, cache StackCache)) {
	bNestedCall4(b, cache, testFunc)
}

func bNestedCall4(b *testing.B, cache StackCache, testFunc func(b *testing.B, cache StackCache)) {
	bNestedCall5(b, cache, testFunc)
}

func bNestedCall5(b *testing.B, cache StackCache, testFunc func(b *testing.B, cache StackCache)) {
	bNestedCall6(b, cache, testFunc)
}

func bNestedCall6(b *testing.B, cache StackCache, testFunc func(b *testing.B, cache StackCache)) {
	bNestedCall7(b, cache, testFunc)
}

func bNestedCall7(b *testing.B, cache StackCache, testFunc func(b *testing.B, cache StackCache)) {
	bNestedCall8(b, cache, testFunc)
}

func bNestedCall8(b *testing.B, cache StackCache, testFunc func(b *testing.B, cache StackCache)) {
	bNestedCall9(b, cache, testFunc)
}

func bNestedCall9(b *testing.B, cache StackCache, testFunc func(b *testing.B, cache StackCache)) {
	bNestedCall10(b, cache, testFunc)
}

func bNestedCall10(b *testing.B, cache StackCache, testFunc func(b *testing.B, cache StackCache)) {
	testFunc(b, cache)
}

func TestGetStackFramesWithBreakpointPackage_RuntimePackageAfterBreakpointPackage(t *testing.T) {
	cache := New(0, 0, "github.com/InjectiveLabs/coretracer/stackcache")

	wrappedCall1(t, func(t *testing.T) {
		frames := testingwrap.GetStackFrames(cache)
		require.NotNil(t, frames)
		require.NotEmpty(t, frames)

		expectedStack := []string{
			"github.com/InjectiveLabs/coretracer/stackcache/internal/testingwrap.GetStackFrames",
			"github.com/InjectiveLabs/coretracer/stackcache.TestGetStackFramesWithBreakpointPackage_RuntimePackageAfterBreakpointPackage",
			"testing.tRunner",
		}

		for i, frame := range frames {
			require.Equal(t, expectedStack[i], frame.Function)
		}
	})
}

func TestGetCallerWithBreakpointPackage_RuntimePackageAfterBreakpointPackage(t *testing.T) {
	cache := New(0, 0, "github.com/InjectiveLabs/coretracer/stackcache")

	wrappedCall1(t, func(t *testing.T) {
		frame := testingwrap.GetCaller(cache)

		require.NotNil(t, frame)
		require.NotEmpty(t, frame.Function)
		require.Equal(t, "github.com/InjectiveLabs/coretracer/stackcache/internal/testingwrap.GetCaller", frame.Function)
	})
}

func TestGetStackFrames_UsefulFrameAfterBreakpointPackageCallFromRuntime(t *testing.T) {
	cache := New(0, 0, "github.com/InjectiveLabs/coretracer/stackcache/internal/testingwrap")

	wrappedCall1(t, func(t *testing.T) {
		defer func() {
			frames := testingwrap.GetStackFrames(cache)
			require.NotNil(t, frames)
			require.NotEmpty(t, frames)

			expectedStack := []string{
				"github.com/InjectiveLabs/coretracer/stackcache.TestGetStackFrames_UsefulFrameAfterBreakpointPackageCallFromRuntime.func1.1", // <- useful frame
				// skipped the breakpoint package
				"runtime.gopanic", // <- runtime package as part of the trace
				"github.com/InjectiveLabs/coretracer/stackcache.TestGetStackFrames_UsefulFrameAfterBreakpointPackageCallFromRuntime.func1",
				"github.com/InjectiveLabs/coretracer/stackcache.wrappedCall3",
				"github.com/InjectiveLabs/coretracer/stackcache.wrappedCall2",
				"github.com/InjectiveLabs/coretracer/stackcache.wrappedCall1",
				"github.com/InjectiveLabs/coretracer/stackcache.TestGetStackFrames_UsefulFrameAfterBreakpointPackageCallFromRuntime",
				"testing.tRunner",
			}

			for i, frame := range frames {
				require.Equal(t, expectedStack[i], frame.Function)
			}

			_ = recover()
		}()

		panic("foo")
	})
}

func TestGetCaller_DeepWrapWithRuntimeRecovers(t *testing.T) {
	cache := New(0, 0, "github.com/InjectiveLabs/coretracer/stackcache/internal/testingwrap")

	wrappedCall1(t, func(t *testing.T) {
		testingwrap.WrapCall(cache, func(st testingwrap.StackCache) {
			defer func() {
				defer func() {
					frame := testingwrap.GetCaller(cache)
					require.NotNil(t, frame)
					require.NotEmpty(t, frame.Function)
					require.Equal(t, "github.com/InjectiveLabs/coretracer/stackcache.TestGetCaller_DeepWrapWithRuntimeRecovers.wrappedCall1.wrappedCall2.wrappedCall3.TestGetCaller_DeepWrapWithRuntimeRecovers.func1.func2.1.1", frame.Function)

					_ = recover()
				}()

				_ = recover()
				panic("foo 2")
			}()

			panic("foo")
		})
	})
}

func TestFuncName(t *testing.T) {
	tests := []struct {
		fullName string
		expected string
	}{
		{"", ""},
		{"github.com/InjectiveLabs/coretracer/stackcache/_underscore", "_underscore"},
		{"github.com/InjectiveLabs/coretracer/stackcache/invalid.$1", "invalid.$1"},
		{"github.com/InjectiveLabs/coretracer/stackcache/$invalid", "$invalid"},
		{"github.com/InjectiveLabs/coretracer/stackcache.TestFuncName", "TestFuncName"},
		{"github.com/InjectiveLabs/coretracer/stackcache.funcName", "funcName"},
		{"github.com/InjectiveLabs/coretracer/stackcache.TestFuncName.func1", "func1"},
		{"github.com/InjectiveLabs/coretracer/stackcache/internal/testingwrap.GetCaller", "GetCaller"},
		{"github.com/InjectiveLabs/coretracer/stackcache.TestGetCaller_UsefulFrameAfterBreakpointPackageCallFromRuntime.wrappedCall1.wrappedCall2.wrappedCall3.TestGetCaller_UsefulFrameAfterBreakpointPackageCallFromRuntime.func1.func2.1.1", "func2.1.1"},
		{"github.com/InjectiveLabs/coretracer/stackcache.TestGetCaller_UsefulFrameAfterBreakpointPackageCallFromRuntime.wrappedCall1.wrappedCall2.wrappedCall3.TestGetCaller_UsefulFrameAfterBreakpointPackageCallFromRuntime.func1.func2", "func2"},
		{"runtime.main", "main"},
	}

	for _, test := range tests {
		t.Run(test.fullName, func(t *testing.T) {
			result := FuncName(test.fullName)
			require.Equal(t, test.expected, result)
		})
	}
}
