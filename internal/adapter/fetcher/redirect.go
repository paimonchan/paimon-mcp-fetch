// Package fetcher provides HTTP fetching with SSRF protection and safe redirects.
package fetcher

import (
	"fmt"
	"net/http"

	"github.com/paimonchan/paimon-mcp-fetch/internal/domain"
)

// redirectTracker follows redirects with per-hop SSRF validation.
type redirectTracker struct {
	guard       *ssrfGuard
	maxRedirects int
	count       int
}

// newRedirectTracker creates a new redirect tracker.
func newRedirectTracker(guard *ssrfGuard, maxRedirects int) *redirectTracker {
	return &redirectTracker{
		guard:        guard,
		maxRedirects: maxRedirects,
	}
}

// checkRedirect returns a CheckRedirect function for net/http.Client.
// It validates each redirect target through the SSRF guard.
func (r *redirectTracker) checkRedirect(req *http.Request, via []*http.Request) error {
	r.count++
	if r.count > r.maxRedirects {
		return fmt.Errorf("%w: exceeded %d redirects", domain.ErrTooManyRedirects, r.maxRedirects)
	}

	// Validate the redirect target URL
	_, err := r.guard.validate(req.URL.String())
	if err != nil {
		return fmt.Errorf("redirect blocked by SSRF guard: %w", err)
	}

	return nil
}

// followed returns the number of redirects followed so far.
func (r *redirectTracker) followed() int {
	return r.count
}
