package cache_test

import (
	"testing"
	"time"

	"github.com/rahul2393/small-assignment-server/cache"
)

func TestCache(t *testing.T) {
	cache.Set("namespace", "key", cache.Item{
		Src: 1,
	})
	src, _ := cache.Get("namespace", "key")
	i := src.(int)
	if i != 1 {
		t.Error("cache should of stored 1")
	}
	cache.Delete("namespace", "key")
	_, in := cache.Get("namespace", "key")
	if in {
		t.Error("cache delete should remove item")
	}
}

func TestExpire(t *testing.T) {
	cache.Set("namespace", "key", cache.Item{
		Src:      1,
		Duration: time.Millisecond,
	})
	time.Sleep(time.Millisecond * 10)
	_, in := cache.Get("namespace", "key")
	if in {
		t.Error("cache should expire item")
	}
}
