// Package main is the entry point for paimon-mcp-fetch.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/paimonchan/paimon-mcp-fetch/internal/adapter/cache"
	"github.com/paimonchan/paimon-mcp-fetch/internal/adapter/extractor"
	"github.com/paimonchan/paimon-mcp-fetch/internal/adapter/fetcher"
	"github.com/paimonchan/paimon-mcp-fetch/internal/adapter/image"
	"github.com/paimonchan/paimon-mcp-fetch/internal/adapter/mcpserver"
	"github.com/paimonchan/paimon-mcp-fetch/internal/adapter/ratelimit"
	"github.com/paimonchan/paimon-mcp-fetch/internal/adapter/robots"
	"github.com/paimonchan/paimon-mcp-fetch/internal/config"
	"github.com/paimonchan/paimon-mcp-fetch/internal/domain"
	"github.com/paimonchan/paimon-mcp-fetch/internal/usecase"
)

// version is set at build time via -ldflags "-X main.version=v0.1.0"
var version string

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	// Load configuration
	cfg := config.Load()

	logger.Info("starting paimon-mcp-fetch",
		"version", version,
		"timeout_ms", cfg.TimeoutMS,
		"max_redirects", cfg.MaxRedirects,
	)

	// Build domain policies
	sizePolicy := domain.SizePolicy{
		MaxHTMLBytes:  cfg.MaxHTMLBytes,
		MaxImageBytes: cfg.MaxImageBytes,
		MaxRedirects:  cfg.MaxRedirects,
		TimeoutMS:     cfg.TimeoutMS,
	}

	ssrfPolicy := domain.DefaultSSRFPolicy()
	if cfg.DisableSSRFGuard {
		ssrfPolicy.AllowPrivateIPs = true
		ssrfPolicy.AllowLocalhost = true
	}

	// Build adapters
	fetchOpts := domain.FetchOptions{
		UserAgent:     cfg.UserAgent,
		Timeout:       time.Duration(cfg.TimeoutMS) * time.Millisecond,
		MaxRedirects:  cfg.MaxRedirects,
		MaxHTMLBytes:  cfg.MaxHTMLBytes,
		MaxImageBytes: cfg.MaxImageBytes,
	}
	baseFetcher := fetcher.NewHTTPFetcher(fetchOpts, ssrfPolicy)
	contentFetcher := fetcher.NewRetryFetcher(
		baseFetcher,
		cfg.RetryMaxAttempts,
		time.Duration(cfg.RetryBaseDelayMS)*time.Millisecond,
		time.Duration(cfg.RetryMaxDelayMS)*time.Millisecond,
	)
	contentExtractor := extractor.NewReadabilityExtractor()
	robotsChecker := robots.NewChecker()

	// Optional: cache
	var cacheStore domain.CacheStore
	if cfg.CacheEnabled {
		mc, err := cache.NewMemoryCache(cfg.CacheMax, cfg.CacheTTL)
		if err != nil {
			logger.Error("failed to create cache", "error", err)
			os.Exit(1)
		}
		cacheStore = mc
		logger.Info("cache enabled", "max_entries", cfg.CacheMax, "ttl_secs", cfg.CacheTTL.Seconds())
	}

	// Optional: rate limiter
	var limiter usecase.RateLimiter
	if cfg.RateLimitEnabled {
		limiter = ratelimit.NewLimiter(
			time.Duration(1.0/cfg.RateLimitPerSecond*float64(time.Second)),
			cfg.RateLimitBurst,
		)
		logger.Info("rate limiter enabled", "per_second", cfg.RateLimitPerSecond, "burst", cfg.RateLimitBurst)
	}

	// Optional: image processor (enabled with `//go:build image`)
	imgProc := domain.ImageProcessor(image.NewProcessor(nil))

	// Build use case
	uc := usecase.NewFetchUseCase(
		contentFetcher,
		contentExtractor,
		imgProc,
		robotsChecker,
		cacheStore,
		limiter,
		sizePolicy,
		cfg.CacheTTL,
	)

	// Build MCP server
	mcpServer := mcpserver.NewServer(uc, cfg.ServerName, cfg.ServerVersion)

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		logger.Info("MCP server ready on stdio")
		errCh <- mcpServer.ServeStdio()
	}()

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		logger.Info("shutting down gracefully...")
	case err := <-errCh:
		if err != nil {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}
}
