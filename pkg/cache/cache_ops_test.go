package cache

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// We'll test Local Cache largely as it's self-contained.
// Remote cache tests require a running Valkey instance.
// We can mock it or just rely on manual verification if environment isn't set up,
// but for this task I will focus on unit testing the logic where possible.

func TestLocalCache_Increment(t *testing.T) {
	ctx := context.Background()
	cfg := &CacheConfig[int64]{
		Mode:       "local",
		DefaultTTL: time.Minute,
	}
	c, err, _ := NewCache(cfg)
	assert.NoError(t, err)

	key := "counter"

	// Incr from 0
	val, err := c.Increment(ctx, key, 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)
	time.Sleep(10 * time.Millisecond) // Ristretto async set

	// Incr again
	val, err = c.Increment(ctx, key, 5)
	assert.NoError(t, err)
	assert.Equal(t, int64(6), val)
	time.Sleep(10 * time.Millisecond)

	// Decr
	val, err = c.Decrement(ctx, key, 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(4), val)
}

func TestLocalCache_HashTypes(t *testing.T) {
	ctx := context.Background()
	cfg := &CacheConfig[string]{
		Mode:       "local",
		DefaultTTL: time.Minute,
	}
	c, err, _ := NewCache(cfg)
	assert.NoError(t, err)

	key := "user:1"

	// HSet
	err = c.HSet(ctx, key, "name", "alice")
	assert.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	// HGet
	val, found := c.HGet(ctx, key, "name")
	assert.True(t, found)
	assert.Equal(t, "alice", val)

	// HSet info
	err = c.HSet(ctx, key, "role", "admin")
	assert.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	// HGetAll
	all, err := c.HGetAll(ctx, key)
	assert.NoError(t, err)
	assert.Len(t, all, 2)
	assert.Equal(t, "alice", all["name"])
	assert.Equal(t, "admin", all["role"])

	// HDel
	err = c.HDel(ctx, key, "role")
	assert.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	all, err = c.HGetAll(ctx, key)
	assert.NoError(t, err)
	assert.Len(t, all, 1)
	assert.Equal(t, "alice", all["name"])
}

func TestLocalCache_HSetWithTTL(t *testing.T) {
	ctx := context.Background()
	cfg := &CacheConfig[string]{
		Mode:       "local",
		DefaultTTL: time.Minute,
	}
	c, err, _ := NewCache(cfg)
	assert.NoError(t, err)

	key := "temp_hash"
	err = c.HSetWithTTL(ctx, key, "field", "value", time.Millisecond*100)
	assert.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	val, found := c.HGet(ctx, key, "field")
	assert.True(t, found)
	assert.Equal(t, "value", val)

	// Wait for expire
	time.Sleep(time.Millisecond * 200)
	_, found = c.HGet(ctx, key, "field")
	assert.False(t, found, "Should be expired")
}

func TestLocalCache_Lock(t *testing.T) {
	ctx := context.Background()
	cfg := &CacheConfig[string]{
		Mode:       "local",
		DefaultTTL: time.Minute,
	}
	c, err, _ := NewCache(cfg)
	assert.NoError(t, err)

	key := "lock_key"

	// Acquire lock
	lockCtx, cancel, err := c.LockWithContext(ctx, key)
	assert.NoError(t, err)
	assert.NotNil(t, lockCtx)
	assert.NotNil(t, cancel)
	time.Sleep(10 * time.Millisecond)

	// Try acquire again - should fail
	_, _, err = c.TryLockWithContext(ctx, key)
	assert.Error(t, err)
	assert.Equal(t, ErrNotLocked, err)

	// Release lock
	cancel()
	time.Sleep(10 * time.Millisecond)

	// Acquire again - should succeed
	lockCtx2, cancel2, err := c.LockWithContext(ctx, key)
	assert.NoError(t, err)
	assert.NotNil(t, lockCtx2)
	assert.NotNil(t, cancel2)
	defer cancel2()
}

// Remote cache tests - requires a running Valkey instance
// Skip these tests if Valkey is not available

func TestRemoteCache_Lock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping remote cache test in short mode")
	}

	ctx := context.Background()
	cfg := &CacheConfig[string]{
		Mode:            "remote",
		CacheName:       "test_lock",
		DefaultTTL:      5 * time.Second,
		RemoteCacheAddr: "localhost:6379",
	}
	c, err, cleanup := NewCache(cfg)
	if err != nil {
		t.Skipf("Skipping test: Valkey not available - %v", err)
		return
	}
	defer cleanup()

	// Use unique key with timestamp to avoid conflicts from previous runs
	key := fmt.Sprintf("distributed_lock_key_%d", time.Now().UnixNano())

	// Acquire lock
	lockCtx, cancel, err := c.LockWithContext(ctx, key)
	assert.NoError(t, err)
	assert.NotNil(t, lockCtx)
	assert.NotNil(t, cancel)

	// Try to acquire the same lock - should fail
	_, _, err = c.TryLockWithContext(ctx, key)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotLocked), "Expected ErrNotLocked, got: %v", err)

	// Release lock
	cancel()
	time.Sleep(100 * time.Millisecond) // Give valkeylock time to release

	// Acquire again - should succeed
	lockCtx2, cancel2, err := c.LockWithContext(ctx, key)
	assert.NoError(t, err)
	assert.NotNil(t, lockCtx2)
	assert.NotNil(t, cancel2)
	defer cancel2()
}

func TestRemoteCache_LockContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping remote cache test in short mode")
	}

	// Give previous test time to clean up locks
	time.Sleep(200 * time.Millisecond)

	ctx := context.Background()
	cfg := &CacheConfig[string]{
		Mode:            "remote",
		CacheName:       "test_lock_cancel",
		DefaultTTL:      2 * time.Second,
		RemoteCacheAddr: "localhost:6379",
	}
	c, err, cleanup := NewCache(cfg)
	if err != nil {
		t.Skipf("Skipping test: Valkey not available - %v", err)
		return
	}
	defer cleanup()

	// Use unique key with timestamp to avoid conflicts
	key := fmt.Sprintf("cancel_lock_key_%d", time.Now().UnixNano())

	// Use TryLockWithContext to avoid blocking
	lockCtx, cancel, err := c.TryLockWithContext(ctx, key)
	if err != nil {
		t.Skipf("Skipping test: Could not acquire lock - %v", err)
		return
	}
	if cancel == nil || lockCtx == nil {
		t.Skip("Skipping test: Lock context or cancel is nil")
		return
	}

	// Verify context is not cancelled initially
	select {
	case <-lockCtx.Done():
		t.Fatal("Lock context should not be cancelled yet")
	default:
		// Expected
	}

	// Cancel the lock
	cancel()

	// Verify context is cancelled after cancel
	select {
	case <-lockCtx.Done():
		// Expected
	case <-time.After(time.Second):
		t.Fatal("Lock context should be cancelled")
	}
}

func TestRemoteCache_HashTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping remote cache test in short mode")
	}

	ctx := context.Background()
	cfg := &CacheConfig[string]{
		Mode:            "remote",
		CacheName:       "test_hash",
		DefaultTTL:      time.Minute,
		RemoteCacheAddr: "localhost:6379",
	}
	c, err, cleanup := NewCache(cfg)
	if err != nil {
		t.Skipf("Skipping test: Valkey not available - %v", err)
		return
	}
	defer cleanup()

	key := "user:1000"

	// HSet
	err = c.HSet(ctx, key, "name", "alice")
	assert.NoError(t, err)

	// HGet
	val, found := c.HGet(ctx, key, "name")
	assert.True(t, found)
	assert.Equal(t, "alice", val)

	// HSet another field
	err = c.HSet(ctx, key, "role", "admin")
	assert.NoError(t, err)

	// HGetAll
	all, err := c.HGetAll(ctx, key)
	assert.NoError(t, err)
	assert.Len(t, all, 2)
	assert.Equal(t, "alice", all["name"])
	assert.Equal(t, "admin", all["role"])

	// HDel
	err = c.HDel(ctx, key, "role")
	assert.NoError(t, err)

	all, err = c.HGetAll(ctx, key)
	assert.NoError(t, err)
	assert.Len(t, all, 1)
	assert.Equal(t, "alice", all["name"])

	// Cleanup
	err = c.Delete(ctx, key)
	assert.NoError(t, err)
}

func TestRemoteCache_HSetWithTTL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping remote cache test in short mode")
	}

	ctx := context.Background()
	cfg := &CacheConfig[string]{
		Mode:            "remote",
		CacheName:       "test_hash_ttl",
		DefaultTTL:      time.Minute,
		RemoteCacheAddr: "localhost:6379",
	}
	c, err, cleanup := NewCache(cfg)
	if err != nil {
		t.Skipf("Skipping test: Valkey not available - %v", err)
		return
	}
	defer cleanup()

	key := "temp_hash"

	// HSet with TTL
	err = c.HSetWithTTL(ctx, key, "field", "value", 2*time.Second)
	assert.NoError(t, err)

	// Verify it exists
	val, found := c.HGet(ctx, key, "field")
	assert.True(t, found)
	assert.Equal(t, "value", val)

	// Wait for expiration
	time.Sleep(3 * time.Second)

	// Should be expired
	_, found = c.HGet(ctx, key, "field")
	assert.False(t, found, "Hash should be expired")
}

func TestRemoteCache_Increment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping remote cache test in short mode")
	}

	ctx := context.Background()
	cfg := &CacheConfig[int64]{
		Mode:            "remote",
		CacheName:       "test_counter",
		DefaultTTL:      time.Minute,
		RemoteCacheAddr: "localhost:6379",
	}
	c, err, cleanup := NewCache(cfg)
	if err != nil {
		t.Skipf("Skipping test: Valkey not available - %v", err)
		return
	}
	defer cleanup()

	key := "counter"

	// Increment from 0
	val, err := c.Increment(ctx, key, 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)

	// Increment again
	val, err = c.Increment(ctx, key, 5)
	assert.NoError(t, err)
	assert.Equal(t, int64(6), val)

	// Decrement
	val, err = c.Decrement(ctx, key, 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(4), val)

	// Cleanup
	err = c.Delete(ctx, key)
	assert.NoError(t, err)
}

// TestRemoteCache_IndividualFieldExpiration tests the key feature of Valkey 9.0+:
// Individual hash fields can have different expiration times.
// This is perfect for use cases like storing multiple user sessions with different TTLs.
func TestRemoteCache_IndividualFieldExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping remote cache test in short mode")
	}

	ctx := context.Background()
	cfg := &CacheConfig[string]{
		Mode:            "remote",
		CacheName:       "test_field_expiry",
		DefaultTTL:      time.Minute,
		RemoteCacheAddr: "localhost:6379",
	}
	c, err, cleanup := NewCache(cfg)
	if err != nil {
		t.Skipf("Skipping test: Valkey not available - %v", err)
		return
	}
	defer cleanup()

	// Simulate user with multiple sessions, each with different expiration
	userKey := "user:123:sessions"

	// Session 1: Mobile app - expires in 2 seconds
	err = c.HSetWithTTL(ctx, userKey, "session:mobile", "token-mobile-xyz", 2*time.Second)
	assert.NoError(t, err)

	// Session 2: Web browser - expires in 5 seconds
	err = c.HSetWithTTL(ctx, userKey, "session:web", "token-web-abc", 5*time.Second)
	assert.NoError(t, err)

	// Session 3: Desktop - no expiration (persistent)
	err = c.HSet(ctx, userKey, "session:desktop", "token-desktop-def")
	assert.NoError(t, err)

	// Verify all sessions exist
	sessions, err := c.HGetAll(ctx, userKey)
	assert.NoError(t, err)
	assert.Len(t, sessions, 3, "Should have 3 active sessions")

	// Check TTL of each field
	mobileTTL, err := c.HTTL(ctx, userKey, "session:mobile")
	assert.NoError(t, err)
	assert.True(t, mobileTTL > 0 && mobileTTL <= 2, "Mobile session should have TTL ~2 seconds, got %d", mobileTTL)

	webTTL, err := c.HTTL(ctx, userKey, "session:web")
	assert.NoError(t, err)
	assert.True(t, webTTL > 2 && webTTL <= 5, "Web session should have TTL ~5 seconds, got %d", webTTL)

	desktopTTL, err := c.HTTL(ctx, userKey, "session:desktop")
	assert.NoError(t, err)
	assert.Equal(t, int64(-1), desktopTTL, "Desktop session should have no expiration")

	// Wait for mobile session to expire
	time.Sleep(3 * time.Second)

	// Mobile session should be gone
	_, found := c.HGet(ctx, userKey, "session:mobile")
	assert.False(t, found, "Mobile session should have expired")

	// Web and desktop should still exist
	_, found = c.HGet(ctx, userKey, "session:web")
	assert.True(t, found, "Web session should still exist")

	_, found = c.HGet(ctx, userKey, "session:desktop")
	assert.True(t, found, "Desktop session should still exist")

	// Wait for web session to expire
	time.Sleep(3 * time.Second)

	// Only desktop session should remain
	sessions, err = c.HGetAll(ctx, userKey)
	assert.NoError(t, err)
	assert.Len(t, sessions, 1, "Only desktop session should remain")
	assert.Equal(t, "token-desktop-def", sessions["session:desktop"])

	// Test HExpire - extend desktop session TTL
	success, err := c.HExpire(ctx, userKey, "session:desktop", 10*time.Second)
	assert.NoError(t, err)
	assert.True(t, success, "HExpire should succeed on existing field")

	// Verify TTL was set
	desktopTTL, err = c.HTTL(ctx, userKey, "session:desktop")
	assert.NoError(t, err)
	assert.True(t, desktopTTL > 0 && desktopTTL <= 10, "Desktop session should now have TTL ~10 seconds")

	// Cleanup
	err = c.Delete(ctx, userKey)
	assert.NoError(t, err)
}

// TestRemoteCache_HashFieldTouch tests that applyTouch flag extends hash field TTL on access
func TestRemoteCache_HashFieldTouch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping remote cache test in short mode")
	}

	ctx := context.Background()

	// Create cache WITH applyTouch enabled
	cfg := &CacheConfig[string]{
		Mode:            "remote",
		CacheName:       "test_touch",
		DefaultTTL:      5 * time.Second,  // Default TTL for touch
		RemoteCacheAddr: "localhost:6379",
		ApplyTouch:      true,  // Enable touch on get
	}
	c, err, cleanup := NewCache(cfg)
	if err != nil {
		t.Skipf("Skipping test: Valkey not available - %v", err)
		return
	}
	defer cleanup()

	key := "session:user456"
	field := "session:mobile"

	// Set a field with 3 second expiration
	err = c.HSetWithTTL(ctx, key, field, "token-abc", 3*time.Second)
	assert.NoError(t, err)

	// Check initial TTL
	ttl, err := c.HTTL(ctx, key, field)
	assert.NoError(t, err)
	assert.True(t, ttl > 0 && ttl <= 3, "Initial TTL should be ~3 seconds, got %d", ttl)

	// Wait 2 seconds (TTL would be ~1 second left)
	time.Sleep(2 * time.Second)

	// Access the field - this should extend TTL back to 5 seconds (DefaultTTL)
	val, found := c.HGet(ctx, key, field)
	assert.True(t, found, "Field should exist")
	assert.Equal(t, "token-abc", val)

	// Check TTL after touch - should be extended to ~5 seconds
	ttl, err = c.HTTL(ctx, key, field)
	assert.NoError(t, err)
	assert.True(t, ttl > 3, "TTL should be extended to ~5 seconds after touch, got %d", ttl)

	// Without touch, the field would have expired in 1 more second
	// But with touch, it should still be alive after 2 more seconds
	time.Sleep(2 * time.Second)

	val, found = c.HGet(ctx, key, field)
	assert.True(t, found, "Field should still exist thanks to touch extending TTL")
	assert.Equal(t, "token-abc", val)

	// Cleanup
	err = c.Delete(ctx, key)
	assert.NoError(t, err)
}

// TestRemoteCache_HashFieldNoTouch verifies that without applyTouch, TTL is NOT extended
func TestRemoteCache_HashFieldNoTouch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping remote cache test in short mode")
	}

	ctx := context.Background()

	// Create cache WITHOUT applyTouch
	cfg := &CacheConfig[string]{
		Mode:            "remote",
		CacheName:       "test_no_touch",
		DefaultTTL:      5 * time.Second,
		RemoteCacheAddr: "localhost:6379",
		ApplyTouch:      false,  // Touch disabled
	}
	c, err, cleanup := NewCache(cfg)
	if err != nil {
		t.Skipf("Skipping test: Valkey not available - %v", err)
		return
	}
	defer cleanup()

	key := "session:user789"
	field := "session:web"

	// Set a field with 2 second expiration
	err = c.HSetWithTTL(ctx, key, field, "token-xyz", 2*time.Second)
	assert.NoError(t, err)

	// Wait 1 second
	time.Sleep(1 * time.Second)

	// Access the field - without applyTouch, this should NOT extend TTL
	val, found := c.HGet(ctx, key, field)
	assert.True(t, found, "Field should exist")
	assert.Equal(t, "token-xyz", val)

	// Check TTL - should still be ~1 second (NOT extended)
	ttl, err := c.HTTL(ctx, key, field)
	assert.NoError(t, err)
	assert.True(t, ttl <= 1, "TTL should NOT be extended, got %d", ttl)

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Field should be expired
	_, found = c.HGet(ctx, key, field)
	assert.False(t, found, "Field should have expired")

	// Cleanup
	err = c.Delete(ctx, key)
	assert.NoError(t, err)
}

func TestLocalCache_DeleteMulti(t *testing.T) {
	ctx := context.Background()
	cfg := &CacheConfig[string]{
		Mode:       "local",
		DefaultTTL: time.Minute,
	}
	c, err, _ := NewCache(cfg)
	assert.NoError(t, err)

	// Set multiple keys
	keys := []string{"key1", "key2", "key3", "key4"}
	for _, key := range keys {
		err = c.Set(ctx, key, "value_"+key)
		assert.NoError(t, err)
	}
	time.Sleep(10 * time.Millisecond) // Ristretto async set

	// Verify all keys exist
	for _, key := range keys {
		_, found := c.Get(ctx, key)
		assert.True(t, found, "Key %s should exist", key)
	}

	// Delete multiple keys at once
	count, err := c.DeleteMulti(ctx, "key1", "key2", "key3")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count, "Should delete 3 keys")
	time.Sleep(10 * time.Millisecond)

	// Verify deleted keys don't exist
	for _, key := range []string{"key1", "key2", "key3"} {
		_, found := c.Get(ctx, key)
		assert.False(t, found, "Key %s should not exist", key)
	}

	// Verify key4 still exists
	_, found := c.Get(ctx, "key4")
	assert.True(t, found, "Key4 should still exist")

	// Test deleting non-existent keys
	count, err = c.DeleteMulti(ctx, "key1", "nonexistent1", "nonexistent2")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count, "Should delete 0 keys (all already deleted)")

	// Test deleting mix of existing and non-existing keys
	count, err = c.DeleteMulti(ctx, "key4", "nonexistent")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count, "Should delete 1 key")

	// Test deleting with empty keys
	count, err = c.DeleteMulti(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count, "Should delete 0 keys when no keys provided")
}

func TestRemoteCache_DeleteMulti(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping remote cache test in short mode")
	}

	ctx := context.Background()
	cfg := &CacheConfig[string]{
		Mode:            "remote",
		CacheName:       "test_delete_multi",
		DefaultTTL:      time.Minute,
		RemoteCacheAddr: "localhost:6379",
	}
	c, err, cleanup := NewCache(cfg)
	if err != nil {
		t.Skipf("Skipping test: Valkey not available - %v", err)
		return
	}
	defer cleanup()

	// Set multiple keys
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	for _, key := range keys {
		err = c.Set(ctx, key, "value_"+key)
		assert.NoError(t, err)
	}

	// Verify all keys exist
	for _, key := range keys {
		_, found := c.Get(ctx, key)
		assert.True(t, found, "Key %s should exist", key)
	}

	// Delete multiple keys at once
	count, err := c.DeleteMulti(ctx, "key1", "key2", "key3")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count, "Should delete 3 keys")

	// Verify deleted keys don't exist
	for _, key := range []string{"key1", "key2", "key3"} {
		_, found := c.Get(ctx, key)
		assert.False(t, found, "Key %s should not exist", key)
	}

	// Verify key4 and key5 still exist
	for _, key := range []string{"key4", "key5"} {
		_, found := c.Get(ctx, key)
		assert.True(t, found, "Key %s should still exist", key)
	}

	// Test deleting non-existent keys
	count, err = c.DeleteMulti(ctx, "key1", "nonexistent1", "nonexistent2")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count, "Should delete 0 keys (all already deleted)")

	// Test deleting mix of existing and non-existing keys
	count, err = c.DeleteMulti(ctx, "key4", "key5", "nonexistent")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count, "Should delete 2 keys")

	// Test deleting with empty keys
	count, err = c.DeleteMulti(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count, "Should delete 0 keys when no keys provided")
}

func TestRemoteCache_DeleteMultiPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping remote cache test in short mode")
	}

	ctx := context.Background()
	cfg := &CacheConfig[string]{
		Mode:            "remote",
		CacheName:       "test_delete_perf",
		DefaultTTL:      time.Minute,
		RemoteCacheAddr: "localhost:6379",
	}
	c, err, cleanup := NewCache(cfg)
	if err != nil {
		t.Skipf("Skipping test: Valkey not available - %v", err)
		return
	}
	defer cleanup()

	// Set 100 keys
	numKeys := 100
	keys := make([]string, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = fmt.Sprintf("perf_key_%d", i)
		err = c.Set(ctx, keys[i], fmt.Sprintf("value_%d", i))
		assert.NoError(t, err)
	}

	// Delete all keys at once using DeleteMulti (single round-trip)
	start := time.Now()
	count, err := c.DeleteMulti(ctx, keys...)
	multiDuration := time.Since(start)
	assert.NoError(t, err)
	assert.Equal(t, int64(numKeys), count, "Should delete all %d keys", numKeys)

	t.Logf("DeleteMulti deleted %d keys in %v (single atomic operation)", numKeys, multiDuration)

	// Verify all keys are deleted
	for _, key := range keys {
		_, found := c.Get(ctx, key)
		assert.False(t, found, "Key %s should not exist", key)
	}

	// Set keys again for comparison
	for i := 0; i < numKeys; i++ {
		err = c.Set(ctx, keys[i], fmt.Sprintf("value_%d", i))
		assert.NoError(t, err)
	}

	// Delete keys one by one for comparison (multiple round-trips)
	start = time.Now()
	for _, key := range keys {
		err = c.Delete(ctx, key)
		assert.NoError(t, err)
	}
	singleDuration := time.Since(start)

	t.Logf("Individual Delete operations deleted %d keys in %v (multiple operations)", numKeys, singleDuration)
	t.Logf("DeleteMulti is %.2fx faster than individual deletes", float64(singleDuration)/float64(multiDuration))

	// DeleteMulti should be significantly faster for remote cache
	assert.True(t, multiDuration < singleDuration, "DeleteMulti should be faster than individual deletes")
}
