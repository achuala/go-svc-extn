package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RemoteCache is an implementation of Cache that uses Redis.
type RemoteCacheValkey struct {
	client      *redis.Client
	name        string
	ttl         time.Duration
	maxElements uint64
	applyTouch  bool
}

func NewRemoteCacheValkey(cacheCfg *CacheConfig) (*RemoteCacheValkey, func()) {
	client := redis.NewClient(&redis.Options{
		Addr:     cacheCfg.RemoteCacheAddr,
		Password: "", // No password set
		DB:       0,  // Use default DB
	})
	cleanup := func() {
		if err := client.Close(); err != nil {
			// Consider logging this error
			fmt.Printf("Error closing Redis client: %v\n", err)
		}
	}

	return &RemoteCacheValkey{client: client, name: cacheCfg.CacheName,
		ttl: cacheCfg.DefaultTTL, maxElements: cacheCfg.MaxElements, applyTouch: cacheCfg.ApplyTouch}, cleanup
}

func (c *RemoteCacheValkey) makeKey(key string) string {
	return c.name + ":" + key
}
func (c *RemoteCacheValkey) Get(ctx context.Context, key string) (interface{}, bool) {
	val, err := c.client.Get(ctx, c.makeKey(key)).Result()
	if err == redis.Nil {
		return nil, false
	} else if err != nil {
		return nil, false
	}
	if c.applyTouch {
		c.Expire(ctx, key, c.ttl)
	}
	return val, true
}

func (c *RemoteCacheValkey) Set(ctx context.Context, key string, value interface{}) error {
	if c.ttl.Seconds() > 0 {
		return c.SetWithTTL(ctx, key, value, c.ttl)
	}
	return c.client.Set(ctx, c.makeKey(key), value, 0).Err()
}

func (c *RemoteCacheValkey) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.client.Set(ctx, c.makeKey(key), value, ttl).Err()
}

func (c *RemoteCacheValkey) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, c.makeKey(key), ttl).Err()
}

func (c *RemoteCacheValkey) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, c.makeKey(key)).Err()
}
