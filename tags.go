package coretracer

import "sync"

// Tags is a safe map of tags. Used to attach tags to a trace.
type Tags SafeMap

// NewTags unions unsafe maps as a safe maps used for Tags.
// It is ok to have empty maps, nil maps, or no arguments at all.
func NewTags(mapsToUnion ...map[string]any) Tags {
	if len(mapsToUnion) == 0 {
		return Tags(newSafeMap())
	}

	safeMap := SafeMap{
		mux: new(sync.RWMutex),
		m:   make(map[string]any, len(mapsToUnion[0])),
	}

	for _, m := range mapsToUnion {
		for k, v := range m {
			safeMap.m[k] = v
		}
	}

	return Tags(safeMap)
}

// With adds a new tag to the set. It will modify the existing set.
func (t Tags) With(k, v string) Tags {
	if t.mux == nil {
		return Tags(newSafeMapWith(k, v))
	}

	return Tags(SafeMap(t).Set(k, v))
}

// Range iterates over the tags and calls the provided function for each key-value pair.
// The iteration stops when the provided function returns false.
// The boolean meaning has to comply with Go 1.23+ iterator pattern.
func (t Tags) Range(rangeFn func(k string, v any) (valid bool)) {
	if t.mux == nil {
		return
	}

	t.mux.RLock()
	defer t.mux.RUnlock()

	for k, v := range t.m {
		if valid := rangeFn(k, v); !valid {
			return
		}
	}
}

// WithGlobalTags allows to inject GlobalTags into custom tag set Tags,
// useful for re-using tags for purposes other than tracing.
func (t Tags) WithGlobalTags() Tags {
	globalTags := config.GlobalTagsMap()
	allTags := newSafeMap()

	for k, v := range globalTags {
		allTags.m[k] = v
	}

	t.mux.RLock()
	defer t.mux.RUnlock()

	for k, v := range t.m {
		allTags.m[k] = v
	}

	return Tags(allTags)
}

// Union merges tags from the provided tags and returns a new Tags copy.
func (t Tags) Union(tags ...Tags) Tags {
	allTags := newSafeMap()

	if t.mux != nil {
		t.mux.RLock()
		for k, v := range t.m {
			allTags.m[k] = v
		}
		t.mux.RUnlock()
	}

	for _, tagMap := range tags {
		if tagMap.mux == nil {
			continue
		}

		tagMap.mux.RLock()
		for k, v := range tagMap.m {
			allTags.m[k] = v
		}
		tagMap.mux.RUnlock()
	}

	return Tags(allTags)
}
