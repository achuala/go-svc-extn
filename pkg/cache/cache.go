package cache

import (
	"context"
	"time"
)

// Cache is the interface that defines the caching operations.
type Cache interface {
	Get(ctx context.Context, key string) (interface{}, bool)
	Set(ctx context.Context, key string, value interface{}) error
	SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

type CacheConfig struct {
	// local/remote, default is local
	Mode            string
	RemoteCacheAddr string
	DefaultTTL      time.Duration
	MaxElements     uint32
}

func NewCache(cacheCfg *CacheConfig) Cache {
	if cacheCfg.Mode == "remote" {
		return NewRemoteCacheValkey(cacheCfg.RemoteCacheAddr)
	}
	return NewLocalCacheRistretto()
}
