package cache

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto"
)

// LocalCacheRistretto is an implementation of Cache that uses Ristretto.
// It provides a local in-memory caching solution.
type LocalCacheRistretto struct {
	cache *ristretto.Cache[string, string]
	ttl   time.Duration
}

// NewLocalCacheRistretto creates a new instance of LocalCacheRistretto.
// It initializes the Ristretto cache with the provided configuration.
func NewLocalCacheRistretto(cacheCfg *CacheConfig) (*LocalCacheRistretto, error, func()) {
	cache, err := ristretto.NewCache[string, string](&ristretto.Config[string, string]{
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
	return &LocalCacheRistretto{cache: cache, ttl: cacheCfg.DefaultTTL}, nil, cleanup
}

// Get retrieves a value from the cache for the given key.
// It returns the value and a boolean indicating whether the key was found.
func (c *LocalCacheRistretto) Get(ctx context.Context, key string) (string, bool) {
	v, found := c.cache.Get(key)
	if !found {
		return "", false
	}
	return v, true
}

// Set stores a value in the cache for the given key.
// If a TTL is set, it calls SetWithTTL instead.
func (c *LocalCacheRistretto) Set(ctx context.Context, key string, value string) error {
	if c.ttl.Seconds() > 0 {
		return c.SetWithTTL(ctx, key, value, c.ttl)
	}
	c.cache.Set(key, value, 1) // Assuming the cost is 1 for simplicity.
	return nil
}

// SetWithTTL stores a value in the cache for the given key with a specified TTL.
func (c *LocalCacheRistretto) SetWithTTL(ctx context.Context, key string, value string, ttl time.Duration) error {
	c.cache.SetWithTTL(key, value, 1, ttl) // Assuming the cost is 1 for simplicity.
	return nil
}

// Expire removes the key from the cache.
// Note: Ristretto doesn't support updating TTL, so we simply delete the key.
func (c *LocalCacheRistretto) Expire(ctx context.Context, key string, ttl time.Duration) error {
	c.cache.Del(key)
	return nil
}

// Delete removes the key from the cache.
func (c *LocalCacheRistretto) Delete(ctx context.Context, key string) error {
	c.cache.Del(key)
	return nil
}
