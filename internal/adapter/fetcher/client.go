// Package fetcher provides HTTP fetching with SSRF protection and safe redirects.
package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/paimonchan/paimon-mcp-fetch/internal/domain"
)

// httpFetcher implements domain.ContentFetcher using net/http.
type httpFetcher struct {
	client *http.Client
	guard  *ssrfGuard
}

// NewHTTPFetcher creates a new ContentFetcher with SSRF protection.
func NewHTTPFetcher(opts domain.FetchOptions, policy domain.SSRFPolicy) domain.ContentFetcher {
	guard := newSSRFGuard(policy)
	tracker := newRedirectTracker(guard, opts.MaxRedirects)

	client := &http.Client{
		Timeout: opts.Timeout,
		CheckRedirect: tracker.checkRedirect,
	}

	return &httpFetcher{
		client: client,
		guard:  guard,
	}
}

// Fetch retrieves content from the given URL with SSRF protection and size limits.
func (f *httpFetcher) Fetch(ctx context.Context, url string, opts domain.FetchOptions) (*domain.FetchResponse, error) {
	// Validate URL through SSRF guard
	parsedURL, err := f.guard.validate(url)
	if err != nil {
		return nil, err
	}

	// Build request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrFetchFailed, err)
	}

	if opts.UserAgent != "" {
		req.Header.Set("User-Agent", opts.UserAgent)
	}

	// Execute request
	resp, err := f.client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, domain.ErrTimeout
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrFetchFailed, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode >= 400 {
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("%w: HTTP %d", domain.ErrHTTPServerError, resp.StatusCode)
		}
		return nil, fmt.Errorf("%w: HTTP %d", domain.ErrHTTPClientError, resp.StatusCode)
	}

	// Stream-based reading with size limit
	body, err := f.readBody(resp.Body, opts.MaxHTMLBytes)
	if err != nil {
		return nil, err
	}

	return &domain.FetchResponse{
		StatusCode:  resp.StatusCode,
		Headers:     resp.Header,
		Body:        body,
		ContentType: resp.Header.Get("Content-Type"),
		FinalURL:    resp.Request.URL.String(),
	}, nil
}

// readBody reads from the response body with a byte limit.
func (f *httpFetcher) readBody(body io.ReadCloser, maxBytes int64) ([]byte, error) {
	// Check content-length header first
	// (This is a hint, but we'll still enforce the limit during reading)

	// Use a LimitedReader to enforce the byte limit
	lr := &io.LimitedReader{R: body, N: maxBytes + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrFetchFailed, err)
	}

	// Check if we hit the limit
	if lr.N <= 0 {
		return nil, domain.ErrContentTooLarge
	}

	return data, nil
}
