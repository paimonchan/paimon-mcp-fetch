// Package extractor converts HTML to cleaned markdown content.
package extractor

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	"codeberg.org/readeck/go-readability/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/PuerkitoBio/goquery"
	"github.com/paimonchan/paimon-mcp-fetch/internal/domain"
)

// readabilityExtractor implements domain.ContentExtractor using go-readability
// and html-to-markdown.
type readabilityExtractor struct{}

// NewReadabilityExtractor creates a new ContentExtractor backed by readability
// and html-to-markdown.
func NewReadabilityExtractor() domain.ContentExtractor {
	return &readabilityExtractor{}
}

// Extract parses the HTML, extracts the article via readability, converts to
// markdown, and collects image references.
func (e *readabilityExtractor) Extract(ctx context.Context, html, pageURL string) (*domain.ExtractedContent, error) {
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid URL %q", domain.ErrInvalidURL, pageURL)
	}

	// 1. Run readability
	article, err := readability.FromReader(strings.NewReader(html), parsedURL)
	var cleanedHTML string
	if err != nil || article.Node == nil {
		// Fallback: use raw <body> content
		cleanedHTML = fallbackBody(html)
		if strings.TrimSpace(cleanedHTML) == "" {
			return nil, fmt.Errorf("%w: readability failed and no body found", domain.ErrNoContent)
		}
	} else {
		// 2. Render cleaned HTML from readability node
		var buf bytes.Buffer
		if err := article.RenderHTML(&buf); err != nil {
			return nil, fmt.Errorf("%w: render HTML: %v", domain.ErrExtractionFailed, err)
		}
		cleanedHTML = buf.String()
	}

	if strings.TrimSpace(cleanedHTML) == "" {
		return nil, fmt.Errorf("%w: no content could be extracted", domain.ErrNoContent)
	}

	// 3. Convert to markdown
	markdown, err := htmlToMarkdown(cleanedHTML)
	if err != nil {
		return nil, fmt.Errorf("%w: markdown conversion: %v", domain.ErrExtractionFailed, err)
	}

	// 4. Extract image references
	images := extractImageRefs(cleanedHTML, pageURL)

	return &domain.ExtractedContent{
		Title:    article.Title(),
		Content:  cleanedHTML,
		Markdown: markdown,
		Images:   images,
	}, nil
}

// extractFromBody handles fallback extraction when readability fails.
func (e *readabilityExtractor) extractFromBody(body, rawHTML string) (*domain.ExtractedContent, error) {
	markdown, err := htmlToMarkdown(body)
	if err != nil {
		return nil, fmt.Errorf("%w: markdown conversion: %v", domain.ErrExtractionFailed, err)
	}
	images := extractImageRefs(body, "")
	return &domain.ExtractedContent{
		Title:    fallbackTitle(rawHTML),
		Content:  body,
		Markdown: markdown,
		Images:   images,
	}, nil
}

// htmlToMarkdown converts HTML string to markdown.
func htmlToMarkdown(html string) (string, error) {
	md, err := htmltomarkdown.ConvertString(html)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(md), nil
}

// extractImageRefs parses HTML and returns all <img> references.
func extractImageRefs(html, baseURL string) []domain.ImageRef {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}

	var refs []domain.ImageRef
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists || src == "" {
			return
		}
		alt, _ := s.Attr("alt")
		refs = append(refs, domain.ImageRef{
			Src:      src,
			Alt:      alt,
			Filename: filenameFromURL(src),
		})
	})
	return refs
}

// fallbackBody extracts the raw <body> tag content from HTML.
// Returns empty string if body exists but is empty.
// Returns original html only if parsing fails or no body tag found.
func fallbackBody(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}
	bodyNodes := doc.Find("body")
	if bodyNodes.Length() == 0 {
		return html
	}
	body, _ := bodyNodes.Html()
	return body
}

// fallbackTitle extracts the <title> tag text from HTML.
func fallbackTitle(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(doc.Find("title").Text())
}

// filenameFromURL extracts a filename from a URL path.
func filenameFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return "image.jpg"
	}
	name := parts[len(parts)-1]
	name = strings.Split(name, "?")[0]
	if name == "" {
		return "image.jpg"
	}
	return name
}
