//go:build image

// Package image provides image processing for paimon-mcp-fetch.
// This package is only compiled when the "image" build tag is provided.
package image

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/paimonchan/paimon-mcp-fetch/internal/domain"
)

// Processor implements domain.ImageProcessor.
type Processor struct {
	client *http.Client
}

// NewProcessor creates a new image processor.
func NewProcessor(client *http.Client) *Processor {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Processor{client: client}
}

// FetchAndProcess downloads, resizes, and optionally merges images.
func (p *Processor) FetchAndProcess(
	ctx context.Context,
	images []domain.ImageRef,
	baseOrigin string,
	opts domain.ImageProcessOptions,
) ([]domain.ImageResult, error) {
	if len(images) == 0 {
		return nil, nil
	}

	// Apply startIndex and maxCount pagination
	start := opts.StartIndex
	if start >= len(images) {
		return nil, nil
	}
	end := start + opts.MaxCount
	if end > len(images) {
		end = len(images)
	}
	selected := images[start:end]

	// Prepare save directory
	saveDir := opts.SaveDir
	if saveDir == "" {
		saveDir = defaultSaveDir()
	}
	if opts.SaveToFile && saveDir != "" {
		if err := os.MkdirAll(saveDir, 0755); err != nil {
			return nil, fmt.Errorf("create save dir: %w", err)
		}
	}

	// Fetch and process each image
	var results []domain.ImageResult
	var decodedImages []image.Image
	var decodedMeta []domain.ImageRef

	for _, imgRef := range selected {
		imgURL, err := resolveURL(imgRef.Src, baseOrigin)
		if err != nil {
			continue // Skip invalid URLs
		}

		result, decoded, err := p.processSingle(ctx, imgURL, imgRef, opts, saveDir)
		if err != nil {
			continue // Skip failed images
		}

		results = append(results, *result)
		if decoded != nil {
			decodedImages = append(decodedImages, decoded)
			decodedMeta = append(decodedMeta, imgRef)
		}
	}

	// Handle merged layout
	if opts.Layout == "merged" || opts.Layout == "both" {
		if len(decodedImages) > 1 {
			merged, err := p.mergeVertically(decodedImages)
			if err == nil {
				mergedBytes, err := p.encodeJPEG(merged, opts.Quality)
				if err == nil {
					mergedResult := domain.ImageResult{
						Src:      "merged",
						Alt:      fmt.Sprintf("Merged %d images", len(decodedImages)),
						Data:     mergedBytes,
						MimeType: "image/jpeg",
						Width:    merged.Bounds().Dx(),
						Height:   merged.Bounds().Dy(),
					}
					if opts.SaveToFile && saveDir != "" {
						mergedResult.FilePath = p.saveToFile(mergedBytes, saveDir, "merged.jpg")
					}
					results = append([]domain.ImageResult{mergedResult}, results...)
				}
			}
		}
	}

	return results, nil
}

// processSingle fetches and processes a single image.
func (p *Processor) processSingle(
	ctx context.Context,
	imgURL string,
	imgRef domain.ImageRef,
	opts domain.ImageProcessOptions,
	saveDir string,
) (*domain.ImageResult, image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		return nil, nil, err
	}

	// Decode image
	decoded, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		// If decode fails, return raw bytes anyway
		result := &domain.ImageResult{
			Src:      imgURL,
			Alt:      imgRef.Alt,
			Data:     data,
			MimeType: resp.Header.Get("Content-Type"),
		}
		if opts.SaveToFile && saveDir != "" {
			ext := extFromMime(result.MimeType, format)
			result.FilePath = p.saveToFile(data, saveDir, sanitizeFilename(imgRef.Filename, imgURL, ext))
		}
		return result, nil, nil
	}

	// Resize if needed
	bounds := decoded.Bounds()
	origW, origH := bounds.Dx(), bounds.Dy()

	resized := decoded
	if opts.MaxWidth > 0 && opts.MaxHeight > 0 && (origW > opts.MaxWidth || origH > opts.MaxHeight) {
		resized = imaging.Fit(decoded, opts.MaxWidth, opts.MaxHeight, imaging.Lanczos)
	}

	// Encode to JPEG
	jpegBytes, err := p.encodeJPEG(resized, opts.Quality)
	if err != nil {
		return nil, nil, err
	}

	result := &domain.ImageResult{
		Src:      imgURL,
		Alt:      imgRef.Alt,
		Data:     jpegBytes,
		MimeType: "image/jpeg",
		Width:    resized.Bounds().Dx(),
		Height:   resized.Bounds().Dy(),
	}

	if opts.SaveToFile && saveDir != "" {
		result.FilePath = p.saveToFile(jpegBytes, saveDir, sanitizeFilename(imgRef.Filename, imgURL, "jpg"))
	}

	return result, resized, nil
}

// mergeVertically merges images into a single vertical strip.
func (p *Processor) mergeVertically(images []image.Image) (image.Image, error) {
	if len(images) == 0 {
		return nil, fmt.Errorf("no images to merge")
	}

	// Find max width and total height
	maxWidth := 0
	totalHeight := 0
	for _, img := range images {
		bounds := img.Bounds()
		if bounds.Dx() > maxWidth {
			maxWidth = bounds.Dx()
		}
		totalHeight += bounds.Dy()
	}

	// Create canvas
	canvas := imaging.New(maxWidth, totalHeight, image.White)

	yOffset := 0
	for _, img := range images {
		bounds := img.Bounds()
		// Center horizontally if narrower than canvas
		xOffset := (maxWidth - bounds.Dx()) / 2
		canvas = imaging.Paste(canvas, img, image.Pt(xOffset, yOffset))
		yOffset += bounds.Dy()
	}

	return canvas, nil
}

// encodeJPEG encodes an image to JPEG bytes.
func (p *Processor) encodeJPEG(img image.Image, quality int) ([]byte, error) {
	if quality < 1 {
		quality = 80
	}
	if quality > 100 {
		quality = 100
	}

	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// saveToFile saves image data to the save directory.
func (p *Processor) saveToFile(data []byte, dir, filename string) string {
	filepath := filepath.Join(dir, filename)
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return ""
	}
	return filepath
}

// defaultSaveDir returns the default save directory.
func defaultSaveDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Downloads", "paimon-mcp-fetch", time.Now().Format("2006-01-02"))
}

// resolveURL resolves a relative URL against a base origin.
func resolveURL(ref, base string) (string, error) {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref, nil
	}
	if base == "" {
		return "", fmt.Errorf("cannot resolve relative URL without base origin")
	}
	return base + ref, nil
}

// sanitizeFilename creates a safe filename from the given inputs.
func sanitizeFilename(preferred, fallbackURL, ext string) string {
	name := preferred
	if name == "" {
		name = path.Base(fallbackURL)
		if name == "" || name == "." || name == "/" {
			name = "image"
		}
	}
	// Remove extension if present
	name = strings.TrimSuffix(name, path.Ext(name))
	// Sanitize
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, name)
	if name == "" {
		name = "image"
	}
	return fmt.Sprintf("%s_%d.%s", name, time.Now().UnixNano(), ext)
}

// extFromMime returns a file extension from a MIME type or format string.
func extFromMime(mimeType, format string) string {
	switch {
	case strings.Contains(mimeType, "jpeg") || strings.Contains(mimeType, "jpg"):
		return "jpg"
	case strings.Contains(mimeType, "png"):
		return "png"
	case strings.Contains(mimeType, "gif"):
		return "gif"
	case strings.Contains(mimeType, "webp"):
		return "webp"
	}
	switch format {
	case "jpeg":
		return "jpg"
	case "png", "gif", "webp":
		return format
	}
	return "jpg"
}
