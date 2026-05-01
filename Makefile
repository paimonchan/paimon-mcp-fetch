# Makefile — paimon-mcp-fetch
# Windows-compatible (uses PowerShell for cross-compilation loops)

PLATFORMS = linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64

BINARY = paimon-mcp-fetch
CMD = ./cmd/$(BINARY)/
DIST = dist
VERSION ?= dev

LDFLAGS = -s -w -X main.version=$(VERSION)

.PHONY: all build build-image build-all test test-image clean tidy fmt lint release

all: build

## Build default (no image processing)
build:
	go build -ldflags="$(LDFLAGS)" -o $(DIST)/$(BINARY) $(CMD)

## Build with image processing support
build-image:
	go build -tags image -ldflags="$(LDFLAGS)" -o $(DIST)/$(BINARY) $(CMD)

## Cross-compile for all platforms (Windows host uses PowerShell)
build-all:
	@powershell -Command " \
		$$platforms = '$(PLATFORMS)'.Split(' '); \
		foreach ($$p in $$platforms) { \
			$$parts = $$p.Split('/'); \
			$$env:GOOS = $$parts[0]; \
			$$env:GOARCH = $$parts[1]; \
			$$suffix = ($$env:GOOS -eq 'windows' ? '.exe' : ''); \
			go build -ldflags='$(LDFLAGS)' -o ('$(DIST)/$(BINARY)-' + $$p + $$suffix) $(CMD); \
			Write-Host ('Built: ' + $$p); \
		} \
	"

## Run tests
test:
	go test ./... -v -cover

## Run tests with image build tag
test-image:
	go test -tags image ./... -v -cover

## Tidy modules
tidy:
	go mod tidy

## Format code
fmt:
	go fmt ./...

## Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

## Clean build artifacts
clean:
	go clean
	if exist $(DIST) rd /s /q $(DIST)

## Build release binaries locally
release: clean build-all
	@powershell -Command " \
		Get-ChildItem $(DIST) -File | Where-Object { $$_.Extension -ne '.sha256' } | ForEach-Object { \
			$$hash = (Get-FileHash $$_.FullName -Algorithm SHA256).Hash.ToLower(); \
			'$$hash  $$($$_.Name)' | Out-File -FilePath ('$$($$_.FullName).sha256') -Encoding utf8; \
			Write-Host ('Checksum: ' + $$_.Name); \
		} \
	"
