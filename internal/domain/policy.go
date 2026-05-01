// Package domain contains enterprise business rules for paimon-mcp-fetch.
package domain

import "time"

// SSRFPolicy controls SSRF guard behavior.
type SSRFPolicy struct {
	AllowPrivateIPs bool
	AllowLocalhost  bool
	AllowedSchemes  []string
	BlockedHosts    []string
	AllowedHosts    []string
}

// DefaultSSRFPolicy returns the default SSRF policy.
func DefaultSSRFPolicy() SSRFPolicy {
	return SSRFPolicy{
		AllowPrivateIPs: false,
		AllowLocalhost:  false,
		AllowedSchemes:  []string{"http", "https"},
	}
}

// SizePolicy controls content size limits.
type SizePolicy struct {
	MaxHTMLBytes  int64
	MaxImageBytes int64
	MaxRedirects  int
	TimeoutMS     int
}

// DefaultSizePolicy returns the default size policy.
func DefaultSizePolicy() SizePolicy {
	return SizePolicy{
		MaxHTMLBytes:  10 * 1024 * 1024, // 10MB for JS-heavy finance sites
		MaxImageBytes: 10 * 1024 * 1024,
		MaxRedirects:  5,
		TimeoutMS:     12000,
	}
}

// CachePolicy controls caching behavior.
type CachePolicy struct {
	Enabled    bool
	DefaultTTL time.Duration
	MaxEntries int
}

// DefaultCachePolicy returns the default cache policy.
func DefaultCachePolicy() CachePolicy {
	return CachePolicy{
		Enabled:    true,
		DefaultTTL: 5 * time.Minute,
		MaxEntries: 100,
	}
}
