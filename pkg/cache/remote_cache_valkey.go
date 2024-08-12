package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RemoteCache is an implementation of Cache that uses Redis.
type RemoteCacheValkey struct {
	client *redis.Client
}

func NewRemoteCacheValkey(addr string) *RemoteCacheValkey {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // No password set
		DB:       0,  // Use default DB
	})
	return &RemoteCacheValkey{client: client}
}

func (c *RemoteCacheValkey) Get(ctx context.Context, key string) (interface{}, bool) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, false
	} else if err != nil {
		return nil, false
	}
	return val, true
}

func (c *RemoteCacheValkey) Set(ctx context.Context, key string, value interface{}) error {
	return c.client.Set(ctx, key, value, 0).Err()
}

func (c *RemoteCacheValkey) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}
