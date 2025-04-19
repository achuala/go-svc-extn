package cache

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/valkey-io/valkey-go"
)

var (
	vkClientOnce sync.Once
	vkClient     valkey.Client
	vkClientErr  error
)

// RemoteCacheValkey is an implementation of Cache that uses Valkey as a remote cache.
type RemoteCacheValkey struct {
	name        string        // Name of the cache, used as a prefix for keys
	ttl         time.Duration // Default time-to-live for cache entries
	maxElements uint64        // Maximum number of elements allowed in the cache
	applyTouch  bool          // Whether to extend TTL on cache hits
}

// NewRemoteCacheValkey creates a new instance of RemoteCacheValkey.
// It initializes the Valkey client with the provided configuration.
func NewRemoteCacheValkey(cacheCfg *CacheConfig) (*RemoteCacheValkey, error, func()) {
	if !strings.HasPrefix(cacheCfg.RemoteCacheAddr, "redis://") {
		cacheCfg.RemoteCacheAddr = "redis://" + cacheCfg.RemoteCacheAddr
	}
	if cacheCfg.Mode == "cluster" {
		vkClientOnce.Do(func() {
			// Connect to a cluster "redis://127.0.0.1:7001?addr=127.0.0.1:7002&addr=127.0.0.1:7003"
			clusterClientOptions := valkey.MustParseURL(cacheCfg.RemoteCacheAddr)
			clusterClientOptions.ShuffleInit = true
			clusterClientOptions.SendToReplicas = func(cmd valkey.Completed) bool {
				return cmd.IsReadOnly()
			}
			vkClient, vkClientErr = valkey.NewClient(clusterClientOptions)
		})
	} else if cacheCfg.Mode == "sentinel" {
		vkClientOnce.Do(func() {
			// // connect to a valkey sentinel redis://127.0.0.1:26379/0?master_set=my_master"
			vkClient, vkClientErr = valkey.NewClient(valkey.MustParseURL(cacheCfg.RemoteCacheAddr))
		})
	} else {
		// Standalone mode
		clientOptions := valkey.MustParseURL(cacheCfg.RemoteCacheAddr)
		clientOptions.SendToReplicas = func(cmd valkey.Completed) bool {
			return cmd.IsReadOnly()
		}
		vkClientOnce.Do(func() {
			vkClient, vkClientErr = valkey.NewClient(clientOptions)
		})
	}

	if vkClientErr != nil {
		return nil, vkClientErr, nil
	}

	cleanup := func() {
		// No need to close the client here as it's shared
	}

	return &RemoteCacheValkey{
		name:        cacheCfg.CacheName,
		ttl:         cacheCfg.DefaultTTL,
		maxElements: cacheCfg.MaxElements,
		applyTouch:  cacheCfg.ApplyTouch,
	}, nil, cleanup
}

// makeKey creates a composite key by prefixing the provided key with the cache name.
func (c *RemoteCacheValkey) makeKey(key string) string {
	return c.name + ":" + key
}

// Get retrieves a value from the cache for the given key.
// It returns the value and a boolean indicating whether the key was found.
func (c *RemoteCacheValkey) Get(ctx context.Context, key string) (string, bool) {
	compositeKey := c.makeKey(key)
	var cmd valkey.Completed
	if !c.applyTouch {
		cmd = vkClient.B().Get().Key(compositeKey).Build()
	} else {
		cmd = vkClient.B().Getex().Key(compositeKey).Ex(c.ttl).Build()
	}
	val, err := vkClient.Do(ctx, cmd).ToString()
	if err != nil {
		return "", false
	}
	return val, val != ""
}

// Set stores a value in the cache for the given key.
// If a TTL is set, it calls SetWithTTL instead.
func (c *RemoteCacheValkey) Set(ctx context.Context, key string, value string) error {
	if c.ttl.Seconds() > 0 {
		return c.SetWithTTL(ctx, key, value, c.ttl)
	}
	cmd := vkClient.B().Set().Key(c.makeKey(key)).Value(value).Build()
	return vkClient.Do(ctx, cmd).Error()
}

// SetWithTTL stores a value in the cache for the given key with a specified TTL.
func (c *RemoteCacheValkey) SetWithTTL(ctx context.Context, key string, value string, ttl time.Duration) error {
	cmd := vkClient.B().Set().Key(c.makeKey(key)).Value(value).Ex(ttl).Build()
	return vkClient.Do(ctx, cmd).Error()
}

// Expire sets the expiration time for the given key.
func (c *RemoteCacheValkey) Expire(ctx context.Context, key string, ttl time.Duration) error {
	cmd := vkClient.B().Expire().Key(c.makeKey(key)).Seconds(int64(ttl.Seconds())).Build()
	return vkClient.Do(ctx, cmd).Error()
}

// Delete removes the key from the cache.
func (c *RemoteCacheValkey) Delete(ctx context.Context, key string) error {
	cmd := vkClient.B().Del().Key(c.makeKey(key)).Build()
	return vkClient.Do(ctx, cmd).Error()
}
