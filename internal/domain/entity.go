// Package domain contains enterprise business rules for paimon-mcp-fetch.
// ZERO imports from other layers.
package domain

import "encoding/base64"

// ContentType indicates the format of extracted content.
type ContentType string

const (
	ContentTypeHTML     ContentType = "html"
	ContentTypeMarkdown ContentType = "markdown"
	ContentTypeRaw      ContentType = "raw"
)

// TextOptions controls text extraction and pagination.
type TextOptions struct {
	MaxLength  int
	StartIndex int
	Raw        bool
}

// ImageOptions controls image fetching and processing.
type ImageOptions struct {
	Enable           bool
	MaxCount         int
	StartIndex       int
	MaxWidth         int
	MaxHeight        int
	Quality          int
	AllowCrossOrigin bool
	SaveDir          string
	OutputBase64     bool
	SaveToFile       bool
	Layout           string // "merged", "individual", "both"
}

// SecurityOptions controls security-related behavior.
type SecurityOptions struct {
	IgnoreRobotsTxt bool
}

// FetchRequest is the input for a fetch operation.
type FetchRequest struct {
	URL      string
	Text     TextOptions
	Images   ImageOptions
	Security SecurityOptions
	Render   string // "static" | "dynamic" (future)
}

// FetchResult is the output of a fetch operation.
type FetchResult struct {
	Title            string
	Content          string
	ContentType      ContentType
	ContentBlocks    []ContentBlock
	Images           []ImageResult
	RemainingContent int
	RemainingImages  int
}

// ContentBlock represents a block of content (text or image).
type ContentBlock struct {
	Type     string // "text", "image"
	Text     string // for text blocks
	MimeType string // for image blocks
	Data     string // base64 data for image blocks
	FilePath string // local file path if saved
}

// ImageResult holds a processed image.
type ImageResult struct {
	Src      string
	Alt      string
	Data     []byte // raw image bytes
	MimeType string
	FilePath string // if saved to disk
	Width    int
	Height   int
}

// Base64 encodes the raw image bytes to a base64 string.
func (ir ImageResult) Base64() string {
	if len(ir.Data) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(ir.Data)
}
