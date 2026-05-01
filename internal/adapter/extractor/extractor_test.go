// Package extractor provides HTML content extraction adapters.
package extractor

import (
	"context"
	"strings"
	"testing"

	"github.com/user/paimon-mcp-fetch/internal/domain"
)

func TestExtract_SimpleArticle(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head><title>Test Article</title></head>
<body>
<article>
<h1>Hello World</h1>
<p>This is a test paragraph.</p>
<img src="https://example.com/img.png" alt="test image">
</article>
</body>
</html>`

	ext := NewReadabilityExtractor()
	result, err := ext.Extract(context.Background(), html, "https://example.com/page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Title == "" {
		t.Error("expected title to be non-empty")
	}
	if !strings.Contains(result.Markdown, "Hello World") {
		t.Errorf("expected markdown to contain 'Hello World', got: %s", result.Markdown)
	}
	if len(result.Images) == 0 {
		t.Error("expected at least one image reference")
	}
	if result.Images[0].Src != "https://example.com/img.png" {
		t.Errorf("expected image src to be 'https://example.com/img.png', got: %s", result.Images[0].Src)
	}
}

func TestExtract_FallbackBody(t *testing.T) {
	html := `<html><head><title>Fallback Title</title></head>
<body><p>No article tag, just body content.</p></body></html>`

	ext := NewReadabilityExtractor()
	result, err := ext.Extract(context.Background(), html, "https://example.com/page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Title != "Fallback Title" {
		t.Errorf("expected title 'Fallback Title', got: %s", result.Title)
	}
	if !strings.Contains(result.Content, "No article tag") {
		t.Errorf("expected content to contain body text, got: %s", result.Content)
	}
}

func TestExtract_NoContent(t *testing.T) {
	html := `<html><head></head><body></body></html>`

	ext := NewReadabilityExtractor()
	result, err := ext.Extract(context.Background(), html, "https://example.com/page")
	if err == nil {
		t.Logf("result: %+v", result)
		t.Logf("markdown: %q", result.Markdown)
		t.Logf("content: %q", result.Content)
		t.Fatal("expected error for empty content")
	}
	if !strings.Contains(err.Error(), domain.ErrNoContent.Error()) {
		t.Errorf("expected ErrNoContent, got: %v", err)
	}
}

func TestFilenameFromURL(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/image.png", "image.png"},
		{"https://example.com/path/to/img.jpg?size=large", "img.jpg"},
		{"https://example.com/", "image.jpg"},
		{"", "image.jpg"},
	}

	for _, tc := range tests {
		got := filenameFromURL(tc.url)
		if got != tc.expected {
			t.Errorf("filenameFromURL(%q) = %q, want %q", tc.url, got, tc.expected)
		}
	}
}
