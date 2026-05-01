# paimon-mcp-fetch

A fast, secure MCP server for fetching web content. Built with Go as a single binary with zero runtime dependencies.

[![Release](https://img.shields.io/github/v/release/paimonchan/paimon-mcp-fetch)](https://github.com/paimonchan/paimon-mcp-fetch/releases)
[![Go Version](https://img.shields.io/badge/go-1.26+-blue.svg)](https://go.dev/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## Quick Start

```bash
# Install via go install (recommended)
go install github.com/paimonchan/paimon-mcp-fetch/cmd/paimon-mcp-fetch@latest

# Or download from GitHub Releases
# https://github.com/paimonchan/paimon-mcp-fetch/releases
```

Add to your MCP client config (e.g. `~/.config/opencode/opencode.json`):

```json
{
  "mcp": {
    "fetch": {
      "type": "local",
      "command": ["paimon-mcp-fetch"],
      "enabled": true
    }
  }
}
```

Done. Start fetching URLs in your AI assistant.

---

## Features

| Feature | Detail |
|---------|--------|
| **Web Content Extraction** | Fetches pages and converts HTML to clean markdown |
| **Image Processing** | Optional (`-tags image`) — resize, merge, optimize images |
| **JS Rendering** | Optional (`-tags jsrender`) — headless Chrome for SPAs and dynamic sites |
| **SSRF Protection** | 7-layer defense against server-side request forgery |
| **Smart Defaults** | robots.txt disabled by default, browser-like User-Agent, 10MB HTML limit |
| **Caching** | In-memory LRU cache with configurable TTL |
| **Rate Limiting** | Per-domain token bucket (5 req/sec, burst 10) |
| **Retry** | Exponential backoff for transient errors only |

---

## Installation

### Option 1: `go install` (Recommended)

Requires [Go 1.26+](https://go.dev/dl/).

```bash
go install github.com/paimonchan/paimon-mcp-fetch/cmd/paimon-mcp-fetch@latest
```

### Option 2: Download Release

Grab the pre-built binary for your OS from [GitHub Releases](https://github.com/paimonchan/paimon-mcp-fetch/releases).

### Option 3: Build from Source

```bash
git clone https://github.com/paimonchan/paimon-mcp-fetch
cd paimon-mcp-fetch
go build -ldflags="-s -w" -o paimon-mcp-fetch ./cmd/paimon-mcp-fetch/
```

### Option 4: Docker

```bash
docker run -i --rm ghcr.io/paimonchan/paimon-mcp-fetch:latest
```

### Option 5: Package Manager

**Homebrew (macOS/Linux):**

```bash
brew tap paimonchan/tap
brew install paimon-mcp-fetch
```

**Scoop (Windows):**

```powershell
scoop bucket add paimonchan https://github.com/paimonchan/scoop-bucket
scoop install paimon-mcp-fetch
```

**Winget (Windows):**

```powershell
winget install paimonchan.paimon-mcp-fetch
```

> Winget manifest is pending review: [microsoft/winget-pkgs#367457](https://github.com/microsoft/winget-pkgs/pull/367457)

### Windows Note

Go binaries are occasionally flagged by some antivirus software as a false positive. If this happens, use `go install` (your antivirus sees `go.exe`, not our binary) or the Docker option. See [Security & Trust](#security--trust) for more details.

---

## MCP Client Configuration

### OpenCode

`~/.config/opencode/opencode.json`:

```json
{
  "mcp": {
    "fetch": {
      "type": "local",
      "command": ["paimon-mcp-fetch"],
      "enabled": true
    }
  }
}
```

### Claude Desktop

`claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "fetch": {
      "command": "paimon-mcp-fetch"
    }
  }
}
```

### VS Code / Cursor / Cline / Windsurf

```json
{
  "mcpServers": {
    "fetch": {
      "command": "paimon-mcp-fetch"
    }
  }
}
```

---

## Environment Variables

All config is via environment variables with sensible defaults.

| Variable | Default | Description |
|----------|---------|-------------|
| `PAIMON_MCP_FETCH_TIMEOUT_MS` | 12000 | Request timeout in milliseconds |
| `PAIMON_MCP_FETCH_MAX_HTML_BYTES` | 10485760 | Max HTML response size (10MB) |
| `PAIMON_MCP_FETCH_MAX_IMAGE_BYTES` | 10485760 | Max image size (10MB) |
| `PAIMON_MCP_FETCH_CACHE_ENABLED` | true | Enable response cache |
| `PAIMON_MCP_FETCH_CACHE_TTL_SECS` | 300 | Cache TTL in seconds |
| `PAIMON_MCP_FETCH_RATE_LIMIT_PER_SECOND` | 5.0 | Requests per second per domain |
| `PAIMON_MCP_FETCH_RATE_LIMIT_BURST` | 10 | Max burst size per domain |
| `PAIMON_MCP_FETCH_RETRY_MAX_ATTEMPTS` | 3 | Max retry attempts |
| `PAIMON_MCP_FETCH_JS_RENDER_ENABLED` | false | Enable headless Chrome JS rendering |
| `PAIMON_MCP_FETCH_DISABLE_SSRF` | false | Disable SSRF guard |

Full list: see `internal/config/config.go`.

---

## Optional Build Features

### Image Processing (`-tags image`)

Compile with image support for downloading, resizing, merging, and saving images:

```bash
go build -tags image -o paimon-mcp-fetch ./cmd/paimon-mcp-fetch/
```

### JS Rendering (`-tags jsrender`)

For JavaScript-heavy sites (Yahoo Finance, TradingView, SPAs):

**Requirements:** Chrome or Chromium must be installed.

```bash
go build -tags jsrender -o paimon-mcp-fetch ./cmd/paimon-mcp-fetch/
PAIMON_MCP_FETCH_JS_RENDER_ENABLED=true ./paimon-mcp-fetch
```

Slower (~3-5s per page) but can extract data that static fetch cannot.

---

## Security & Trust

### Why a Go Binary?

- **Zero runtime dependencies** — no Node.js, no Python, no Docker
- **Startup in ~5ms** (vs 500ms–2s for Node.js)
- **Memory usage ~8–15MB** (vs 50–100MB for Node.js)
- **Fully open source** — MIT license, public repo
- **Reproducible builds** — GitHub Actions CI/CD with public logs
- **SHA256 checksums** — every release includes verified hashes

### Security Features

- **SSRF Protection**: 7-layer defense (URL parse, scheme validation, hostname blocklist, DNS resolution, private IP filtering, redirect re-validation, stream limits)
- **Size Limits**: Stream-based reading with configurable byte limits
- **Timeouts**: All network calls have context deadlines
- **No Secrets in Logs**: API keys and tokens are never logged

### Windows Antivirus Note

Some antivirus software may flag unsigned Go binaries as a false positive. This is a known industry-wide issue with compiled binaries, not specific to this project. If you encounter this:

- Use `go install` — your antivirus sees the signed Go compiler, not a raw binary
- Use the Docker option — no local binary at all
- Build from source — verify the code yourself

---

## MCP Tool Schema

### `fetch_webpage`

Fetch and extract content from any webpage URL, returning clean structured markdown. Uses the Readability algorithm to identify the main article body.

**When to use `fetch_webpage` over other fetch tools:**

| Scenario | Use `fetch_webpage` | Use basic text fetch |
|----------|---------------------|----------------------|
| Need structured output (headings, lists, tables, code) | ✅ Markdown preserves structure | ❌ Plain text loses formatting |
| Reading articles, docs, blog posts | ✅ Readability extracts main content | ⚠️ May include navigation/ads |
| Need images from the page | ✅ Built-in image extraction | ❌ No image support |
| Long articles with pagination | ✅ `startIndex` for pagination | ⚠️ Limited pagination |
| JS-heavy sites (SPAs, dynamic content) | ✅ Optional JS rendering | ❌ Static HTML only |
| Quick plain text extraction | ⚠️ Overkill but works | ✅ Simpler, faster |

```json
{
  "url": "https://example.com",
  "text": {
    "maxLength": 20000,
    "startIndex": 0,
    "raw": false
  },
  "images": {
    "output": "base64",
    "layout": "merged",
    "maxCount": 3,
    "size": {
      "maxWidth": 1000,
      "maxHeight": 1600,
      "quality": 80
    }
  },
  "security": {
    "ignoreRobotsTxt": true
  }
}
```

---

## Architecture

```
MCP Server (mcp-go) → UseCase → Domain (entities, ports)
                              ↑
              Adapters implement port interfaces
```

**Layers:**
- **Domain** — Entities, errors, port interfaces, policies (zero external deps)
- **UseCase** — Orchestration logic with constructor injection
- **Adapter** — HTTP client, content extractor, robots checker, cache, rate limiter, image processor, JS renderer

---

## License

MIT

---

> Built with Go. Zero runtime dependencies. Single binary.
