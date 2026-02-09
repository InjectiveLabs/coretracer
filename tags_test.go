package coretracer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTags(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		tags := NewTags()
		require.True(t, SafeMap(tags).IsValid(), "Expected Tags to be valid")
		require.Empty(t, SafeMap(tags).Map(), "Expected Tags to be empty")
	})

	t.Run("single map input", func(t *testing.T) {
		input := map[string]any{"key1": "value1"}
		tags := NewTags(input)
		require.True(t, SafeMap(tags).IsValid(), "Expected Tags to be valid")
		require.Equal(t, input, SafeMap(tags).Map(), "Expected Tags to match input map")
	})

	t.Run("multiple maps input", func(t *testing.T) {
		input1 := map[string]any{"key1": "value1"}
		input2 := map[string]any{"key2": "value2"}
		tags := NewTags(input1, input2)
		expected := map[string]any{"key1": "value1", "key2": "value2"}
		require.True(t, SafeMap(tags).IsValid(), "Expected Tags to be valid")
		require.Equal(t, expected, SafeMap(tags).Map(), "Expected Tags to match union of input maps")
	})
}

func TestTags_With(t *testing.T) {
	tags := NewTags()
	tags = tags.With("key", "value")
	require.Equal(t, "value", SafeMap(tags).Map()["key"], "Expected Tags to contain key: value")
}

func TestTags_Range(t *testing.T) {
	tags := NewTags(map[string]any{"key1": "value1", "key2": "value2"})
	collected := make(map[string]any)
	tags.Range(func(k string, v any) bool {
		collected[k] = v
		return true
	})
	require.Equal(t, SafeMap(tags).Map(), collected, "Expected Range to iterate over all key-value pairs")
}

func TestTags_WithGlobalTags(t *testing.T) {
	config = &Config{EnvName: "test_env"}
	tags := NewTags(map[string]any{"key1": "value1"})
	tags = tags.WithGlobalTags()
	expected := map[string]any{"deployment.environment": "test_env", "key1": "value1"}
	require.Equal(t, expected, SafeMap(tags).Map(), "Expected Tags to include global tags")
}

func TestTags_RangeEarlyStop(t *testing.T) {
	tags := NewTags(map[string]any{"key1": "value1", "key2": "value2", "key3": "value3"})
	collected := make(map[string]any)
	tags.Range(func(k string, v any) bool {
		collected[k] = v
		return false // Stop after the first element
	})
	require.Len(t, collected, 1, "Expected Range to stop early and collect only one element")
}

func TestTags_WithOnNonInitializedMap(t *testing.T) {
	var tags Tags // Non-initialized map
	tags = tags.With("key", "value")
	require.True(t, SafeMap(tags).IsValid(), "Expected Tags to be valid after With() on non-initialized map")
	require.Equal(t, "value", SafeMap(tags).Map()["key"], "Expected Tags to contain key: value after With() on non-initialized map")
}

func TestTags_Union(t *testing.T) {
	tags1 := NewTags(map[string]any{"key1": "value1"})
	tags2 := NewTags(map[string]any{"key2": "value2"})
	tags3 := NewTags(map[string]any{"key3": "value3"})

	unionTags := tags1.Union(tags2, tags3)
	expected := map[string]any{"key1": "value1", "key2": "value2", "key3": "value3"}

	require.True(t, SafeMap(unionTags).IsValid(), "Expected Union Tags to be valid")
	require.Equal(t, expected, SafeMap(unionTags).Map(), "Expected Union Tags to match union of input tags")
}
