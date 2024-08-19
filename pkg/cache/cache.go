package cache

import (
	"context"
	"time"
)

// Cache is the interface that defines the caching operations.
type Cache interface {
	Get(ctx context.Context, key string) (any, bool)
	Set(ctx context.Context, key string, value any) error
	Delete(ctx context.Context, key string) error
	Expire(ctx context.Context, key string, ttl time.Duration) error
	SetWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error
}

type CacheConfig struct {
	// local/remote, default is local
	Mode            string
	CacheName       string
	RemoteCacheAddr string
	// Default time to live for the key. See also ApplyTouch
	DefaultTTL  time.Duration
	MaxElements uint64
	// Set this to true in order to extend the TTL of the key
	ApplyTouch bool
}

func NewCache(cacheCfg *CacheConfig) Cache {
	if cacheCfg.Mode == "remote" {
		return NewRemoteCacheValkey(cacheCfg)
	}
	return NewLocalCacheRistretto(cacheCfg)
}
