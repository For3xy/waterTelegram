package repository

import (
	"sync"
	"time"
	"waterTelegram/pkg/post"
)

var (
	postsCache []post.Post
	lastUpdate time.Time
	mu         sync.RWMutex
	cacheTTL   = 24 * time.Hour
)

func UpdateCache(newPosts []post.Post) {
	mu.Lock()
	defer mu.Unlock()
	postsCache = newPosts
	lastUpdate = time.Now()
}

func RefreshCacheTime() {
	mu.Lock()
	defer mu.Unlock()
	lastUpdate = time.Now()
}

func GetPosts() []post.Post {
	mu.RLock()
	defer mu.RUnlock()
	if time.Since(lastUpdate) > cacheTTL {
		return nil
	}

	return postsCache
}

func IsCacheEmpty() bool {
	mu.RLock()
	defer mu.RUnlock()
	return postsCache == nil || time.Since(lastUpdate) > cacheTTL
}
