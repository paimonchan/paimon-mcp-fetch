// Package domain contains enterprise business rules for paimon-mcp-fetch.
package domain

import (
	"context"
	"net/http"
	"time"
)

// ContentFetcher fetches raw HTTP content from a URL.
type ContentFetcher interface {
	Fetch(ctx context.Context, url string, opts FetchOptions) (*FetchResponse, error)
}

// FetchOptions configures the HTTP fetch behavior.
type FetchOptions struct {
	UserAgent     string
	Timeout       time.Duration
	MaxRedirects  int
	MaxHTMLBytes  int64
	MaxImageBytes int64
}

// FetchResponse is the raw HTTP response.
type FetchResponse struct {
	StatusCode  int
	Headers     http.Header
	Body        []byte
	ContentType string
	FinalURL    string
}

// ContentExtractor converts HTML to cleaned content.
type ContentExtractor interface {
	Extract(ctx context.Context, html string, url string) (*ExtractedContent, error)
}

// ExtractedContent holds the result of HTML extraction.
type ExtractedContent struct {
	Title    string
	Content  string // cleaned HTML
	Markdown string // converted markdown
	Images   []ImageRef
}

// ImageRef is a reference to an image found in the content.
type ImageRef struct {
	Src      string
	Alt      string
	Filename string
}

// ImageProcessor downloads and processes images.
type ImageProcessor interface {
	FetchAndProcess(ctx context.Context, images []ImageRef, baseOrigin string, opts ImageProcessOptions) ([]ImageResult, error)
}

// ImageProcessOptions configures image processing.
type ImageProcessOptions struct {
	MaxCount      int
	MaxWidth      int
	MaxHeight     int
	Quality       int
	StartIndex    int
	CrossOrigin   bool
	SaveDir       string
	OutputBase64  bool
	SaveToFile    bool
	Layout        string // "merged", "individual", "both"
	MaxBytes      int64  // max image size in bytes
}

// RobotsChecker validates URLs against robots.txt.
type RobotsChecker interface {
	IsAllowed(ctx context.Context, url string, userAgent string) (bool, error)
}

// CacheEntry is a typed cache value.
type CacheEntry struct {
	Body        []byte
	ContentType string
	FinalURL    string
}

// CacheStore caches fetch responses.
type CacheStore interface {
	Get(ctx context.Context, key string) (*CacheEntry, bool, error)
	Set(ctx context.Context, key string, value *CacheEntry, ttl time.Duration) error
	Invalidate(ctx context.Context, key string) error
}
