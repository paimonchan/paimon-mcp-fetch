// Package robots provides robots.txt checking with fail-open behavior and caching.
package robots

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
	"github.com/paimonchan/paimon-mcp-fetch/internal/domain"
)

const (
	robotsTimeout  = 5 * time.Second
	cacheTTL       = 1 * time.Hour
	robotsPath     = "/robots.txt"
	maxRobotsBytes = 100_000 // 100KB limit for robots.txt
)

// cacheEntry holds parsed robots.txt data with expiration.
type cacheEntry struct {
	data      *robotstxt.RobotsData
	expiresAt time.Time
}

// Checker implements domain.RobotsChecker with per-host caching.
type Checker struct {
	client *http.Client
	cache  map[string]*cacheEntry
	mu     sync.RWMutex
}

// NewChecker creates a new robots.txt checker.
func NewChecker() *Checker {
	return &Checker{
		client: &http.Client{
			Timeout:   robotsTimeout,
			Transport: &http.Transport{DisableKeepAlives: true},
		},
		cache: make(map[string]*cacheEntry),
	}
}

// IsAllowed checks if the given URL is allowed by the site's robots.txt.
// Implements fail-open: any fetch error (timeout, DNS, 5xx) → allowed.
// Only blocks on explicit Disallowed or 401/403 responses.
func (c *Checker) IsAllowed(ctx context.Context, pageURL, userAgent string) (bool, error) {
	robotsURL, err := getRobotsURL(pageURL)
	if err != nil {
		// Can't parse URL → allow (fail-open)
		return true, nil
	}

	host := robotsURL.Host

	// 1. Check cache
	c.mu.RLock()
	entry, found := c.cache[host]
	c.mu.RUnlock()

	if found && time.Now().Before(entry.expiresAt) {
		return c.test(entry.data, pageURL, userAgent)
	}

	// 2. Fetch robots.txt
	data, err := c.fetch(ctx, robotsURL)
	if err != nil {
		// Distinguish between "hard block" (401/403) and "fail-open" (network/5xx errors)
		if err == domain.ErrRobotsTxtForbidden {
			return false, err
		}
		// All other errors (timeout, DNS, 5xx) → fail-open (allow)
		return true, nil
	}

	// 3. Cache result
	c.mu.Lock()
	c.cache[host] = &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(cacheTTL),
	}
	c.mu.Unlock()

	// 4. Test against parsed rules
	return c.test(data, pageURL, userAgent)
}

// fetch retrieves and parses robots.txt with custom status handling.
func (c *Checker) fetch(ctx context.Context, robotsURL *url.URL) (*robotstxt.RobotsData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Custom status handling (fail-open philosophy)
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		// 401/403 → treat as forbidden for autonomous fetching
		return nil, domain.ErrRobotsTxtForbidden

	case resp.StatusCode >= 500:
		// 5xx → fail-open (allow). Override library default which is disallowAll.
		return robotstxt.FromString("User-agent: *\nAllow: /\n")

	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		// Other 4xx (404, etc.) → allow (Google spec)
		return robotstxt.FromStatusAndBytes(resp.StatusCode, nil)

	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		// 2xx → parse body
		return robotstxt.FromResponse(resp)

	default:
		// Unknown status → fail-open
		return robotstxt.FromStatusAndBytes(resp.StatusCode, nil)
	}
}

// test checks if a path is allowed for the given user agent.
func (c *Checker) test(data *robotstxt.RobotsData, pageURL, userAgent string) (bool, error) {
	if data == nil {
		return true, nil
	}

	u, err := url.Parse(pageURL)
	if err != nil {
		return true, nil
	}

	allowed := data.TestAgent(u.Path, userAgent)
	if !allowed {
		return false, fmt.Errorf("%w for URL %s", domain.ErrRobotsTxtDisallowed, pageURL)
	}
	return true, nil
}

// getRobotsURL extracts the robots.txt URL from a page URL.
func getRobotsURL(pageURL string) (*url.URL, error) {
	u, err := url.Parse(pageURL)
	if err != nil {
		return nil, err
	}
	u.Path = robotsPath
	u.RawQuery = ""
	u.Fragment = ""
	return u, nil
}

// Invalidate removes a host from the cache.
func (c *Checker) Invalidate(host string) {
	c.mu.Lock()
	delete(c.cache, host)
	c.mu.Unlock()
}

// Clear removes all entries from the cache.
func (c *Checker) Clear() {
	c.mu.Lock()
	c.cache = make(map[string]*cacheEntry)
	c.mu.Unlock()
}

// Len returns the number of cached hosts.
func (c *Checker) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}
