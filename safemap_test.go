package coretracer

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSafeMap(t *testing.T) {
	sm := newSafeMap()
	require.NotNil(t, sm.mux, "Expected mux to be initialized")
	require.NotNil(t, sm.m, "Expected map to be initialized")
}

func TestNewSafeMapWith(t *testing.T) {
	key, value := "key", "value"
	sm := newSafeMapWith(key, value)
	require.NotNil(t, sm.mux, "Expected mux to be initialized")
	require.NotNil(t, sm.m, "Expected map to be initialized")
	require.Equal(t, value, sm.m[key], "Expected map to contain %s: %s", key, value)
}

func TestSafeMap_IsValid(t *testing.T) {
	sm := newSafeMap()
	require.True(t, sm.IsValid(), "Expected SafeMap to be valid")
}

func TestSafeMap_Set(t *testing.T) {
	sm := newSafeMap()
	key, value := "key", "value"
	sm = sm.Set(key, value)
	require.Equal(t, value, sm.m[key], "Expected map to contain %s: %s", key, value)
}

func TestSafeMap_RLock(t *testing.T) {
	sm := newSafeMap()
	sm.RLock()
	defer sm.RUnlock()
	// No panic expected

	_ = sm.m["kek"]
}

func TestSafeMap_RUnlock(t *testing.T) {
	sm := newSafeMap()
	sm.RLock()
	_ = sm.m["kek"]
	sm.RUnlock()
	// No panic expected
}

func TestSafeMap_Map(t *testing.T) {
	sm := newSafeMap()
	require.NotNil(t, sm.Map(), "Expected map to be returned")
}

func BenchmarkSafeMap(b *testing.B) {
	sm := newSafeMap()
	var wg sync.WaitGroup

	b.Run("writeOne", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				key := fmt.Sprintf("key%d", i)
				value := fmt.Sprintf("value%d", i)
				sm.Set(key, value)
			}(i)
		}
		wg.Wait()
	})

	b.Run("readOne", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sm.RLock()
				for k, v := range sm.Map() {
					_ = k
					_ = v
					break
				}
				sm.RUnlock()
			}()
		}
		wg.Wait()
	})

	b.Run("readAll", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sm.RLock()
				for k, v := range sm.Map() {
					_ = k
					_ = v
				}
				sm.RUnlock()
			}()
		}
		wg.Wait()
	})
}
