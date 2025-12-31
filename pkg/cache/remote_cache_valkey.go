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
	"github.com/valkey-io/valkey-go/valkeylock"
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
	locker      valkeylock.Locker // Distributed lock manager
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

	// Set default SerDe if not provided
	if cacheCfg.SerDe == nil {
		cacheCfg.SerDe = &DefaultSerDe[T]{}
	}

	// Initialize distributed lock manager
	// Set KeyMajority based on mode
	keyMajority := int32(1) // Default for standalone
	if cacheCfg.Mode == "cluster" || cacheCfg.Mode == "sentinel" {
		keyMajority = 2 // For distributed setups, require majority of 3 keys
	}

	lockTTL := cacheCfg.DefaultTTL
	if lockTTL == 0 {
		lockTTL = 5 * time.Second // Default lock validity
	}

	locker, err := valkeylock.NewLocker(valkeylock.LockerOption{
		ClientBuilder: func(option valkey.ClientOption) (valkey.Client, error) {
			return vkClient, nil // Reuse existing client
		},
		KeyPrefix:      cacheCfg.CacheName + ":lock",
		KeyValidity:    lockTTL,
		ExtendInterval: lockTTL / 2,
		KeyMajority:    keyMajority,
		NoLoopTracking: true, // Better performance for Valkey >= 7.0.5
	})
	if err != nil {
		return nil, err, nil
	}

	cleanup := func() {
		// Note: We don't close the locker here because it uses the shared vkClient.
		// Closing the locker would close the shared client and break other cache instances.
		// The shared vkClient will be closed when the application exits.
		// If you need explicit cleanup, manage the locker lifecycle separately.
	}

	return &RemoteCacheValkey[T]{
		name:        cacheCfg.CacheName,
		ttl:         cacheCfg.DefaultTTL,
		maxElements: cacheCfg.MaxElements,
		applyTouch:  cacheCfg.ApplyTouch,
		serDe:       cacheCfg.SerDe,
		locker:      locker,
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

// DeleteMulti removes multiple keys from the cache.
// Returns the number of keys that were deleted.
// Uses the DEL command which accepts multiple keys and is atomic.
func (c *RemoteCacheValkey[T]) DeleteMulti(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	// Convert all keys to composite keys with cache name prefix
	compositeKeys := make([]string, len(keys))
	for i, key := range keys {
		compositeKeys[i] = c.makeKey(key)
	}

	// Use DEL command which accepts multiple keys
	cmd := vkClient.B().Del().Key(compositeKeys...).Build()
	count, err := vkClient.Do(ctx, cmd).ToInt64()
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Increment increments the integer value of a key by delta.
func (c *RemoteCacheValkey[T]) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	cmd := vkClient.B().Incrby().Key(c.makeKey(key)).Increment(delta).Build()
	return vkClient.Do(ctx, cmd).ToInt64()
}

// Decrement decrements the integer value of a key by delta.
func (c *RemoteCacheValkey[T]) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	cmd := vkClient.B().Decrby().Key(c.makeKey(key)).Decrement(delta).Build()
	return vkClient.Do(ctx, cmd).ToInt64()
}

// HSet sets the value of a field in a hash.
func (c *RemoteCacheValkey[T]) HSet(ctx context.Context, key string, field string, value T) error {
	data, err := c.serDe.Serialize(value)
	if err != nil {
		return err
	}
	// Attempting to use Hset with FieldValue matching generated code patterns.
	// If FieldValue has no methods, maybe it takes args? BUT previous error said "too many arguments".
	// This is contradictory unless I misread error or it takes variadic optional?
	// Let's rely on standard method: Key().FieldValue().Field("f").Value("v") is failing.
	// Maybe Key().Item().Field().Value()? No Items.

	// I will try to use the raw Arbitrary command if available on builder to fix this quickly as I can't see source.
	// If not, I will try `Hset().Key(key).Record()...`

	// Actually, looking at `rueidis` again: `Hset().Key(key).FieldValue().Field(f).Value(v)` IS correct there.
	// If `valkey-go` follows it, it should work.
	// Maybe there's a typo in my previous attempt? `vkClient.B().Hset().Key(c.makeKey(key)).FieldValue().Field(field).Value(string(data)).Build()`
	// Error: `type cmds.HsetFieldValue has no field or method Field`.

	// This defines `HsetFieldValue` exists.
	// Maybe it's `F` and `V`?
	// Or maybe `Field` is `Field` but `Value` is `Val`?

	// Let's try `vkClient.B().Hset().Key(c.makeKey(key)).Items().Add(field, string(data))` again?
	// Wait, Step 67 said `HsetKey` has no `Items`.

	// What about `Hsetex`?
	// Error: `type cmds.HsetexKey has no field or method Seconds`.
	// Maybe `Hsetex` takes seconds in `Hsetex(seconds)`? No.
	// Maybe `Key().Seconds(s)`? No.
	// Maybe `Key().Ex(time)`?

	// Strategies:
	// 1. Try `Hset().Key().Values()...`?
	// 2. Try usage of `Arbitrary`.

	// I'll try Arbitrary and call Build() to get Completed.
	cmd := vkClient.B().Arbitrary("HSET", c.makeKey(key), field, string(data)).Build()
	return vkClient.Do(ctx, cmd).Error()
}

// HSetWithTTL sets the value of a field in a hash with individual field expiration.
// This uses HSETEX command (Valkey 9.0+) which allows setting expiration on individual fields.
// Each field in the hash can have its own independent expiration time.
func (c *RemoteCacheValkey[T]) HSetWithTTL(ctx context.Context, key string, field string, value T, ttl time.Duration) error {
	data, err := c.serDe.Serialize(value)
	if err != nil {
		return err
	}

	compositeKey := c.makeKey(key)

	// Use HSETEX with correct syntax: HSETEX key EX seconds FIELDS numfields field value
	// Note: numfields is always 1 since we're setting a single field
	cmd := vkClient.B().Arbitrary("HSETEX", compositeKey,
		"EX", strconv.FormatInt(int64(ttl.Seconds()), 10),
		"FIELDS", "1",
		field, string(data)).Build()

	return vkClient.Do(ctx, cmd).Error()
}

// HGet gets the value of a field in a hash.
// If applyTouch is enabled, this uses HGETEX to extend the field's TTL on access.
func (c *RemoteCacheValkey[T]) HGet(ctx context.Context, key string, field string) (T, bool) {
	compositeKey := c.makeKey(key)
	var val string
	var err error

	if !c.applyTouch {
		// Standard HGET - no TTL extension
		cmd := vkClient.B().Hget().Key(compositeKey).Field(field).Build()
		val, err = vkClient.Do(ctx, cmd).ToString()
	} else {
		// HGETEX - extends field TTL on access (Valkey 9.0+)
		// Syntax: HGETEX key EX seconds FIELDS numfields field
		cmd := vkClient.B().Arbitrary("HGETEX", compositeKey,
			"EX", strconv.FormatInt(int64(c.ttl.Seconds()), 10),
			"FIELDS", "1", field).Build()

		// HGETEX returns an array of values
		result, err := vkClient.Do(ctx, cmd).AsStrSlice()
		if err != nil {
			var zero T
			if valkey.IsValkeyNil(err) {
				return zero, false
			}
			return zero, false
		}

		// Get first element from array
		if len(result) == 0 || result[0] == "" {
			var zero T
			return zero, false
		}
		val = result[0]
		err = nil
	}

	if err != nil {
		var zero T
		if valkey.IsValkeyNil(err) {
			return zero, false
		}
		return zero, false
	}

	result, err := c.serDe.Deserialize([]byte(val))
	if err != nil {
		var zero T
		return zero, false
	}
	return result, true
}

// HGetAll gets all fields and values in a hash.
func (c *RemoteCacheValkey[T]) HGetAll(ctx context.Context, key string) (map[string]T, error) {
	cmd := vkClient.B().Hgetall().Key(c.makeKey(key)).Build()

	// Fixing AsStrMap -> AsMap and checking values
	val, err := vkClient.Do(ctx, cmd).AsMap()
	if err != nil {
		return nil, err
	}

	result := make(map[string]T, len(val))
	for k, v := range val {
		// v is ValkeyMessage?
		s, err := v.ToString()
		if err != nil {
			return nil, err
		}
		deserialized, err := c.serDe.Deserialize([]byte(s))
		if err != nil {
			return nil, err
		}
		result[k] = deserialized
	}
	return result, nil
}

// HDel deletes one or more fields from a hash.
func (c *RemoteCacheValkey[T]) HDel(ctx context.Context, key string, fields ...string) error {
	cmd := vkClient.B().Hdel().Key(c.makeKey(key)).Field(fields...).Build()
	return vkClient.Do(ctx, cmd).Error()
}

// HExpire sets expiration time on an existing hash field (Valkey 9.0+).
// Returns true if the field exists and expiration was set.
// Syntax: HEXPIRE key seconds FIELDS numfields field [field ...]
func (c *RemoteCacheValkey[T]) HExpire(ctx context.Context, key string, field string, ttl time.Duration) (bool, error) {
	compositeKey := c.makeKey(key)

	// HEXPIRE key seconds FIELDS numfields field
	cmd := vkClient.B().Arbitrary("HEXPIRE", compositeKey,
		strconv.FormatInt(int64(ttl.Seconds()), 10),
		"FIELDS", "1", field).Build()

	// HEXPIRE returns an array with status for each field
	// 1 = expiration set, -2 = field doesn't exist, 0 = operation skipped
	result, err := vkClient.Do(ctx, cmd).AsIntSlice()
	if err != nil {
		return false, err
	}

	if len(result) > 0 && result[0] == 1 {
		return true, nil
	}
	return false, nil
}

// HTTL returns the remaining TTL of a hash field in seconds.
// Returns -2 if field doesn't exist, -1 if field has no expiration.
// Syntax: HTTL key FIELDS numfields field [field ...]
func (c *RemoteCacheValkey[T]) HTTL(ctx context.Context, key string, field string) (int64, error) {
	compositeKey := c.makeKey(key)

	// HTTL key FIELDS numfields field
	cmd := vkClient.B().Arbitrary("HTTL", compositeKey,
		"FIELDS", "1", field).Build()

	// HTTL returns an array of TTL values for each field
	result, err := vkClient.Do(ctx, cmd).AsIntSlice()
	if err != nil {
		return -2, err
	}

	if len(result) > 0 {
		return result[0], nil
	}
	return -2, nil
}

// LockWithContext acquires a distributed lock for the given key.
// Returns a new context and a cancel function to release the lock.
// The returned context is automatically canceled if the lock is lost.
// This uses valkeylock which implements the Redlock algorithm for safe distributed locking.
func (c *RemoteCacheValkey[T]) LockWithContext(ctx context.Context, key string) (context.Context, context.CancelFunc, error) {
	return c.locker.WithContext(ctx, c.makeKey(key))
}

// TryLockWithContext attempts to acquire a lock without waiting.
// Returns ErrNotLocked if the lock cannot be acquired immediately.
func (c *RemoteCacheValkey[T]) TryLockWithContext(ctx context.Context, key string) (context.Context, context.CancelFunc, error) {
	lockCtx, cancel, err := c.locker.TryWithContext(ctx, c.makeKey(key))
	if err != nil {
		// Check if it's a "not locked" error from valkeylock
		if strings.Contains(err.Error(), "not locked") {
			return nil, nil, fmt.Errorf("%w: %v", ErrNotLocked, err)
		}
		return nil, nil, err
	}
	return lockCtx, cancel, nil
}
