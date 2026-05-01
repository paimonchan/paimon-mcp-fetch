//go:build image

package image

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/paimonchan/paimon-mcp-fetch/internal/domain"
)

func createTestPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 0, 255})
		}
	}
	var buf []byte
	// We can't easily use bytes.Buffer here without import, but for the test server
	// we can encode on the fly in the handler.
	return buf
}

func TestProcessor_FetchAndProcess(t *testing.T) {
	// Create a test server that serves a simple PNG
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		img := image.NewRGBA(image.Rect(0, 0, 100, 100))
		for y := 0; y < 100; y++ {
			for x := 0; x < 100; x++ {
				img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 0, 255})
			}
		}
		w.Header().Set("Content-Type", "image/png")
		png.Encode(w, img)
	}))
	defer server.Close()

	proc := NewProcessor(nil)

	images := []domain.ImageRef{
		{Src: server.URL + "/img1.png", Alt: "Test 1", Filename: "test1.png"},
		{Src: server.URL + "/img2.png", Alt: "Test 2", Filename: "test2.png"},
	}

	// Test basic fetch + resize
	t.Run("fetch_and_resize", func(t *testing.T) {
		opts := domain.ImageProcessOptions{
			MaxCount:   2,
			MaxWidth:   50,
			MaxHeight:  50,
			Quality:    80,
			Layout:     "individual",
			OutputBase64: true,
		}

		results, err := proc.FetchAndProcess(context.Background(), images, "", opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
		for _, r := range results {
			if r.Width > 50 || r.Height > 50 {
				t.Errorf("image exceeds max size: %dx%d", r.Width, r.Height)
			}
			if r.MimeType != "image/jpeg" {
				t.Errorf("expected jpeg, got %s", r.MimeType)
			}
			if len(r.Data) == 0 {
				t.Error("expected non-empty image data")
			}
		}
	})

	// Test merged layout
	t.Run("merged_layout", func(t *testing.T) {
		opts := domain.ImageProcessOptions{
			MaxCount:   2,
			MaxWidth:   100,
			MaxHeight:  100,
			Quality:    80,
			Layout:     "merged",
			OutputBase64: true,
		}

		results, err := proc.FetchAndProcess(context.Background(), images, "", opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Merged layout prepends a merged image + individual images
		if len(results) < 3 {
			t.Errorf("expected at least 3 results (merged + 2 individual), got %d", len(results))
		}
		if results[0].Src != "merged" {
			t.Errorf("expected first result to be merged, got %s", results[0].Src)
		}
	})

	// Test save to file
	t.Run("save_to_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		opts := domain.ImageProcessOptions{
			MaxCount:   1,
			MaxWidth:   50,
			MaxHeight:  50,
			Quality:    80,
			Layout:     "individual",
			SaveToFile: true,
			SaveDir:    tmpDir,
		}

		results, err := proc.FetchAndProcess(context.Background(), images[:1], "", opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected at least 1 result")
		}
		if results[0].FilePath == "" {
			t.Error("expected FilePath to be set")
		}
		if _, err := os.Stat(results[0].FilePath); os.IsNotExist(err) {
			t.Errorf("file not found: %s", results[0].FilePath)
		}
	})

	// Test pagination (startIndex + maxCount)
	t.Run("pagination", func(t *testing.T) {
		opts := domain.ImageProcessOptions{
			MaxCount:   1,
			StartIndex: 1,
			MaxWidth:   100,
			MaxHeight:  100,
			Quality:    80,
			Layout:     "individual",
		}

		results, err := proc.FetchAndProcess(context.Background(), images, "", opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("expected 1 result (startIndex=1, maxCount=1), got %d", len(results))
		}
	})
}

func TestProcessor_EmptyImages(t *testing.T) {
	proc := NewProcessor(nil)
	results, err := proc.FetchAndProcess(context.Background(), nil, "", domain.ImageProcessOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		ref, base, want string
		wantErr         bool
	}{
		{"https://example.com/img.jpg", "", "https://example.com/img.jpg", false},
		{"/img.jpg", "https://example.com", "https://example.com/img.jpg", false},
		{"img.jpg", "https://example.com/", "https://example.com/img.jpg", false},
		{"img.jpg", "", "", true},
	}

	for _, tc := range tests {
		got, err := resolveURL(tc.ref, tc.base)
		if tc.wantErr {
			if err == nil {
				t.Errorf("resolveURL(%q, %q) expected error", tc.ref, tc.base)
			}
			continue
		}
		if err != nil {
			t.Errorf("resolveURL(%q, %q) unexpected error: %v", tc.ref, tc.base, err)
			continue
		}
		if got != tc.want {
			t.Errorf("resolveURL(%q, %q) = %q, want %q", tc.ref, tc.base, got, tc.want)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		preferred, fallback, ext, wantPrefix string
	}{
		{"test.png", "", "jpg", "test_"},
		{"", "https://example.com/image.png", "jpg", "image_"},
		{"../../etc/passwd", "", "jpg", "etc_passwd_"},
	}

	for _, tc := range tests {
		got := sanitizeFilename(tc.preferred, tc.fallback, tc.ext)
		if !filepath.IsAbs(got) {
			// Should not contain path traversal
			if filepath.Base(got) != got {
				t.Errorf("sanitizeFilename produced path with dir: %q", got)
			}
		}
		if len(got) < len(tc.wantPrefix) {
			t.Errorf("sanitizeFilename(%q, %q, %q) = %q, expected prefix %q",
				tc.preferred, tc.fallback, tc.ext, got, tc.wantPrefix)
		}
	}
}
