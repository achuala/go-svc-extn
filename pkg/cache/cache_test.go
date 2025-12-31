package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/achuala/go-svc-extn/pkg/cache"
	"github.com/stretchr/testify/require"
)

type TestUser struct {
	ID   int
	Name string
}

func TestLocalGenericCache(t *testing.T) {
	cache, err, cleanup := cache.NewCache[TestUser](&cache.CacheConfig[TestUser]{
		Mode: "local",
	})
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	user := TestUser{ID: 1, Name: "Alice"}
	err = cache.Set(ctx, "user:1", user)
	require.NoError(t, err)

	// Ristretto is async, so wait a bit for the set to complete
	time.Sleep(10 * time.Millisecond)

	got, found := cache.Get(ctx, "user:1")
	require.True(t, found)
	require.Equal(t, user, got)

	// Test SetWithTTL
	err = cache.SetWithTTL(ctx, "user:2", user, 1*time.Second)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	got, found = cache.Get(ctx, "user:2")
	require.True(t, found)
	time.Sleep(2 * time.Second)
	_, found = cache.Get(ctx, "user:2")
	require.False(t, found)
}

func TestRemoteGenericCache(t *testing.T) {
	cache, err, cleanup := cache.NewCache[TestUser](&cache.CacheConfig[TestUser]{
		Mode:            "remote",
		CacheName:       "test",
		DefaultTTL:      2 * time.Second,
		RemoteCacheAddr: "localhost:6379", // Adjust as needed
		SerDe:           cache.NewJSONSerDe[TestUser](),
	})
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	user := TestUser{ID: 2, Name: "Bob"}
	err = cache.Set(ctx, "user:remote", user)
	require.NoError(t, err)

	got, found := cache.Get(ctx, "user:remote")
	require.True(t, found)
	require.Equal(t, user, got)

	// Test SetWithTTL
	err = cache.SetWithTTL(ctx, "user:remote2", user, 1*time.Second)
	require.NoError(t, err)
	got, found = cache.Get(ctx, "user:remote2")
	require.True(t, found)
	time.Sleep(2 * time.Second)
	_, found = cache.Get(ctx, "user:remote2")
	require.False(t, found)
}

func TestLocalStringCache(t *testing.T) {
	cache, err, cleanup := cache.NewCache[string](&cache.CacheConfig[string]{Mode: "local"})
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	err = cache.Set(ctx, "key", "value")
	require.NoError(t, err)

	// Ristretto is async, so wait a bit for the set to complete
	time.Sleep(10 * time.Millisecond)

	got, found := cache.Get(ctx, "key")
	require.True(t, found)
	require.Equal(t, "value", got)
}
