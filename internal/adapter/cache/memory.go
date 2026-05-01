// Package cache provides in-memory caching adapters for paimon-mcp-fetch.
package cache

import (
	"context"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/paimonchan/paimon-mcp-fetch/internal/domain"
)

// entry wraps a cache value with expiration time.
type entry struct {
	value      *domain.CacheEntry
	expiresAt  time.Time
}

// MemoryCache is an in-memory LRU cache with TTL support.
type MemoryCache struct {
	mu       sync.RWMutex
	lru      *lru.Cache[string, *entry]
	defaultTTL time.Duration
}

// NewMemoryCache creates a new in-memory cache with the given max entries and TTL.
func NewMemoryCache(maxEntries int, defaultTTL time.Duration) (*MemoryCache, error) {
	cache, err := lru.New[string, *entry](maxEntries)
	if err != nil {
		return nil, err
	}
	return &MemoryCache{
		lru:        cache,
		defaultTTL: defaultTTL,
	}, nil
}

// Get retrieves a value from the cache if it exists and is not expired.
func (c *MemoryCache) Get(ctx context.Context, key string) (*domain.CacheEntry, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, found := c.lru.Get(key)
	if !found {
		return nil, false, nil
	}

	// Check expiration
	if time.Now().After(e.expiresAt) {
		c.lru.Remove(key)
		return nil, false, nil
	}

	return e.value, true, nil
}

// Set stores a value in the cache with the given TTL.
func (c *MemoryCache) Set(ctx context.Context, key string, value *domain.CacheEntry, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.lru.Add(key, &entry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	})
	return nil
}

// Invalidate removes a value from the cache.
func (c *MemoryCache) Invalidate(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lru.Remove(key)
	return nil
}

// Stats returns cache statistics for observability.
func (c *MemoryCache) Stats() (hits, misses int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// LRU stats are tracked internally; we could add our own counters if needed.
	// For now, return simple values.
	return 0, 0
}

// Len returns the current number of items in the cache.
func (c *MemoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lru.Len()
}
