package utils

import (
	"fmt"
	"github.com/1f349/cache"
	"golang.org/x/sync/singleflight"
	"time"
)

// CacheFlight wraps a cache with single flight protection
type CacheFlight[K Key, V any] struct {
	store *cache.Cache[K, V]
	group singleflight.Group
}

type Key interface {
	fmt.Stringer
	comparable
}

func NewCacheFlight[K Key, V any]() *CacheFlight[K, V] {
	return &CacheFlight[K, V]{store: cache.New[K, V]()}
}

// LoadOrStore tries to load from the cache first, then uses a single flight protected database fetch
func (c *CacheFlight[K, V]) LoadOrStore(key K, ttl time.Duration, loader func() (V, error)) (V, error) {
	// Try cache
	if val, ok := c.store.Get(key); ok {
		return val, nil
	}

	// Deduplicated fetch
	v, err, _ := c.group.Do(key.String(), func() (any, error) {
		val, err := loader()
		if err != nil {
			var zero V
			return zero, err
		}
		c.store.Set(key, val, ttl)
		return val, nil
	})

	return v.(V), err
}

type StringKey string

func (k StringKey) String() string { return string(k) }
