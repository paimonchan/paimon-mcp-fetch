//go:build jsrender

// Package jsrender provides headless Chrome rendering for JS-heavy websites.
// This package requires Chrome/Chromium to be installed on the system.
// Use build tag `jsrender` to compile with JS rendering support.
package jsrender

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/paimonchan/paimon-mcp-fetch/internal/domain"
)

// Renderer implements domain.ContentFetcher using headless Chrome.
type Renderer struct {
	timeout time.Duration
}

// NewRenderer creates a new JS renderer.
func NewRenderer(timeout time.Duration) *Renderer {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Renderer{timeout: timeout}
}

// Fetch renders a URL in headless Chrome and returns the final HTML.
func (r *Renderer) Fetch(ctx context.Context, urlStr string, opts domain.FetchOptions) (*domain.FetchResponse, error) {
	// Validate URL first
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, domain.ErrSchemeNotAllowed
	}

	// Create chromedp context with timeout
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Use headless Chrome with minimal flags
	allocCtx, allocCancel := chromedp.NewContext(ctx)
	defer allocCancel()

	var html string
	var statusCode int

	// Navigate and wait for page to be stable
	err = chromedp.Run(allocCtx,
		chromedp.Navigate(urlStr),
		// Wait for network to be idle (no new requests for 500ms)
		chromedp.WaitReady("body", chromedp.ByQuery),
		// Optional: wait a bit more for JS to execute
		chromedp.Sleep(2*time.Second),
		// Get the rendered HTML
		chromedp.OuterHTML("html", &html),
		// Get status code from Performance API
		chromedp.Evaluate(`window.performance && window.performance.getEntriesByType('navigation')[0] ? window.performance.getEntriesByType('navigation')[0].responseStatus : 200`, &statusCode),
	)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, domain.ErrTimeout
		}
		if strings.Contains(err.Error(), "net::ERR_CONNECTION_REFUSED") ||
			strings.Contains(err.Error(), "net::ERR_NAME_NOT_RESOLVED") {
			return nil, fmt.Errorf("%w: %v", domain.ErrFetchFailed, err)
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrFetchFailed, err)
	}

	// Enforce byte limit
	if int64(len(html)) > opts.MaxHTMLBytes {
		return nil, domain.ErrContentTooLarge
	}

	return &domain.FetchResponse{
		StatusCode:  statusCode,
		Body:        []byte(html),
		ContentType: "text/html",
		FinalURL:    urlStr,
	}, nil
}
