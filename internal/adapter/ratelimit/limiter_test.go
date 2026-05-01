package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestLimiter_AllowsBurst(t *testing.T) {
	l := NewLimiter(100*time.Millisecond, 3)
	ctx := context.Background()

	// First 3 requests should be immediate (burst)
	start := time.Now()
	for i := 0; i < 3; i++ {
		if err := l.Wait(ctx, "https://example.com"); err != nil {
			t.Fatalf("request %d failed: %v", i+1, err)
		}
	}
	elapsed := time.Since(start)

	if elapsed > 50*time.Millisecond {
		t.Errorf("burst took too long: %v", elapsed)
	}
}

func TestLimiter_RateLimits(t *testing.T) {
	l := NewLimiter(100*time.Millisecond, 1)
	ctx := context.Background()

	// First request immediate
	if err := l.Wait(ctx, "https://example.com"); err != nil {
		t.Fatal(err)
	}

	// Second request should wait
	start := time.Now()
	if err := l.Wait(ctx, "https://example.com"); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)

	if elapsed < 50*time.Millisecond {
		t.Errorf("expected rate limit delay, got: %v", elapsed)
	}
}

func TestLimiter_PerDomainIsolation(t *testing.T) {
	l := NewLimiter(1*time.Second, 1)
	ctx := context.Background()

	// Request to example.com
	if err := l.Wait(ctx, "https://example.com/path"); err != nil {
		t.Fatal(err)
	}

	// Request to other.com should be immediate (different domain)
	start := time.Now()
	if err := l.Wait(ctx, "https://other.com/path"); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)

	if elapsed > 50*time.Millisecond {
		t.Errorf("different domain should not be rate limited, took: %v", elapsed)
	}
}

func TestLimiter_ContextCancellation(t *testing.T) {
	l := NewLimiter(1*time.Second, 1)
	ctx, cancel := context.WithCancel(context.Background())

	// First request consumes the token
	if err := l.Wait(ctx, "https://example.com"); err != nil {
		t.Fatal(err)
	}

	// Cancel context before second request
	cancel()

	err := l.Wait(ctx, "https://example.com")
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/path", "example.com"},
		{"https://example.com:8080/path", "example.com"},
		{"http://sub.example.com", "sub.example.com"},
		{"https://example.com?query=1", "example.com"},
	}

	for _, tc := range tests {
		got, err := extractDomain(tc.url)
		if err != nil {
			t.Fatalf("extractDomain(%q) error: %v", tc.url, err)
		}
		if got != tc.want {
			t.Errorf("extractDomain(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}
