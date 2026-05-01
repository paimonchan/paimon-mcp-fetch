//go:build !image

// Package image provides a no-op image processor when the "image" build tag is not set.
package image

import (
	"context"

	"github.com/user/paimon-mcp-fetch/internal/domain"
)

// Processor is a no-op image processor.
type Processor struct{}

// NewProcessor creates a no-op processor (image support not compiled).
func NewProcessor(_ interface{}) *Processor {
	return &Processor{}
}

// FetchAndProcess returns nil — image support requires the "image" build tag.
func (p *Processor) FetchAndProcess(
	ctx context.Context,
	images []domain.ImageRef,
	baseOrigin string,
	opts domain.ImageProcessOptions,
) ([]domain.ImageResult, error) {
	return nil, nil
}
