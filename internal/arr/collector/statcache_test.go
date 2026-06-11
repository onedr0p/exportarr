package collector

import (
	"slices"
	"sync"
	"testing"

	"github.com/onedr0p/exportarr/internal/assert"
)

func TestStatCache_UpdateMergesPerKey(t *testing.T) {
	cache := newStatCache(func(prev, next int) int { return prev + next })

	cache.Update("a", 1)
	cache.Update("a", 2)
	cache.Update("b", 5)

	got := cache.Values()
	slices.Sort(got)
	assert.DeepEqual(t, got, []int{3, 5})
}

func TestStatCache_MergeSeesZeroValueForNewKeys(t *testing.T) {
	var prevs []string
	cache := newStatCache(func(prev, next string) string {
		prevs = append(prevs, prev)
		return next
	})

	cache.Update("k", "first")
	cache.Update("k", "second")

	assert.DeepEqual(t, prevs, []string{"", "first"})
}

func TestStatCache_ValuesIsASnapshot(t *testing.T) {
	cache := newStatCache(func(_, next int) int { return next })
	cache.Update("a", 1)

	got := cache.Values()
	cache.Update("a", 2)

	assert.DeepEqual(t, got, []int{1})
}

func TestStatCache_ConcurrentUpdates(t *testing.T) {
	cache := newStatCache(func(prev, next int) int { return prev + next })

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 100 {
				cache.Update("hits", 1)
			}
		})
	}
	wg.Wait()

	assert.DeepEqual(t, cache.Values(), []int{800})
}
