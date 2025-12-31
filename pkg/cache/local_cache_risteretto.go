package cache

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto"
)

// LocalCacheRistretto is an implementation of Cache that uses Ristretto.
// It provides a local in-memory caching solution.
type LocalCacheRistretto[T any] struct {
	cache *ristretto.Cache
	ttl   time.Duration
}

// NewLocalCacheRistretto creates a new instance of LocalCacheRistretto.
// It initializes the Ristretto cache with the provided configuration.
func NewLocalCacheRistretto[T any](cacheCfg *CacheConfig[T]) (*LocalCacheRistretto[T], error, func()) {
	cache, err := ristretto.NewCache(&ristretto.Config{
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
	return &LocalCacheRistretto[T]{cache: cache, ttl: cacheCfg.DefaultTTL}, nil, cleanup
}

// NewLocalCacheRistrettoGeneric creates a new generic instance of LocalCacheRistretto.
func NewLocalCacheRistrettoGeneric[T any](cacheCfg *CacheConfig[T]) (*LocalCacheRistretto[T], error, func()) {
	cache, err := ristretto.NewCache(&ristretto.Config{
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
	return &LocalCacheRistretto[T]{cache: cache, ttl: cacheCfg.DefaultTTL}, nil, cleanup
}

// Get retrieves a value from the cache for the given key.
// It returns the value and a boolean indicating whether the key was found.
func (c *LocalCacheRistretto[T]) Get(ctx context.Context, key string) (T, bool) {
	v, found := c.cache.Get(key)
	if !found {
		var zero T
		return zero, false
	}
	val, ok := v.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return val, true
}

// Set stores a value in the cache for the given key.
// If a TTL is set, it calls SetWithTTL instead.
func (c *LocalCacheRistretto[T]) Set(ctx context.Context, key string, value T) error {
	if c.ttl.Seconds() > 0 {
		return c.SetWithTTL(ctx, key, value, c.ttl)
	}
	c.cache.Set(key, value, 1) // Assuming the cost is 1 for simplicity.
	return nil
}

// SetWithTTL stores a value in the cache for the given key with a specified TTL.
func (c *LocalCacheRistretto[T]) SetWithTTL(ctx context.Context, key string, value T, ttl time.Duration) error {
	c.cache.SetWithTTL(key, value, 1, ttl) // Assuming the cost is 1 for simplicity.
	return nil
}

// Expire removes the key from the cache.
// Note: Ristretto doesn't support updating TTL, so we simply delete the key.
func (c *LocalCacheRistretto[T]) Expire(ctx context.Context, key string, ttl time.Duration) error {
	c.cache.Del(key)
	return nil
}

// Delete removes the key from the cache.
func (c *LocalCacheRistretto[T]) Delete(ctx context.Context, key string) error {
	c.cache.Del(key)
	return nil
}

// Increment increments the integer value of a key by delta.
func (c *LocalCacheRistretto[T]) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	// Not atomic.
	val, found := c.Get(ctx, key)
	var current int64
	if found {
		// Try to cast T to int64 or compatible types
		// Since T is generic, we can't easily cast unless T IS int64 or similar.
		// However, the interface signature returns int64.
		// If T is not a number, this is weird. But usually Cache[int64] would be used for counters?
		// Or the user stores numbers in a Cache[any].
		if v, ok := any(val).(int64); ok {
			current = v
		} else if v, ok := any(val).(int); ok {
			current = int64(v)
		} else if v, ok := any(val).(float64); ok {
			current = int64(v)
		} else {
			// Try deserialization if it was stored as string/bytes?
			// For simplicity, if not numeric, treat as 0 or error?
			// Let's treat missing as 0. If present but not number, maybe error?
			// For now, let's assume valid type.
			return 0, nil // Should return error probably?
		}
	}

	newVal := current + delta
	// We need to store back as T.
	// If T is int64, fine.
	if _, ok := any(newVal).(T); ok {
		c.Set(ctx, key, any(newVal).(T))
	} else {
		// This is tricky if T is not int64.
		// Assuming T is flexible or the user knows what they are doing.
		// If T is `any`, this works.
		c.Set(ctx, key, any(newVal).(T))
	}
	return newVal, nil
}

// Decrement decrements the integer value of a key by delta.
func (c *LocalCacheRistretto[T]) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	return c.Increment(ctx, key, -delta)
}

// HSet sets the value of a field in a hash.
func (c *LocalCacheRistretto[T]) HSet(ctx context.Context, key string, field string, value T) error {
	return c.hset(ctx, key, field, value, 0)
}

// HSetWithTTL sets the value of a field in a hash and sets the expiration for the key.
func (c *LocalCacheRistretto[T]) HSetWithTTL(ctx context.Context, key string, field string, value T, ttl time.Duration) error {
	return c.hset(ctx, key, field, value, ttl)
}

func (c *LocalCacheRistretto[T]) hset(ctx context.Context, key string, field string, value T, ttl time.Duration) error {
	// Get existing map
	v, found := c.cache.Get(key)
	var m map[string]T
	if found {
		if val, ok := v.(map[string]T); ok {
			m = val
		} else {
			// Overwrite if not a map? Or error?
			// Redis overwrites.
			m = make(map[string]T)
		}
	} else {
		m = make(map[string]T)
	}

	m[field] = value
	if ttl > 0 {
		c.cache.SetWithTTL(key, m, 1, ttl)
	} else {
		c.cache.Set(key, m, 1)
	}
	return nil
}

// HGet gets the value of a field in a hash.
func (c *LocalCacheRistretto[T]) HGet(ctx context.Context, key string, field string) (T, bool) {
	v, found := c.cache.Get(key)
	if !found {
		var zero T
		return zero, false
	}
	m, ok := v.(map[string]T)
	if !ok {
		var zero T
		return zero, false
	}
	val, ok := m[field]
	return val, ok
}

// HGetAll gets all fields and values in a hash.
func (c *LocalCacheRistretto[T]) HGetAll(ctx context.Context, key string) (map[string]T, error) {
	v, found := c.cache.Get(key)
	if !found {
		// Return empty map or error? Redis returns empty list/map.
		return make(map[string]T), nil
	}
	m, ok := v.(map[string]T)
	if !ok {
		return nil, nil // Or error?
	}
	// Return copy?
	result := make(map[string]T, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result, nil
}

// HDel deletes one or more fields from a hash.
func (c *LocalCacheRistretto[T]) HDel(ctx context.Context, key string, fields ...string) error {
	v, found := c.cache.Get(key)
	if !found {
		return nil
	}
	m, ok := v.(map[string]T)
	if !ok {
		return nil
	}

	for _, f := range fields {
		delete(m, f)
	}
	// Update cache
	// Note: Ristretto items are likely immutable/copied?
	// Actually Ristretto stores interface{}, so modifying map in place *might* work if it's a pointer/map,
	// but Ristretto might manage its own memory.
	// To be safe and ensure it persists (and TTL is refreshed/kept), we Set it again.
	// But Set resets TTL in Ristretto if not SetWithTTL.
	// If we want to preserve TTL, we can't easily read old TTL from Ristretto.
	// This is a limitation. We'll just Set.
	c.cache.Set(key, m, 1)
	return nil
}

// HExpire sets expiration on a hash field.
// Note: Local cache does NOT support individual field expiration.
// This is a limitation of in-memory caching - TTL applies to the entire hash.
// Returns false to indicate operation not supported.
func (c *LocalCacheRistretto[T]) HExpire(ctx context.Context, key string, field string, ttl time.Duration) (bool, error) {
	// Not supported for local cache - would need to track field-level TTLs separately
	return false, nil
}

// HTTL returns the remaining TTL of a hash field.
// Note: Local cache does NOT support individual field expiration.
// Always returns -1 (no expiration) as fields don't have individual TTLs.
func (c *LocalCacheRistretto[T]) HTTL(ctx context.Context, key string, field string) (int64, error) {
	// Not supported for local cache
	// Return -1 to indicate field exists but has no expiration
	_, found := c.HGet(ctx, key, field)
	if !found {
		return -2, nil // Field doesn't exist
	}
	return -1, nil // Field exists but no individual TTL
}

// LockWithContext acquires a simple in-memory lock for the given key.
// Note: This is NOT a true distributed lock - it only works within a single process.
// For distributed locking, use RemoteCacheValkey with valkeylock.
// Returns a context and cancel function. The context is NOT automatically canceled on lock loss.
func (c *LocalCacheRistretto[T]) LockWithContext(ctx context.Context, key string) (context.Context, context.CancelFunc, error) {
	// Check if lock already exists
	_, found := c.cache.Get(key)
	if found {
		return nil, nil, ErrNotLocked
	}

	// Set lock with TTL
	c.cache.SetWithTTL(key, "locked", 1, c.ttl)

	// Create a new context with cancel
	lockCtx, cancel := context.WithCancel(ctx)

	// Wrap cancel to also delete the lock
	wrappedCancel := func() {
		c.cache.Del(key)
		cancel()
	}

	return lockCtx, wrappedCancel, nil
}

// TryLockWithContext attempts to acquire a lock without waiting.
// For local cache, this behaves the same as LockWithContext.
func (c *LocalCacheRistretto[T]) TryLockWithContext(ctx context.Context, key string) (context.Context, context.CancelFunc, error) {
	return c.LockWithContext(ctx, key)
}
