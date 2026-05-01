// Package config loads and validates server configuration from environment variables.
package config

import (
	"os"
	"strconv"
	"time"
)

const envPrefix = "PAIMON_MCP_FETCH_"

// Config holds all server configuration.
type Config struct {
	ServerName    string
	ServerVersion string

	// Fetcher
	TimeoutMS     int
	MaxRedirects  int
	MaxHTMLBytes  int64
	MaxImageBytes int64
	UserAgent     string

	// SSRF
	DisableSSRFGuard bool

	// JS Rendering
	JSRenderEnabled bool

	// Cache
	CacheEnabled bool
	CacheTTL     time.Duration
	CacheMax     int

	// Rate Limiting
	RateLimitEnabled   bool
	RateLimitPerSecond float64
	RateLimitBurst     int

	// Retry
	RetryMaxAttempts int
	RetryBaseDelayMS int
	RetryMaxDelayMS  int

	// Image (optional)
	ImageEnabled        bool
	ImageDefaultMax     int
	ImageDefaultWidth   int
	ImageDefaultHeight  int
	ImageDefaultQuality int

	// Output
	DefaultMaxLength int
	DefaultRaw       bool
}

// Default returns a Config with all default values.
func Default() *Config {
	return &Config{
		ServerName:    "paimon-mcp-fetch",
		ServerVersion: "0.1.0",

		TimeoutMS:     12000,
		MaxRedirects:  5,
		MaxHTMLBytes:  10 * 1024 * 1024,
		MaxImageBytes: 10 * 1024 * 1024,
		UserAgent:     "ModelContextProtocol/1.0 (Autonomous; +https://github.com/paimonchan/paimon-mcp-fetch)",

		DisableSSRFGuard: false,

		JSRenderEnabled: false,

		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
		CacheMax:     100,

		RateLimitEnabled:   true,
		RateLimitPerSecond: 1.0,
		RateLimitBurst:     3,

		RetryMaxAttempts: 3,
		RetryBaseDelayMS: 500,
		RetryMaxDelayMS:  10000,

		ImageEnabled:        false,
		ImageDefaultMax:     3,
		ImageDefaultWidth:   1000,
		ImageDefaultHeight:  1600,
		ImageDefaultQuality: 80,

		DefaultMaxLength: 20000,
		DefaultRaw:       false,
	}
}

// Load reads configuration from environment variables with defaults.
func Load() *Config {
	cfg := Default()

	if v := getInt("TIMEOUT_MS"); v != nil {
		cfg.TimeoutMS = *v
	}
	if v := getInt("MAX_REDIRECTS"); v != nil {
		cfg.MaxRedirects = *v
	}
	if v := getInt64("MAX_HTML_BYTES"); v != nil {
		cfg.MaxHTMLBytes = *v
	}
	if v := getInt64("MAX_IMAGE_BYTES"); v != nil {
		cfg.MaxImageBytes = *v
	}
	if v := os.Getenv(envPrefix + "USER_AGENT"); v != "" {
		cfg.UserAgent = v
	}
	if v := getBool("DISABLE_SSRF"); v != nil {
		cfg.DisableSSRFGuard = *v
	}
	if v := getBool("JS_RENDER_ENABLED"); v != nil {
		cfg.JSRenderEnabled = *v
	}
	if v := getBool("CACHE_ENABLED"); v != nil {
		cfg.CacheEnabled = *v
	}
	if v := getInt("CACHE_TTL_SECS"); v != nil {
		cfg.CacheTTL = time.Duration(*v) * time.Second
	}
	if v := getInt("CACHE_MAX_ENTRIES"); v != nil {
		cfg.CacheMax = *v
	}
	if v := getBool("RATE_LIMIT_ENABLED"); v != nil {
		cfg.RateLimitEnabled = *v
	}
	if v := getFloat("RATE_LIMIT_PER_SECOND"); v != nil {
		cfg.RateLimitPerSecond = *v
	}
	if v := getInt("RATE_LIMIT_BURST"); v != nil {
		cfg.RateLimitBurst = *v
	}
	if v := getInt("RETRY_MAX_ATTEMPTS"); v != nil {
		cfg.RetryMaxAttempts = *v
	}
	if v := getInt("RETRY_BASE_DELAY_MS"); v != nil {
		cfg.RetryBaseDelayMS = *v
	}
	if v := getInt("RETRY_MAX_DELAY_MS"); v != nil {
		cfg.RetryMaxDelayMS = *v
	}
	if v := getBool("IMAGE_ENABLED"); v != nil {
		cfg.ImageEnabled = *v
	}
	if v := getInt("IMAGE_DEFAULT_MAX"); v != nil {
		cfg.ImageDefaultMax = *v
	}
	if v := getInt("IMAGE_DEFAULT_WIDTH"); v != nil {
		cfg.ImageDefaultWidth = *v
	}
	if v := getInt("IMAGE_DEFAULT_HEIGHT"); v != nil {
		cfg.ImageDefaultHeight = *v
	}
	if v := getInt("IMAGE_DEFAULT_QUALITY"); v != nil {
		cfg.ImageDefaultQuality = *v
	}
	if v := getInt("DEFAULT_MAX_LENGTH"); v != nil {
		cfg.DefaultMaxLength = *v
	}
	if v := getBool("DEFAULT_RAW"); v != nil {
		cfg.DefaultRaw = *v
	}

	return cfg
}

func getInt(key string) *int {
	s := os.Getenv(envPrefix + key)
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}

func getInt64(key string) *int64 {
	s := os.Getenv(envPrefix + key)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

func getBool(key string) *bool {
	s := os.Getenv(envPrefix + key)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return nil
	}
	return &v
}

func getFloat(key string) *float64 {
	s := os.Getenv(envPrefix + key)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}
