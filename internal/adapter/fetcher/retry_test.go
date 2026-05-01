package fetcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/paimonchan/paimon-mcp-fetch/internal/domain"
)

// mockFetcher is a test double for ContentFetcher.
type mockFetcher struct {
	results   []mockResult
	callCount int
}

type mockResult struct {
	resp *domain.FetchResponse
	err  error
}

func (m *mockFetcher) Fetch(ctx context.Context, url string, opts domain.FetchOptions) (*domain.FetchResponse, error) {
	if m.callCount >= len(m.results) {
		return nil, domain.ErrFetchFailed
	}
	result := m.results[m.callCount]
	m.callCount++
	return result.resp, result.err
}

func TestRetryFetcher_SuccessOnFirst(t *testing.T) {
	inner := &mockFetcher{
		results: []mockResult{{resp: &domain.FetchResponse{StatusCode: 200}}},
	}
	rf := NewRetryFetcher(inner, 3, 10*time.Millisecond, 100*time.Millisecond)

	resp, err := rf.Fetch(context.Background(), "https://example.com", domain.FetchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if inner.callCount != 1 {
		t.Errorf("callCount = %d, want 1", inner.callCount)
	}
}

func TestRetryFetcher_SuccessAfterRetry(t *testing.T) {
	inner := &mockFetcher{
		results: []mockResult{
			{err: domain.ErrFetchFailed},
			{err: domain.ErrFetchFailed},
			{resp: &domain.FetchResponse{StatusCode: 200}},
		},
	}
	rf := NewRetryFetcher(inner, 3, 10*time.Millisecond, 100*time.Millisecond)

	resp, err := rf.Fetch(context.Background(), "https://example.com", domain.FetchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if inner.callCount != 3 {
		t.Errorf("callCount = %d, want 3", inner.callCount)
	}
}

func TestRetryFetcher_MaxRetriesExceeded(t *testing.T) {
	inner := &mockFetcher{
		results: []mockResult{
			{err: domain.ErrFetchFailed},
			{err: domain.ErrFetchFailed},
			{err: domain.ErrFetchFailed},
			{err: domain.ErrFetchFailed},
		},
	}
	rf := NewRetryFetcher(inner, 2, 10*time.Millisecond, 100*time.Millisecond)

	_, err := rf.Fetch(context.Background(), "https://example.com", domain.FetchOptions{})
	if err == nil {
		t.Fatal("expected error after max retries")
	}
	if inner.callCount != 3 { // initial + 2 retries
		t.Errorf("callCount = %d, want 3", inner.callCount)
	}
}

func TestRetryFetcher_NoRetryOnSSRF(t *testing.T) {
	inner := &mockFetcher{
		results: []mockResult{{err: domain.ErrSSRFBlocked}},
	}
	rf := NewRetryFetcher(inner, 3, 10*time.Millisecond, 100*time.Millisecond)

	_, err := rf.Fetch(context.Background(), "https://example.com", domain.FetchOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, domain.ErrSSRFBlocked) {
		t.Errorf("expected ErrSSRFBlocked, got: %v", err)
	}
	if inner.callCount != 1 {
		t.Errorf("callCount = %d, want 1 (no retries for SSRF)", inner.callCount)
	}
}

func TestRetryFetcher_NoRetryOnRobots(t *testing.T) {
	inner := &mockFetcher{
		results: []mockResult{{err: domain.ErrRobotsTxtDisallowed}},
	}
	rf := NewRetryFetcher(inner, 3, 10*time.Millisecond, 100*time.Millisecond)

	_, err := rf.Fetch(context.Background(), "https://example.com", domain.FetchOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if inner.callCount != 1 {
		t.Errorf("callCount = %d, want 1 (no retries for robots.txt)", inner.callCount)
	}
}

func TestRetryFetcher_ContextCancellation(t *testing.T) {
	inner := &mockFetcher{
		results: []mockResult{{err: domain.ErrFetchFailed}},
	}
	rf := NewRetryFetcher(inner, 3, 1*time.Second, 10*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := rf.Fetch(ctx, "https://example.com", domain.FetchOptions{})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if inner.callCount != 1 {
		t.Errorf("callCount = %d, want 1", inner.callCount)
	}
}
