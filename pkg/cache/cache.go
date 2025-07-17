package cache

import (
	"context"
	"encoding/json"
	"time"

	"google.golang.org/protobuf/proto"
)

// Cache is the interface that defines the caching operations.
type Cache[T any] interface {
	// Returns the value for the given key.
	// If the key is not found, it returns nil and false.
	Get(ctx context.Context, key string) (T, bool)
	// Sets the value for the given key.
	// If the key already exists, it returns an error.
	Set(ctx context.Context, key string, value T) error
	// Deletes the value for the given key.
	// If the key is not found, it returns an error.
	Delete(ctx context.Context, key string) error
	// Sets the expiration time for the given key.
	Expire(ctx context.Context, key string, ttl time.Duration) error
	// Sets the value for the given key with a specific TTL.
	SetWithTTL(ctx context.Context, key string, value T, ttl time.Duration) error
}

// CacheConfig is the configuration for the cache.
type CacheConfig[T any] struct {
	// local/remote/cluster, default is local
	Mode            string
	CacheName       string
	RemoteCacheAddr string
	// Default time to live for the key. See also ApplyTouch
	DefaultTTL  time.Duration
	MaxElements uint64
	// Set this to true in order to extend the TTL of the key
	ApplyTouch bool
	SerDe      SerDe[T]
}

type SerDe[T any] interface {
	Serialize(value T) ([]byte, error)
	Deserialize(data []byte) (T, error)
}

type JSONSerDe[T any] struct{}

func (s *JSONSerDe[T]) Serialize(value T) ([]byte, error) {
	return json.Marshal(value)
}

func (s *JSONSerDe[T]) Deserialize(data []byte) (T, error) {
	var value T
	return value, json.Unmarshal(data, &value)
}

func NewJSONSerDe[T any]() *JSONSerDe[T] {
	return &JSONSerDe[T]{}
}

type ProtoSerDe[T proto.Message] struct{}

func (s *ProtoSerDe[T]) Serialize(value T) ([]byte, error) {
	return proto.Marshal(value)
}

func (s *ProtoSerDe[T]) Deserialize(data []byte) (T, error) {
	var value T
	err := proto.Unmarshal(data, value)
	return value, err
}

// NewCache creates a new cache instance based on the provided configuration.
func NewCache[T any](cacheCfg *CacheConfig[T]) (Cache[T], error, func()) {
	if cacheCfg.Mode == "local" {
		return NewLocalCacheRistretto(cacheCfg)
	}
	return NewRemoteCacheValkey(cacheCfg)
}
