package collector

import (
	"maps"
	"slices"
	"sync"
)

// statCache accumulates keyed statistics across scrapes via a merge function,
// so counters keep growing even though the upstream API returns windowed
// deltas.
type statCache[T any] struct {
	mu    sync.Mutex
	data  map[string]T
	merge func(prev, next T) T
}

// newStatCache returns an empty statCache that folds new samples into
// existing entries with merge (prev is the zero value for unseen keys).
func newStatCache[T any](merge func(prev, next T) T) *statCache[T] {
	return &statCache[T]{
		data:  make(map[string]T),
		merge: merge,
	}
}

// Update folds a new sample for key into the cache.
func (c *statCache[T]) Update(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = c.merge(c.data[key], value)
}

// Values returns a snapshot of the accumulated entries.
func (c *statCache[T]) Values() []T {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Collect(maps.Values(c.data))
}
