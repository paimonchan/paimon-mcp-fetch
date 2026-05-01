# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install git (needed for go mod download)
RUN apk add --no-cache git

# Copy module files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary (statically linked for Alpine)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=$(git describe --tags --always)" -o paimon-mcp-fetch ./cmd/paimon-mcp-fetch/

# Final stage — minimal image
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.source="https://github.com/paimonchan/paimon-mcp-fetch"
LABEL org.opencontainers.image.description="Web content fetching MCP server"
LABEL org.opencontainers.image.licenses="MIT"

# Copy binary from builder
COPY --from=builder /app/paimon-mcp-fetch /usr/local/bin/paimon-mcp-fetch

# Use non-root user (distroless provides nonroot:65532)
USER nonroot:nonroot

# Expose stdio (MCP uses stdio transport, no TCP port needed)
# This is informational only

ENTRYPOINT ["/usr/local/bin/paimon-mcp-fetch"]
