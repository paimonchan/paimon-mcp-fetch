package cache

import (
	"context"
	"testing"
	"time"

	"github.com/user/paimon-mcp-fetch/internal/domain"
)

func TestMemoryCache_GetSet(t *testing.T) {
	c, err := NewMemoryCache(10, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	ctx := context.Background()
	key := "test-key"
	entry := &domain.CacheEntry{
		Body:        []byte("hello world"),
		ContentType: "text/html",
		FinalURL:    "https://example.com",
	}

	// Set
	if err := c.Set(ctx, key, entry, time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get
	got, found, err := c.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("expected cache hit")
	}
	if string(got.Body) != "hello world" {
		t.Errorf("body = %q, want %q", string(got.Body), "hello world")
	}
}

func TestMemoryCache_Expiration(t *testing.T) {
	c, err := NewMemoryCache(10, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	ctx := context.Background()
	key := "expiring-key"
	entry := &domain.CacheEntry{Body: []byte("data")}

	// Set with short TTL
	c.Set(ctx, key, entry, 50*time.Millisecond)

	// Should be found immediately
	_, found, _ := c.Get(ctx, key)
	if !found {
		t.Fatal("expected cache hit before expiration")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	_, found, _ = c.Get(ctx, key)
	if found {
		t.Fatal("expected cache miss after expiration")
	}
}

func TestMemoryCache_Invalidation(t *testing.T) {
	c, err := NewMemoryCache(10, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	ctx := context.Background()
	key := "invalidate-key"
	c.Set(ctx, key, &domain.CacheEntry{Body: []byte("data")}, time.Minute)

	// Invalidate
	if err := c.Invalidate(ctx, key); err != nil {
		t.Fatalf("Invalidate failed: %v", err)
	}

	_, found, _ := c.Get(ctx, key)
	if found {
		t.Fatal("expected cache miss after invalidation")
	}
}

func TestMemoryCache_LRU(t *testing.T) {
	c, err := NewMemoryCache(2, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	ctx := context.Background()
	c.Set(ctx, "a", &domain.CacheEntry{Body: []byte("A")}, time.Minute)
	c.Set(ctx, "b", &domain.CacheEntry{Body: []byte("B")}, time.Minute)
	c.Set(ctx, "c", &domain.CacheEntry{Body: []byte("C")}, time.Minute) // Should evict "a"

	_, found, _ := c.Get(ctx, "a")
	if found {
		t.Fatal("expected 'a' to be evicted (LRU)")
	}

	_, found, _ = c.Get(ctx, "b")
	if !found {
		t.Fatal("expected 'b' to still be in cache")
	}
}
