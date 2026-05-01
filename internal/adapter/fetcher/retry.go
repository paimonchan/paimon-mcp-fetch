// Package fetcher provides HTTP fetching with SSRF protection and safe redirects.
package fetcher

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/user/paimon-mcp-fetch/internal/domain"
)

// retryFetcher wraps a ContentFetcher with retry logic.
type retryFetcher struct {
	inner      domain.ContentFetcher
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
}

// NewRetryFetcher wraps a ContentFetcher with exponential backoff retry.
// Retries on transient errors: timeout, 5xx status, temporary network errors.
func NewRetryFetcher(inner domain.ContentFetcher, maxRetries int, baseDelay, maxDelay time.Duration) domain.ContentFetcher {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if baseDelay <= 0 {
		baseDelay = 500 * time.Millisecond
	}
	if maxDelay <= 0 {
		maxDelay = 10 * time.Second
	}
	return &retryFetcher{
		inner:      inner,
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
		maxDelay:   maxDelay,
	}
}

// Fetch implements domain.ContentFetcher with retry logic.
func (r *retryFetcher) Fetch(ctx context.Context, url string, opts domain.FetchOptions) (*domain.FetchResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		resp, err := r.inner.Fetch(ctx, url, opts)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Don't retry if context is cancelled
		if ctx.Err() != nil {
			return nil, err
		}

		// Don't retry on permanent errors
		if !isRetryable(err) {
			return nil, err
		}

		// Don't sleep after the last attempt
		if attempt < r.maxRetries {
			delay := r.calculateDelay(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				// Retry
			}
		}
	}

	return nil, fmt.Errorf("fetch failed after %d retries: %w", r.maxRetries, lastErr)
}

// calculateDelay computes exponential backoff with jitter.
func (r *retryFetcher) calculateDelay(attempt int) time.Duration {
	// Exponential: baseDelay * 2^attempt
	delay := r.baseDelay * time.Duration(math.Pow(2, float64(attempt)))
	if delay > r.maxDelay {
		delay = r.maxDelay
	}
	// Add jitter: ±25%
	jitter := time.Duration(float64(delay) * 0.25)
	if jitter > 0 {
		// Simple jitter: subtract up to 25%
		delay = delay - jitter/2
	}
	return delay
}

// isRetryable determines if an error is transient and worth retrying.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Never retry SSRF or robots.txt errors
	if errors.Is(err, domain.ErrSSRFBlocked) ||
		errors.Is(err, domain.ErrLocalhostBlocked) ||
		errors.Is(err, domain.ErrRobotsTxtDisallowed) ||
		errors.Is(err, domain.ErrRobotsTxtForbidden) {
		return false
	}

	// Retry timeouts (might be transient)
	if errors.Is(err, domain.ErrTimeout) {
		return true
	}

	// Retry content too large (might be a temporary issue)
	if errors.Is(err, domain.ErrContentTooLarge) {
		return false // Actually, don't retry — it'll always be too large
	}

	// Retry fetch failures (network issues, 5xx, etc.)
	if errors.Is(err, domain.ErrFetchFailed) {
		return true
	}

	// Default: retry unknown errors (network hiccups)
	return true
}
