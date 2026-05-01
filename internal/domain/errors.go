// Package domain contains enterprise business rules for paimon-mcp-fetch.
package domain

import "errors"

// Sentinel errors for the fetch domain.
var (
	ErrInvalidURL          = errors.New("invalid URL format")
	ErrSchemeNotAllowed    = errors.New("only http and https schemes are allowed")
	ErrSSRFBlocked         = errors.New("URL resolves to private/reserved address")
	ErrLocalhostBlocked    = errors.New("localhost/local hostnames are not allowed")
	ErrRobotsTxtDisallowed = errors.New("disallowed by robots.txt")
	ErrRobotsTxtForbidden  = errors.New("robots.txt returned 401/403")
	ErrContentTooLarge     = errors.New("response exceeds size limit")
	ErrImageTooLarge       = errors.New("image exceeds size limit")
	ErrTimeout             = errors.New("request timed out")
	ErrTooManyRedirects    = errors.New("too many redirects")
	ErrFetchFailed         = errors.New("fetch failed")
	ErrHTTPClientError     = errors.New("HTTP client error") // 4xx, don't retry
	ErrHTTPServerError     = errors.New("HTTP server error") // 5xx, may retry
	ErrExtractionFailed    = errors.New("content extraction failed")
	ErrNoContent           = errors.New("no content could be extracted")
)

// ErrorType categorizes fetch errors for structured responses.
type ErrorType string

const (
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypeDNS        ErrorType = "dns"
	ErrorTypeSSRF       ErrorType = "ssrf"
	ErrorTypeRobots     ErrorType = "robots"
	ErrorTypeSize       ErrorType = "size"
	ErrorTypeHTTP       ErrorType = "http"
	ErrorTypeExtraction ErrorType = "extraction"
)

// FetchError is a structured error with categorization.
type FetchError struct {
	Type    ErrorType
	Message string
	URL     string
	Err     error
}

func (e *FetchError) Error() string { return e.Message }

// Unwrap returns the underlying error.
func (e *FetchError) Unwrap() error { return e.Err }
