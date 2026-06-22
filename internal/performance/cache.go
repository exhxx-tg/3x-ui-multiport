package performance

import (
	"sync"
	"time"
)

type cacheEntry[V any] struct {
	value   V
	expires int64
}

type Cache[K comparable, V any] struct {
	mu       sync.RWMutex
	entries  map[K]*cacheEntry[V]
	ttl      time.Duration
	maxSize  int
	hitFn    func()
	missFn   func()
}

func NewCache[K comparable, V any](ttl time.Duration, maxSize int) *Cache[K, V] {
	c := &Cache[K, V]{
		entries: make(map[K]*cacheEntry[V]),
		ttl:     ttl,
		maxSize: maxSize,
		hitFn:   func() { TrackCacheResult(true) },
		missFn:  func() { TrackCacheResult(false) },
	}
	return c
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		if c.missFn != nil {
			c.missFn()
		}
		var zero V
		return zero, false
	}

	if time.Now().UnixNano() > entry.expires {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		if c.missFn != nil {
			c.missFn()
		}
		var zero V
		return zero, false
	}

	if c.hitFn != nil {
		c.hitFn()
	}
	return entry.value, true
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.maxSize {
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}

	c.entries[key] = &cacheEntry[V]{
		value:   value,
		expires: time.Now().Add(c.ttl).UnixNano(),
	}
}

func (c *Cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.maxSize {
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}

	c.entries[key] = &cacheEntry[V]{
		value:   value,
		expires: time.Now().Add(ttl).UnixNano(),
	}
}

func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[K]*cacheEntry[V])
}

func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

func (c *Cache[K, V]) Cleanup() {
	now := time.Now().UnixNano()
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, entry := range c.entries {
		if now > entry.expires {
			delete(c.entries, k)
		}
	}
}

func (c *Cache[K, V]) GetOrSet(key K, fn func() (V, error)) (V, error) {
	if val, ok := c.Get(key); ok {
		return val, nil
	}
	val, err := fn()
	if err != nil {
		var zero V
		return zero, err
	}
	c.Set(key, val)
	return val, nil
}

type CacheManager struct {
	mu     sync.RWMutex
	caches map[string]any
	cfg    Config
}

var globalCacheManager *CacheManager

func InitCacheManager(cfg Config) {
	globalCacheManager = &CacheManager{
		caches: make(map[string]any),
		cfg:    cfg,
	}
}

func GlobalCacheManager() *CacheManager {
	return globalCacheManager
}

func (cm *CacheManager) GetOrCreateString(name string, ttl time.Duration, maxSize int) *Cache[string, string] {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if existing, ok := cm.caches[name]; ok {
		return existing.(*Cache[string, string])
	}

	if ttl <= 0 {
		ttl = cm.cfg.CacheDefaultTTL
	}
	if maxSize <= 0 {
		maxSize = cm.cfg.CacheMaxEntries
	}

	c := NewCache[string, string](ttl, maxSize)
	cm.caches[name] = c
	return c
}

func (cm *CacheManager) GetOrCreateAny(name string) *Cache[string, any] {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if existing, ok := cm.caches[name]; ok {
		return existing.(*Cache[string, any])
	}

	c := NewCache[string, any](cm.cfg.CacheDefaultTTL, cm.cfg.CacheMaxEntries)
	cm.caches[name] = c
	return c
}

func (cm *CacheManager) GetOrCreateInt(name string) *Cache[string, int64] {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if existing, ok := cm.caches[name]; ok {
		return existing.(*Cache[string, int64])
	}

	c := NewCache[string, int64](cm.cfg.CacheDefaultTTL, cm.cfg.CacheMaxEntries)
	cm.caches[name] = c
	return c
}

func (cm *CacheManager) CleanupAll() {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, c := range cm.caches {
		if cleaner, ok := c.(interface{ Cleanup() }); ok {
			cleaner.Cleanup()
		}
	}
}

func (cm *CacheManager) ClearAll() {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, c := range cm.caches {
		if clearer, ok := c.(interface{ Clear() }); ok {
			clearer.Clear()
		}
	}
}

func (cm *CacheManager) Stats() map[string]int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := make(map[string]int, len(cm.caches))
	for name, c := range cm.caches {
		if l, ok := c.(interface{ Len() int }); ok {
			stats[name] = l.Len()
		}
	}
	return stats
}
