//go:build !jsrender

// Package jsrender provides a no-op stub when the "jsrender" build tag is not set.
package jsrender

import (
	"context"
	"errors"

	"github.com/paimonchan/paimon-mcp-fetch/internal/domain"
)

// Renderer is a no-op JS renderer.
type Renderer struct{}

// NewRenderer returns a stub that always errors.
func NewRenderer(_ interface{}) *Renderer {
	return &Renderer{}
}

// Fetch always returns an error — JS rendering requires the "jsrender" build tag.
func (r *Renderer) Fetch(ctx context.Context, url string, opts domain.FetchOptions) (*domain.FetchResponse, error) {
	return nil, errors.New("JS rendering not compiled: build with -tags jsrender")
}
