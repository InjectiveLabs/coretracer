package coretracer

import "sync"

// SafeMap is a map that is safe to use in concurrent code.
// Protected by a RWMutex. Do not overcomplicate with sync.Map.
type SafeMap struct {
	mux *sync.RWMutex
	m   map[string]any
}

func newSafeMap() SafeMap {
	return SafeMap{
		mux: new(sync.RWMutex),
		m:   make(map[string]any),
	}
}

func newSafeMapWith(k, v string) SafeMap {
	return SafeMap{
		mux: new(sync.RWMutex),
		m: map[string]any{
			k: v,
		},
	}
}

func (sm SafeMap) IsValid() bool {
	return sm.mux != nil
}

func (sm SafeMap) Set(k string, v any) SafeMap {
	sm.mux.Lock()
	sm.m[k] = v
	sm.mux.Unlock()

	return sm
}

func (sm SafeMap) RLock() {
	sm.mux.RLock()
}

func (sm SafeMap) RUnlock() {
	sm.mux.RUnlock()
}

func (sm SafeMap) Map() map[string]any {
	return sm.m
}
