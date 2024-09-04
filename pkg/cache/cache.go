package cache

import (
	"context"
	"time"
)

// Cache is the interface that defines the caching operations.
type Cache interface {
	// Returns the value for the given key.
	// If the key is not found, it returns nil and false.
	Get(ctx context.Context, key string) (any, bool)
	// Sets the value for the given key.
	// If the key already exists, it returns an error.
	Set(ctx context.Context, key string, value any) error
	// Deletes the value for the given key.
	// If the key is not found, it returns an error.
	Delete(ctx context.Context, key string) error
	// Sets the expiration time for the given key.
	Expire(ctx context.Context, key string, ttl time.Duration) error
	// Sets the value for the given key with a specific TTL.
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

func NewCache(cacheCfg *CacheConfig) (Cache, func()) {
	if cacheCfg.Mode == "remote" {
		return NewRemoteCacheValkey(cacheCfg)
	}
	return NewLocalCacheRistretto(cacheCfg)
}
