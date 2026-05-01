// Package usecase contains application business rules for paimon-mcp-fetch.
package usecase

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"time"

	"github.com/paimonchan/paimon-mcp-fetch/internal/domain"
)

// RateLimiter defines the interface for rate limiting.
type RateLimiter interface {
	Wait(ctx context.Context, url string) error
}

// FetchUseCase orchestrates the fetch flow.
type FetchUseCase struct {
	fetcher     domain.ContentFetcher
	extractor   domain.ContentExtractor
	imgProc     domain.ImageProcessor  // optional, can be nil
	robots      domain.RobotsChecker
	cache       domain.CacheStore      // optional, can be nil
	limiter     RateLimiter            // optional, can be nil
	policy      domain.SizePolicy
	cacheTTL    time.Duration
}

// NewFetchUseCase creates a FetchUseCase with constructor injection.
func NewFetchUseCase(
	fetcher domain.ContentFetcher,
	extractor domain.ContentExtractor,
	imgProc domain.ImageProcessor,
	robots domain.RobotsChecker,
	cache domain.CacheStore,
	limiter RateLimiter,
	policy domain.SizePolicy,
	cacheTTL time.Duration,
) *FetchUseCase {
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}
	return &FetchUseCase{
		fetcher:   fetcher,
		extractor: extractor,
		imgProc:   imgProc,
		robots:    robots,
		cache:     cache,
		limiter:   limiter,
		policy:    policy,
		cacheTTL:  cacheTTL,
	}
}

// Fetch orchestrates the complete fetch pipeline:
// 1. Validate request
// 2. Check cache
// 3. Check robots.txt
// 4. Fetch URL
// 5. Extract content
// 6. Process images (if enabled)
// 7. Apply pagination
// 8. Store in cache
// 9. Return result
func (uc *FetchUseCase) Fetch(ctx context.Context, req domain.FetchRequest) (*domain.FetchResult, error) {
	// 1. Validate request
	if err := uc.validateRequest(&req); err != nil {
		return nil, err
	}

	// 2. Check cache
	cacheKey := uc.cacheKey(req.URL)
	if uc.cache != nil {
		entry, found, err := uc.cache.Get(ctx, cacheKey)
		if err == nil && found {
			// Cache hit — extract from cached data
			return uc.buildResultFromCache(ctx, entry, req)
		}
	}

	// Apply rate limiting
	if uc.limiter != nil {
		if err := uc.limiter.Wait(ctx, req.URL); err != nil {
			return nil, fmt.Errorf("rate limit: %w", err)
		}
	}

	// Build fetch options
	fetchOpts := domain.FetchOptions{
		UserAgent:    uc.defaultUserAgent(),
		Timeout:      time.Duration(uc.policy.TimeoutMS) * time.Millisecond,
		MaxRedirects: uc.policy.MaxRedirects,
		MaxHTMLBytes: uc.policy.MaxHTMLBytes,
	}

	// 3. Check robots.txt (if not ignored)
	if !req.Security.IgnoreRobotsTxt && uc.robots != nil {
		allowed, err := uc.robots.IsAllowed(ctx, req.URL, fetchOpts.UserAgent)
		if err != nil {
			// Robots error — return it
			return nil, err
		}
		if !allowed {
			return nil, domain.ErrRobotsTxtDisallowed
		}
	}

	// 4. Fetch URL
	resp, err := uc.fetcher.Fetch(ctx, req.URL, fetchOpts)
	if err != nil {
		return nil, err
	}

	// 5. Extract content
	bodyStr := string(resp.Body)
	extracted, err := uc.extractor.Extract(ctx, bodyStr, resp.FinalURL)
	if err != nil {
		return nil, err
	}

	// 6. Process images (if enabled)
	var images []domain.ImageResult
	if req.Images.Enable && uc.imgProc != nil && len(extracted.Images) > 0 {
		imgOpts := domain.ImageProcessOptions{
			MaxCount:     req.Images.MaxCount,
			MaxWidth:     req.Images.MaxWidth,
			MaxHeight:    req.Images.MaxHeight,
			Quality:      req.Images.Quality,
			StartIndex:   req.Images.StartIndex,
			CrossOrigin:  req.Images.AllowCrossOrigin,
			SaveDir:      req.Images.SaveDir,
			OutputBase64: req.Images.OutputBase64,
			SaveToFile:   req.Images.SaveToFile,
			Layout:       req.Images.Layout,
			MaxBytes:     uc.policy.MaxImageBytes,
		}
		baseOrigin := ""
		if u, err := parseOrigin(resp.FinalURL); err == nil {
			baseOrigin = u
		}
		images, err = uc.imgProc.FetchAndProcess(ctx, extracted.Images, baseOrigin, imgOpts)
		if err != nil {
			// Image processing error is non-fatal — continue without images
			images = nil
		}
	}

	// 7. Apply pagination
	content := extracted.Markdown
	if req.Text.Raw {
		content = extracted.Content
	}

	paginatedContent, remainingContent := uc.paginate(content, req.Text.StartIndex, req.Text.MaxLength)

	// 8. Store in cache
	if uc.cache != nil {
		entry := &domain.CacheEntry{
			Body:        resp.Body,
			ContentType: resp.ContentType,
			FinalURL:    resp.FinalURL,
		}
		_ = uc.cache.Set(ctx, cacheKey, entry, uc.cacheTTL) // best-effort
	}

	// 9. Return result
	result := &domain.FetchResult{
		Title:            extracted.Title,
		Content:          paginatedContent,
		ContentType:      domain.ContentTypeMarkdown,
		Images:           images,
		RemainingContent: remainingContent,
		RemainingImages:  uc.remainingImages(extracted.Images, req.Images.StartIndex, req.Images.MaxCount),
	}

	if req.Text.Raw {
		result.ContentType = domain.ContentTypeRaw
	}

	return result, nil
}

// validateRequest validates the fetch request parameters.
func (uc *FetchUseCase) validateRequest(req *domain.FetchRequest) error {
	if req.URL == "" {
		return fmt.Errorf("%w: URL is required", domain.ErrInvalidURL)
	}
	if req.Text.MaxLength < 0 || req.Text.MaxLength > 1_000_000 {
		return fmt.Errorf("%w: maxLength must be between 0 and 1,000,000", domain.ErrInvalidURL)
	}
	if req.Images.MaxCount < 0 || req.Images.MaxCount > 10 {
		return fmt.Errorf("%w: imageMaxCount must be between 0 and 10", domain.ErrInvalidURL)
	}
	if req.Images.Quality < 1 || req.Images.Quality > 100 {
		return fmt.Errorf("%w: imageQuality must be between 1 and 100", domain.ErrInvalidURL)
	}
	if req.Images.MaxWidth < 100 || req.Images.MaxWidth > 10000 {
		return fmt.Errorf("%w: imageMaxWidth must be between 100 and 10000", domain.ErrInvalidURL)
	}
	if req.Images.MaxHeight < 100 || req.Images.MaxHeight > 10000 {
		return fmt.Errorf("%w: imageMaxHeight must be between 100 and 10000", domain.ErrInvalidURL)
	}
	return nil
}

// paginate slices content from startIndex with maxLength.
func (uc *FetchUseCase) paginate(content string, startIndex, maxLength int) (string, int) {
	if startIndex >= len(content) {
		return "", 0
	}

	endIndex := startIndex + maxLength
	if endIndex > len(content) || maxLength == 0 {
		endIndex = len(content)
	}

	result := content[startIndex:endIndex]
	remaining := len(content) - endIndex

	return result, remaining
}

// remainingImages calculates how many images remain after pagination.
func (uc *FetchUseCase) remainingImages(allImages []domain.ImageRef, startIndex, maxCount int) int {
	if startIndex >= len(allImages) {
		return 0
	}
	shown := maxCount
	if startIndex+shown > len(allImages) {
		shown = len(allImages) - startIndex
	}
	return len(allImages) - startIndex - shown
}

// cacheKey generates a cache key from the URL.
func (uc *FetchUseCase) cacheKey(url string) string {
	h := sha256.New()
	h.Write([]byte(url))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// buildResultFromCache rebuilds a result from cached data.
func (uc *FetchUseCase) buildResultFromCache(ctx context.Context, entry *domain.CacheEntry, req domain.FetchRequest) (*domain.FetchResult, error) {
	// Re-extract from cached body
	extracted, err := uc.extractor.Extract(ctx, string(entry.Body), entry.FinalURL)
	if err != nil {
		return nil, err
	}

	content := extracted.Markdown
	if req.Text.Raw {
		content = extracted.Content
	}

	paginatedContent, remainingContent := uc.paginate(content, req.Text.StartIndex, req.Text.MaxLength)

	return &domain.FetchResult{
		Title:            extracted.Title,
		Content:          paginatedContent,
		ContentType:      domain.ContentTypeMarkdown,
		Images:           nil, // Don't re-process images from cache in MVP
		RemainingContent: remainingContent,
		RemainingImages:  uc.remainingImages(extracted.Images, req.Images.StartIndex, req.Images.MaxCount),
	}, nil
}

// defaultUserAgent returns the default user agent string.
func (uc *FetchUseCase) defaultUserAgent() string {
	return "ModelContextProtocol/1.0 (Autonomous; +https://github.com/paimonchan/paimon-mcp-fetch)"
}

// parseOrigin extracts the origin (scheme + host) from a URL.
func parseOrigin(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid URL: missing scheme or host")
	}
	return u.Scheme + "://" + u.Host, nil
}
