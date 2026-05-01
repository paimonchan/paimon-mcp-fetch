// Package usecase contains application business rules for paimon-mcp-fetch.
package usecase

import (
	"testing"

	"github.com/user/paimon-mcp-fetch/internal/domain"
)

func TestPaginate(t *testing.T) {
	uc := &FetchUseCase{}

	tests := []struct {
		content      string
		startIndex   int
		maxLength    int
		wantResult   string
		wantRemaining int
	}{
		{"hello world", 0, 5, "hello", 6},
		{"hello world", 6, 5, "world", 0},
		{"hello world", 0, 100, "hello world", 0},
		{"hello world", 20, 5, "", 0},
		{"", 0, 10, "", 0},
	}

	for _, tc := range tests {
		gotResult, gotRemaining := uc.paginate(tc.content, tc.startIndex, tc.maxLength)
		if gotResult != tc.wantResult {
			t.Errorf("paginate(%q, %d, %d) result = %q, want %q",
				tc.content, tc.startIndex, tc.maxLength, gotResult, tc.wantResult)
		}
		if gotRemaining != tc.wantRemaining {
			t.Errorf("paginate(%q, %d, %d) remaining = %d, want %d",
				tc.content, tc.startIndex, tc.maxLength, gotRemaining, tc.wantRemaining)
		}
	}
}

func TestRemainingImages(t *testing.T) {
	uc := &FetchUseCase{}

	images := []domain.ImageRef{
		{Src: "1.jpg"}, {Src: "2.jpg"}, {Src: "3.jpg"}, {Src: "4.jpg"}, {Src: "5.jpg"},
	}

	tests := []struct {
		startIndex int
		maxCount   int
		want       int
	}{
		{0, 3, 2},  // show 0-2, remaining 3-4 = 2
		{3, 2, 0},  // show 3-4, remaining 0
		{0, 10, 0}, // show all, remaining 0
		{5, 3, 0},  // start at end, remaining 0
	}

	for _, tc := range tests {
		got := uc.remainingImages(images, tc.startIndex, tc.maxCount)
		if got != tc.want {
			t.Errorf("remainingImages(..., %d, %d) = %d, want %d",
				tc.startIndex, tc.maxCount, got, tc.want)
		}
	}
}

func TestValidateRequest(t *testing.T) {
	uc := NewFetchUseCase(nil, nil, nil, nil, nil, nil, domain.DefaultSizePolicy())

	// Valid request
	err := uc.validateRequest(&domain.FetchRequest{
		URL: "https://example.com",
		Text: domain.TextOptions{
			MaxLength: 1000,
		},
		Images: domain.ImageOptions{
			MaxCount: 3,
			Quality:  80,
			MaxWidth: 1000,
			MaxHeight: 1600,
		},
	})
	if err != nil {
		t.Errorf("unexpected error for valid request: %v", err)
	}

	// Missing URL
	err = uc.validateRequest(&domain.FetchRequest{})
	if err == nil {
		t.Error("expected error for missing URL")
	}

	// Invalid maxLength
	err = uc.validateRequest(&domain.FetchRequest{
		URL: "https://example.com",
		Text: domain.TextOptions{MaxLength: -1},
	})
	if err == nil {
		t.Error("expected error for negative maxLength")
	}

	// Invalid imageMaxCount
	err = uc.validateRequest(&domain.FetchRequest{
		URL: "https://example.com",
		Images: domain.ImageOptions{MaxCount: 20},
	})
	if err == nil {
		t.Error("expected error for imageMaxCount > 10")
	}
}

func TestCacheKey(t *testing.T) {
	uc := &FetchUseCase{}

	key1 := uc.cacheKey("https://example.com")
	key2 := uc.cacheKey("https://example.com")
	key3 := uc.cacheKey("https://other.com")

	if key1 != key2 {
		t.Error("same URL should produce same cache key")
	}
	if key1 == key3 {
		t.Error("different URLs should produce different cache keys")
	}
	if len(key1) != 64 {
		t.Errorf("expected SHA256 hex string (64 chars), got %d", len(key1))
	}
}
