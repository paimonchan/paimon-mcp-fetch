// Package robots provides robots.txt checking with fail-open behavior and caching.
package robots

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/user/paimon-mcp-fetch/internal/domain"
)

func TestIsAllowed_Allow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			fmt.Fprint(w, "User-agent: *\nDisallow: /private/\n")
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := NewChecker()
	checker.client = server.Client()

	allowed, err := checker.IsAllowed(context.Background(), server.URL+"/page", "TestBot")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected allowed for /page")
	}
}

func TestIsAllowed_Disallowed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			fmt.Fprint(w, "User-agent: *\nDisallow: /private/\n")
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := NewChecker()
	checker.client = server.Client()

	allowed, err := checker.IsAllowed(context.Background(), server.URL+"/private/secret", "TestBot")
	if err == nil {
		t.Fatal("expected error for disallowed path")
	}
	if allowed {
		t.Error("expected not allowed for /private/secret")
	}
	if !strings.Contains(err.Error(), domain.ErrRobotsTxtDisallowed.Error()) {
		t.Errorf("expected ErrRobotsTxtDisallowed, got: %v", err)
	}
}

func TestIsAllowed_404FailOpen(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := NewChecker()
	checker.client = server.Client()

	allowed, err := checker.IsAllowed(context.Background(), server.URL+"/page", "TestBot")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected allowed (fail-open) when robots.txt returns 404")
	}
}

func TestIsAllowed_403Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := NewChecker()
	checker.client = server.Client()

	allowed, err := checker.IsAllowed(context.Background(), server.URL+"/page", "TestBot")
	if err == nil {
		t.Fatal("expected error for 403 robots.txt")
	}
	if allowed {
		t.Error("expected not allowed when robots.txt returns 403")
	}
	if !strings.Contains(err.Error(), domain.ErrRobotsTxtForbidden.Error()) {
		t.Errorf("expected ErrRobotsTxtForbidden, got: %v", err)
	}
}

func TestIsAllowed_500FailOpen(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	checker := NewChecker()
	checker.client = server.Client()

	allowed, err := checker.IsAllowed(context.Background(), server.URL+"/page", "TestBot")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected allowed (fail-open) when robots.txt returns 500")
	}
}

func TestIsAllowed_Cache(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			callCount++
			fmt.Fprint(w, "User-agent: *\nDisallow: /private/\n")
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := NewChecker()
	checker.client = server.Client()

	// First call → fetch from server
	_, _ = checker.IsAllowed(context.Background(), server.URL+"/page", "TestBot")
	if callCount != 1 {
		t.Fatalf("expected 1 server call, got %d", callCount)
	}

	// Second call → should use cache
	_, _ = checker.IsAllowed(context.Background(), server.URL+"/page2", "TestBot")
	if callCount != 1 {
		t.Fatalf("expected 1 server call (cached), got %d", callCount)
	}

	if checker.Len() != 1 {
		t.Fatalf("expected 1 cached host, got %d", checker.Len())
	}
}

func TestIsAllowed_CacheExpiration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			fmt.Fprint(w, "User-agent: *\nDisallow: /private/\n")
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := NewChecker()
	checker.client = server.Client()

	_, _ = checker.IsAllowed(context.Background(), server.URL+"/page", "TestBot")
	if checker.Len() != 1 {
		t.Fatalf("expected 1 cached host, got %d", checker.Len())
	}

	// Expire the cache entry
	checker.mu.Lock()
	for k, v := range checker.cache {
		checker.cache[k] = &cacheEntry{
			data:      v.data,
			expiresAt: time.Now().Add(-1 * time.Second),
		}
	}
	checker.mu.Unlock()

	// Next call should refetch
	_, _ = checker.IsAllowed(context.Background(), server.URL+"/page2", "TestBot")
	// Should still work without error
}

func TestGetRobotsURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/page", "https://example.com/robots.txt"},
		{"https://example.com:8080/path?query=1", "https://example.com:8080/robots.txt"},
		{"http://example.com", "http://example.com/robots.txt"},
	}

	for _, tc := range tests {
		got, err := getRobotsURL(tc.input)
		if err != nil {
			t.Errorf("getRobotsURL(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if got.String() != tc.expected {
			t.Errorf("getRobotsURL(%q) = %q, want %q", tc.input, got.String(), tc.expected)
		}
	}
}

func TestInvalidate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			fmt.Fprint(w, "User-agent: *\nDisallow: /private/\n")
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := NewChecker()
	checker.client = server.Client()

	_, _ = checker.IsAllowed(context.Background(), server.URL+"/page", "TestBot")
	if checker.Len() != 1 {
		t.Fatalf("expected 1 cached host, got %d", checker.Len())
	}

	u, _ := getRobotsURL(server.URL + "/page")
	checker.Invalidate(u.Host)

	if checker.Len() != 0 {
		t.Fatalf("expected 0 cached hosts after invalidate, got %d", checker.Len())
	}
}
