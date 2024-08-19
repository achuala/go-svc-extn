package cache

import (
	"context"
	"log"
	"time"

	"github.com/dgraph-io/ristretto"
)

// LocalCache is an implementation of Cache that uses Ristretto.
type LocalCacheRistretto struct {
	cache *ristretto.Cache
}

func NewLocalCacheRistretto(cacheCfg *CacheConfig) *LocalCacheRistretto {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // Number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // Maximum cost of cache (1GB).
		BufferItems: 64,      // Number of keys per Get buffer.
	})
	if err != nil {
		log.Fatalf("failed to create local cache: %v", err)
	}
	return &LocalCacheRistretto{cache: cache}
}

func (c *LocalCacheRistretto) Get(ctx context.Context, key string) (any, bool) {
	return c.cache.Get(key)
}

func (c *LocalCacheRistretto) Set(ctx context.Context, key string, value any) error {
	c.cache.Set(key, value, 1) // Assuming the cost is 1 for simplicity.
	return nil
}

func (c *LocalCacheRistretto) SetWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error {
	c.cache.SetWithTTL(key, value, 1, ttl) // Assuming the cost is 1 for simplicity.
	return nil
}

func (c *LocalCacheRistretto) Expire(ctx context.Context, key string, ttl time.Duration) error {
	c.cache.Del(key)
	return nil
}

func (c *LocalCacheRistretto) Delete(ctx context.Context, key string) error {
	c.cache.Del(key)
	return nil
}
