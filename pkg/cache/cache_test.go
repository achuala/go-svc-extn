package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/achuala/go-svc-extn/pkg/cache"
	"github.com/stretchr/testify/assert"
)

func TestLocalCache(t *testing.T) {
	cache, cleanup := cache.NewCache(&cache.CacheConfig{Mode: "local"})
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

}
