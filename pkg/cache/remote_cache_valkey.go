package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valkey-io/valkey-go"
)

// DefaultSerDe provides default serialization/deserialization for various types
type DefaultSerDe[T any] struct{}

func (s *DefaultSerDe[T]) Serialize(value T) ([]byte, error) {
	// Handle different types appropriately
	switch v := any(value).(type) {
	case string:
		return []byte(v), nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return []byte(fmt.Sprintf("%v", v)), nil
	case float32, float64:
		return []byte(fmt.Sprintf("%v", v)), nil
	case bool:
		if v {
			return []byte("true"), nil
		}
		return []byte("false"), nil
	default:
		// For complex types, use JSON as fallback
		return json.Marshal(value)
	}
}

func (s *DefaultSerDe[T]) Deserialize(data []byte) (T, error) {
	// Try to convert back to the original type
	var zero T
	zeroType := reflect.TypeOf(zero)

	switch zeroType.Kind() {
	case reflect.String:
		return any(string(data)).(T), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val, err := strconv.ParseInt(string(data), 10, 64); err == nil {
			return any(val).(T), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val, err := strconv.ParseUint(string(data), 10, 64); err == nil {
			return any(val).(T), nil
		}
	case reflect.Float32, reflect.Float64:
		if val, err := strconv.ParseFloat(string(data), 64); err == nil {
			return any(val).(T), nil
		}
	case reflect.Bool:
		if val, err := strconv.ParseBool(string(data)); err == nil {
			return any(val).(T), nil
		}
	}

	// Fallback to JSON for complex types
	return zero, json.Unmarshal(data, &zero)
}

var (
	vkClientOnce sync.Once
	vkClient     valkey.Client
	vkClientErr  error
)

// RemoteCacheValkey is an implementation of Cache that uses Valkey as a remote cache.
type RemoteCacheValkey[T any] struct {
	name        string        // Name of the cache, used as a prefix for keys
	ttl         time.Duration // Default time-to-live for cache entries
	maxElements uint64        // Maximum number of elements allowed in the cache
	applyTouch  bool          // Whether to extend TTL on cache hits
	serDe       SerDe[T]
}

// NewRemoteCacheValkey creates a new instance of RemoteCacheValkey.
// It initializes the Valkey client with the provided configuration.
func NewRemoteCacheValkey[T any](cacheCfg *CacheConfig[T]) (*RemoteCacheValkey[T], error, func()) {
	if !strings.HasPrefix(cacheCfg.RemoteCacheAddr, "redis://") {
		cacheCfg.RemoteCacheAddr = "redis://" + cacheCfg.RemoteCacheAddr
	}
	switch cacheCfg.Mode {
	case "cluster":
		vkClientOnce.Do(func() {
			// Connect to a cluster "redis://127.0.0.1:7001?addr=127.0.0.1:7002&addr=127.0.0.1:7003"
			clusterClientOptions := valkey.MustParseURL(cacheCfg.RemoteCacheAddr)
			clusterClientOptions.ShuffleInit = true
			clusterClientOptions.SendToReplicas = func(cmd valkey.Completed) bool {
				return cmd.IsReadOnly()
			}
			vkClient, vkClientErr = valkey.NewClient(clusterClientOptions)
		})
	case "sentinel":
		vkClientOnce.Do(func() {
			// // connect to a valkey sentinel redis://127.0.0.1:26379/0?master_set=my_master"
			vkClient, vkClientErr = valkey.NewClient(valkey.MustParseURL(cacheCfg.RemoteCacheAddr))
		})
	default:
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
	// Set default SerDe if not provided
	if cacheCfg.SerDe == nil {
		cacheCfg.SerDe = &DefaultSerDe[T]{}
	}

	return &RemoteCacheValkey[T]{
		name:        cacheCfg.CacheName,
		ttl:         cacheCfg.DefaultTTL,
		maxElements: cacheCfg.MaxElements,
		applyTouch:  cacheCfg.ApplyTouch,
		serDe:       cacheCfg.SerDe,
	}, nil, cleanup
}

// makeKey creates a composite key by prefixing the provided key with the cache name.
func (c *RemoteCacheValkey[T]) makeKey(key string) string {
	return c.name + ":" + key
}

// Get retrieves a value from the cache for the given key.
// It returns the value and a boolean indicating whether the key was found.
func (c *RemoteCacheValkey[T]) Get(ctx context.Context, key string) (T, bool) {
	compositeKey := c.makeKey(key)
	var cmd valkey.Completed
	if !c.applyTouch {
		cmd = vkClient.B().Get().Key(compositeKey).Build()
	} else {
		cmd = vkClient.B().Getex().Key(compositeKey).Ex(c.ttl).Build()
	}

	val, err := vkClient.Do(ctx, cmd).ToString()
	if err != nil {
		var zero T
		return zero, false
	}

	if val == "" {
		var zero T
		return zero, false
	}

	// Use the SerDe to deserialize
	result, err := c.serDe.Deserialize([]byte(val))
	if err != nil {
		var zero T
		return zero, false
	}

	return result, true
}

// Set stores a value in the cache for the given key.
// If a TTL is set, it calls SetWithTTL instead.
func (c *RemoteCacheValkey[T]) Set(ctx context.Context, key string, value T) error {
	if c.ttl.Seconds() > 0 {
		return c.SetWithTTL(ctx, key, value, c.ttl)
	}

	// Use the SerDe to serialize
	data, err := c.serDe.Serialize(value)
	if err != nil {
		return err
	}

	cmd := vkClient.B().Set().Key(c.makeKey(key)).Value(string(data)).Build()
	return vkClient.Do(ctx, cmd).Error()
}

// SetWithTTL stores a value in the cache for the given key with a specified TTL.
func (c *RemoteCacheValkey[T]) SetWithTTL(ctx context.Context, key string, value T, ttl time.Duration) error {
	// Use the SerDe to serialize
	data, err := c.serDe.Serialize(value)
	if err != nil {
		return err
	}

	cmd := vkClient.B().Set().Key(c.makeKey(key)).Value(string(data)).Ex(ttl).Build()
	return vkClient.Do(ctx, cmd).Error()
}

// Expire sets the expiration time for the given key.
func (c *RemoteCacheValkey[T]) Expire(ctx context.Context, key string, ttl time.Duration) error {
	cmd := vkClient.B().Expire().Key(c.makeKey(key)).Seconds(int64(ttl.Seconds())).Build()
	return vkClient.Do(ctx, cmd).Error()
}

// Delete removes the key from the cache.
func (c *RemoteCacheValkey[T]) Delete(ctx context.Context, key string) error {
	cmd := vkClient.B().Del().Key(c.makeKey(key)).Build()
	return vkClient.Do(ctx, cmd).Error()
}
