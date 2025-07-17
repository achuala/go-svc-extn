package cache

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto"
)

// LocalCacheRistretto is an implementation of Cache that uses Ristretto.
// It provides a local in-memory caching solution.
type LocalCacheRistretto[T any] struct {
	cache *ristretto.Cache
	ttl   time.Duration
}

// NewLocalCacheRistretto creates a new instance of LocalCacheRistretto.
// It initializes the Ristretto cache with the provided configuration.
func NewLocalCacheRistretto[T any](cacheCfg *CacheConfig[T]) (*LocalCacheRistretto[T], error, func()) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // Number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // Maximum cost of cache (1GB).
		BufferItems: 64,      // Number of keys per Get buffer.
	})
	if err != nil {
		return nil, err, nil
	}
	cleanup := func() {
		cache.Close()
	}
	return &LocalCacheRistretto[T]{cache: cache, ttl: cacheCfg.DefaultTTL}, nil, cleanup
}

// NewLocalCacheRistrettoGeneric creates a new generic instance of LocalCacheRistretto.
func NewLocalCacheRistrettoGeneric[T any](cacheCfg *CacheConfig[T]) (*LocalCacheRistretto[T], error, func()) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // Number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // Maximum cost of cache (1GB).
		BufferItems: 64,      // Number of keys per Get buffer.
	})
	if err != nil {
		return nil, err, nil
	}
	cleanup := func() {
		cache.Close()
	}
	return &LocalCacheRistretto[T]{cache: cache, ttl: cacheCfg.DefaultTTL}, nil, cleanup
}

// Get retrieves a value from the cache for the given key.
// It returns the value and a boolean indicating whether the key was found.
func (c *LocalCacheRistretto[T]) Get(ctx context.Context, key string) (T, bool) {
	v, found := c.cache.Get(key)
	if !found {
		var zero T
		return zero, false
	}
	val, ok := v.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return val, true
}

// Set stores a value in the cache for the given key.
// If a TTL is set, it calls SetWithTTL instead.
func (c *LocalCacheRistretto[T]) Set(ctx context.Context, key string, value T) error {
	if c.ttl.Seconds() > 0 {
		return c.SetWithTTL(ctx, key, value, c.ttl)
	}
	c.cache.Set(key, value, 1) // Assuming the cost is 1 for simplicity.
	return nil
}

// SetWithTTL stores a value in the cache for the given key with a specified TTL.
func (c *LocalCacheRistretto[T]) SetWithTTL(ctx context.Context, key string, value T, ttl time.Duration) error {
	c.cache.SetWithTTL(key, value, 1, ttl) // Assuming the cost is 1 for simplicity.
	return nil
}

// Expire removes the key from the cache.
// Note: Ristretto doesn't support updating TTL, so we simply delete the key.
func (c *LocalCacheRistretto[T]) Expire(ctx context.Context, key string, ttl time.Duration) error {
	c.cache.Del(key)
	return nil
}

// Delete removes the key from the cache.
func (c *LocalCacheRistretto[T]) Delete(ctx context.Context, key string) error {
	c.cache.Del(key)
	return nil
}
