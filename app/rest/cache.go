package rest

import (
	"log"
	"net/http"
	"time"

	cache "github.com/patrickmn/go-cache"
)

// LoadingCache defines interface for caching
type LoadingCache interface {
	Get(key string, ttl time.Duration, fn func() ([]byte, error)) (data []byte, err error)
	Flush()
}

// loadingCache implements LoadingCache interface on top of cache.Cache
type loadingCache struct {
	bytesCache  *cache.Cache
	postFlushFn func()
}

// NewLoadingCache makes loadingCache implementation
func NewLoadingCache(defaultExpiration, cleanupInterval time.Duration, postFlushFn func()) LoadingCache {
	return &loadingCache{bytesCache: cache.New(defaultExpiration, cleanupInterval), postFlushFn: postFlushFn}
}

// Get is loading cache method to get value by key or load via fn if not found
func (lc *loadingCache) Get(key string, ttl time.Duration, fn func() ([]byte, error)) (data []byte, err error) {
	if b, ok := lc.bytesCache.Get(key); ok {
		log.Printf("[DEBUG] cache hit %s", key)
		return b.([]byte), nil
	}

	log.Printf("[DEBUG] cache miss %s", key)
	if data, err = fn(); err != nil {
		return data, err
	}
	lc.bytesCache.Set(key, data, ttl)
	return data, nil
}

// Flush clears cache and calls postFlushFn async
func (lc *loadingCache) Flush() {
	lc.bytesCache.Flush()
	if lc.postFlushFn != nil {
		go lc.postFlushFn()
	}
}

// URLKey gets url from request to use is as cache key
// admins will have separate keys in order tp prevent leak of admin-only data to regular users
func URLKey(r *http.Request) string {
	key := r.URL.String()
	if user, err := GetUserInfo(r); err == nil && user.Admin { // make seprate cache key for admins
		key = "admin!!" + key
	}
	return key
}
