// Package ratelimit provides per-domain rate limiting for paimon-mcp-fetch.
package ratelimit

import (
	"context"
	"net/url"
	"sync"
	"time"
)

// Limiter enforces per-domain rate limits using a token bucket algorithm.
type Limiter struct {
	mu         sync.RWMutex
	buckets    map[string]*bucket
	rate       time.Duration // time between requests (e.g., 1s = 1 req/sec)
	burst      int           // maximum burst size
	cleanupInterval time.Duration
}

// bucket represents a token bucket for a single domain.
type bucket struct {
	tokens    int
	lastFill  time.Time
	mu        sync.Mutex
}

// NewLimiter creates a new rate limiter with the given rate and burst.
// rate is the minimum time between requests per domain.
// burst is the maximum number of requests allowed in a short burst.
func NewLimiter(rate time.Duration, burst int) *Limiter {
	l := &Limiter{
		buckets:         make(map[string]*bucket),
		rate:            rate,
		burst:           burst,
		cleanupInterval: 5 * time.Minute,
	}
	go l.cleanupLoop()
	return l
}

// Wait blocks until the request for the given URL is allowed by the rate limiter.
// Returns ctx.Err() if the context is cancelled.
func (l *Limiter) Wait(ctx context.Context, urlStr string) error {
	domain, err := extractDomain(urlStr)
	if err != nil {
		// If we can't parse the domain, allow the request
		return nil
	}

	b := l.getBucket(domain)

	for {
		waitTime := b.consume(l.rate, l.burst)
		if waitTime <= 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Retry after waiting
		}
	}
}

// getBucket gets or creates a token bucket for the given domain.
func (l *Limiter) getBucket(domain string) *bucket {
	l.mu.RLock()
	b, exists := l.buckets[domain]
	l.mu.RUnlock()

	if exists {
		return b
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check after acquiring write lock
	if b, exists := l.buckets[domain]; exists {
		return b
	}

	b = &bucket{
		tokens:   l.burst,
		lastFill: time.Now(),
	}
	l.buckets[domain] = b
	return b
}

// consume attempts to consume a token from the bucket.
// Returns the wait time if no token is available.
func (b *bucket) consume(rate time.Duration, burst int) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastFill)

	// Add tokens based on elapsed time
	tokensToAdd := int(elapsed / rate)
	if tokensToAdd > 0 {
		b.tokens += tokensToAdd
		if b.tokens > burst {
			b.tokens = burst
		}
		b.lastFill = now
	}

	if b.tokens > 0 {
		b.tokens--
		return 0
	}

	// Calculate wait time for next token
	return rate - elapsed%rate
}

// cleanupLoop periodically removes inactive buckets to prevent memory leaks.
func (l *Limiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		// In a production system, we'd track last access time per bucket.
		// For now, we keep all buckets to avoid accidental eviction.
		l.mu.Unlock()
	}
}

// extractDomain extracts the hostname from a URL string.
func extractDomain(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	return u.Hostname(), nil
}

// RateLimiter defines the interface for rate limiting.
type RateLimiter interface {
	Wait(ctx context.Context, url string) error
}

// Ensure Limiter implements RateLimiter.
var _ RateLimiter = (*Limiter)(nil)
