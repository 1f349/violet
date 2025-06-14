package utils

import (
	"context"
	"fmt"
	"github.com/1f349/cache"
	"github.com/1f349/violet/logger"
	"golang.org/x/sync/singleflight"
	"time"
)

// CacheFlight wraps a cache with single flight protection
type CacheFlight[K Key, V any] struct {
	group  singleflight.Group
	store  *cache.Cache[K, V]
	ttl    time.Duration
	getter func(ctx context.Context, k K) (V, error)
}

type Key interface {
	fmt.Stringer
	comparable
}

func NewCacheFlight[K Key, V any](ttl time.Duration, getter func(ctx context.Context, k K) (V, error)) *CacheFlight[K, V] {
	return &CacheFlight[K, V]{
		store:  cache.New[K, V](),
		ttl:    ttl,
		getter: getter,
	}
}

// LoadOrStore tries to load from the cache first, then uses a single flight protected database fetch
func (c *CacheFlight[K, V]) LoadOrStore(ctx context.Context, key K) (V, error) {
	// Try cache
	if val, ok := c.store.Get(key); ok {
		logger.Logger.Debug("Loading domain from cache", "domain", key)
		return val, nil
	}

	// Deduplicated fetch
	v, err, _ := c.group.Do(key.String(), func() (any, error) {
		logger.Logger.Debug("Loading domain from database", "domain", key)
		val, err := c.getter(ctx, key)
		if err != nil {
			var zero V
			return zero, err
		}
		logger.Logger.Debug("Storing domain in cache", "domain", key)
		c.store.Set(key, val, c.ttl)
		return val, nil
	})

	return v.(V), err
}

type StringKey string

func (k StringKey) String() string { return string(k) }
