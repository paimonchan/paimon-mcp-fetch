# paimon-mcp-fetch

**Give your AI assistant the ability to read any webpage.**

A lightweight MCP server that fetches URLs and returns clean, readable markdown. Built with Go — starts in 5ms, uses ~10MB RAM, zero runtime dependencies.

[![Release](https://img.shields.io/github/v/release/paimonchan/paimon-mcp-fetch)](https://github.com/paimonchan/paimon-mcp-fetch/releases)
[![Go Version](https://img.shields.io/badge/go-1.26+-blue.svg)](https://go.dev/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

---

## What It Does

You give it a URL → it returns clean markdown.

**Good for:**
- Reading articles, blog posts, documentation
- Extracting data from news sites, forums, schedules
- Summarizing web content for your AI assistant
- Getting structured output (headings, lists, tables, code blocks preserved)

**Not good for:**
- Scraping login-protected pages
- Bypassing paywalls
- Replacing a full browser automation tool

---

## Quick Start

### 1. Install

Pick one method:

```bash
# Go (recommended)
go install github.com/paimonchan/paimon-mcp-fetch/cmd/paimon-mcp-fetch@latest

# Homebrew (macOS/Linux)
brew tap paimonchan/tap
brew install paimon-mcp-fetch

# Scoop (Windows)
scoop bucket add paimonchan https://github.com/paimonchan/scoop-bucket
scoop install paimon-mcp-fetch

# Winget (Windows)
winget install paimonchan.paimon-mcp-fetch

# Docker
docker run -i --rm ghcr.io/paimonchan/paimon-mcp-fetch:latest
```

### 2. Configure Your AI Assistant

Add this to your MCP client config:

```json
{
  "mcp": {
    "paimon-mcp-fetch": {
      "type": "local",
      "command": ["paimon-mcp-fetch"],
      "enabled": true
    }
  }
}
```

### 3. Done

Your AI can now read any URL you give it.

---

## Why This Over Other Fetch Tools?

| | paimon-mcp-fetch | Basic text fetch |
|--|------------------|------------------|
| **Output** | Structured markdown | Plain text |
| **Article extraction** | Readability algorithm (strips ads, nav, sidebars) | Raw HTML body |
| **Images** | Optional extraction + processing | None |
| **JS rendering** | Optional (headless Chrome) | Static only |
| **Caching** | Built-in LRU cache | None |
| **Rate limiting** | Per-domain, configurable | None |
| **SSRF protection** | 7-layer defense | None |
| **Startup time** | ~5ms | Varies |
| **Memory** | ~10MB | Varies |

---

## Configuration

Everything is controlled via environment variables. You probably don't need to change anything — defaults work well for most use cases.

| Variable | Default | What it does |
|----------|---------|--------------|
| `PAIMON_MCP_FETCH_TIMEOUT_MS` | `12000` | Request timeout (ms) |
| `PAIMON_MCP_FETCH_MAX_HTML_BYTES` | `10485760` | Max page size (10MB) |
| `PAIMON_MCP_FETCH_CACHE_TTL_SECS` | `300` | Cache lifetime (5 min) |
| `PAIMON_MCP_FETCH_RATE_LIMIT_PER_SECOND` | `5.0` | Requests/sec per domain |
| `PAIMON_MCP_FETCH_RATE_LIMIT_BURST` | `10` | Max burst size |
| `PAIMON_MCP_FETCH_RETRY_MAX_ATTEMPTS` | `3` | Retry on transient errors |
| `PAIMON_MCP_FETCH_JS_RENDER_ENABLED` | `false` | Enable headless Chrome |

---

## Optional Features

### Image Processing

Extract and process images from webpages:

```bash
go build -tags image -o paimon-mcp-fetch ./cmd/paimon-mcp-fetch/
```

### JS Rendering

For JavaScript-heavy sites (SPAs, dynamic content):

```bash
go build -tags jsrender -o paimon-mcp-fetch ./cmd/paimon-mcp-fetch/
PAIMON_MCP_FETCH_JS_RENDER_ENABLED=true ./paimon-mcp-fetch
```

**Note:** Requires Chrome or Chromium installed. Slower (~3-5s/page) but handles sites that static fetch can't.

---

## Supported AI Assistants

Works with any MCP-compatible client:

| Client | Config File |
|--------|-------------|
| OpenCode | `~/.config/opencode/opencode.json` |
| Claude Desktop | `claude_desktop_config.json` |
| Cursor | `.cursor/mcp.json` |
| Cline | `.cline/mcp.json` |
| Windsurf | `.windsurf/mcp.json` |
| VS Code | `mcp.json` |

---

## Security

Built with security in mind from day one:

- **SSRF Protection** — 7-layer defense against server-side request forgery
- **Private IP blocking** — Can't access localhost or internal networks
- **Redirect validation** — Every redirect hop re-checked for safety
- **Size limits** — Stream-based reading, no memory bombs
- **Timeouts** — All requests have deadlines
- **No secrets in logs** — API keys and tokens never logged

### Windows Antivirus

Some antivirus may flag unsigned Go binaries as a false positive. This is a known industry issue. Solutions:

1. Use `go install` — antivirus sees the signed Go compiler
2. Use Docker — no local binary
3. Build from source — verify the code yourself

---

## Architecture

Built with Clean Architecture principles:

```
MCP Server → UseCase → Domain (entities, ports)
                        ↑
            Adapters implement interfaces
```

- **Domain** — Business rules, zero external dependencies
- **UseCase** — Orchestration logic
- **Adapters** — HTTP client, content extractor, cache, rate limiter, image processor, JS renderer

Full details in the [project plan](https://github.com/paimonchan/mcp-plan).

---

## License

MIT — do whatever you want with it.

---

> Built with Go. Zero runtime dependencies. Single binary. ~10MB RAM. Starts in 5ms.
