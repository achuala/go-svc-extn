package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/achuala/go-svc-extn/pkg/cache"
	"github.com/stretchr/testify/assert"
)

func TestLocalCache(t *testing.T) {
	cache, err, cleanup := cache.NewCache(&cache.CacheConfig{Mode: "local"})
	defer cleanup()
	value := "val1"
	cache.Set(context.Background(), "key1", value)
	time.Sleep(time.Second * 1)
	if found, ok := cache.Get(context.Background(), "key1"); ok {
		t.Log(found)
	} else {
		assert := assert.New(t)
		assert.Equal(value, found, "they should be equal")
	}
	assert.NoError(t, err)
}

func TestRemoteCache(t *testing.T) {
	// Initialize remote cache
	remoteCache, err, cleanup := cache.NewCache(&cache.CacheConfig{
		Mode:            "remote",
		CacheName:       "test",
		DefaultTTL:      time.Second * 10,
		RemoteCacheAddr: "localhost:6379", // Adjust this to your remote cache address
	})
	remoteCache2, err1, cleanup1 := cache.NewCache(&cache.CacheConfig{
		Mode:            "remote",
		CacheName:       "test1",
		DefaultTTL:      time.Second * 10,
		ApplyTouch:      true,
		RemoteCacheAddr: "localhost:6379", // Adjust this to your remote cache address
	})
	assert.NoError(t, err)
	assert.NoError(t, err1)
	defer cleanup1()
	defer cleanup()

	ctx := context.Background()
	key := "remoteKey"
	value := "remoteValue"

	// Test Set
	err = remoteCache.Set(ctx, key, value)
	assert.NoError(t, err)

	// Test Get
	retrievedValue, ok := remoteCache.Get(ctx, key)
	assert.True(t, ok)
	assert.Equal(t, value, retrievedValue)

	// Test Delete
	err = remoteCache.Delete(ctx, key)
	assert.NoError(t, err)

	// Verify key is deleted
	_, ok = remoteCache.Get(ctx, key)
	assert.False(t, ok)

	// Test SetWithTTL
	ttl := 2 * time.Second
	err = remoteCache.SetWithTTL(ctx, key, value, ttl)
	assert.NoError(t, err)

	// Verify key exists
	retrievedValue, ok = remoteCache.Get(ctx, key)
	assert.True(t, ok)
	assert.Equal(t, value, retrievedValue)

	// Wait for TTL to expire
	time.Sleep(ttl + time.Second)

	// Verify key has expired
	_, ok = remoteCache.Get(ctx, key)
	assert.False(t, ok)

	remoteCache2.Set(ctx, key, value)
	time.Sleep(time.Second * 1)
	retrievedValue, ok = remoteCache2.Get(ctx, key)
	assert.True(t, ok)
	assert.Equal(t, value, retrievedValue)
	// Wait for TTL to expire
	time.Sleep(ttl + time.Second)

	// Verify key has not expired
	_, ok = remoteCache2.Get(ctx, key)
	assert.True(t, ok)
}
