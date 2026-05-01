# paimon-mcp-fetch

> Web content fetching MCP server built with Go.
> Replaces `@kazuph/mcp-fetch` with a single binary, zero runtime dependencies.

## Features

- **Web Content Extraction**: Fetches web pages and converts HTML to markdown
- **Article Title Extraction**: Extracts and displays article titles
- **Image Processing** (optional, `go build -tags image`): Download, resize, merge, and optimize images
- **SSRF Protection**: 7-layer defense in depth against server-side request forgery
- **Robots.txt Compliance**: Respects robots.txt by default (fail-open with timeout)
- **Pagination Support**: Read large pages in chunks via `start_index`
- **Caching**: In-memory LRU cache with TTL
- **Rate Limiting**: Per-domain token bucket
- **Clean Architecture**: Domain → UseCase → Adapter layers

## Installation

> **Note for Windows users:** Go binaries are often flagged by antivirus as false positives ([read why](#-windows-antivirus-warning)). For the smoothest experience, we recommend **Package Manager** or **`go install`** methods.

### Option 1: Package Manager (Recommended — Best AV Compatibility)

Installing via a package manager significantly reduces antivirus false positives because the installation comes through an audited, standardized pipeline.

**Scoop** (Windows):
```powershell
scoop install paimon-mcp-fetch
```

**Winget** (Windows):
```powershell
winget install paimon-mcp-fetch
```

**Homebrew** (macOS / Linux):
```bash
brew install paimon-mcp-fetch
```

> Package manager manifests are included in this repository. To submit to official repositories, see:
> - Scoop: `scoop/paimon-mcp-fetch.json`
> - Winget: `winget/manifests/...`
> - Homebrew: `homebrew/paimon-mcp-fetch.rb`

---

### Option 2: `go install` (For Developers — AV-Safe)

If you already have [Go](https://go.dev/dl/) installed, this is the **safest method** with zero AV issues:

```bash
go install github.com/paimonchan/paimon-mcp-fetch/cmd/paimon-mcp-fetch@latest
```

The binary will be placed in your `$(go env GOPATH)/bin`. Make sure this directory is in your PATH.

**Why this is safer:** Your antivirus sees `go.exe` (signed by Google) doing the work, not an unknown binary.

---

### Option 3: Install Script (Easiest — May Trigger AV Warning)

One-line install that auto-detects your OS and architecture:

**Windows (PowerShell)**:
```powershell
irm https://raw.githubusercontent.com/user/paimon-mcp-fetch/main/install.ps1 | iex
```

**macOS / Linux (Bash)**:
```bash
curl -fsSL https://raw.githubusercontent.com/user/paimon-mcp-fetch/main/install.sh | sh
```

> ⚠️ Windows users: Your antivirus may flag the downloaded `.exe`. See our [AV explanation & workarounds](#-windows-antivirus-warning).

---

### Option 4: Download from GitHub Releases

Download the pre-built binary for your OS from [GitHub Releases](https://github.com/paimonchan/paimon-mcp-fetch/releases).

Extract and place it somewhere in your PATH.

> ⚠️ Same AV warning as Option 3 — this is a raw unsigned binary.

---

### Option 5: Build from Source

Requires [Go 1.22+](https://go.dev/dl/).

```bash
git clone https://github.com/paimonchan/paimon-mcp-fetch
cd paimon-mcp-fetch
go build -ldflags="-s -w" -o paimon-mcp-fetch ./cmd/paimon-mcp-fetch/
```

**Why build yourself?** You can verify the source code and produce a byte-for-byte identical binary to our releases. See [Reproducible Builds](https://github.com/paimonchan/paimon-mcp-fetch/blob/main/.github/workflows/build.yml).

---

### Option 6: Docker (Future — No AV Issues)

Coming in a future release. Will run entirely in a container with no local binary.

```bash
docker run -i --rm ghcr.io/user/paimon-mcp-fetch:latest
```

---

## ⚠️ Windows Antivirus Warning

**This is a false positive — our code is 100% safe and open source.**

Windows Defender and some antivirus programs (Avast, AVG, etc.) may flag the `paimon-mcp-fetch.exe` binary as `IDP.Generic` or similar. This happens because:

- Go compiles to a **single self-contained binary** without a digital signature
- The binary contains network operations (`net/http`) and file access patterns
- Antivirus uses **heuristic detection** that matches these patterns to potential malware

**Our mitigation:**
- ✅ Full source code is public and auditable in this repository
- ✅ You can [build from source](#option-4-build-from-source) and verify the binary yourself
- ✅ Public CI/CD build logs via GitHub Actions
- ✅ Submit false positive reports to antivirus vendors (Phase 4)

> **Note:** We do not have a budget for a code signing certificate (~$200-500/year), which is the only guaranteed fix for this issue. Package manager installs (Scoop, Winget) may reduce but not eliminate these warnings.

**Workaround:** Add an exclusion in your antivirus for the install directory, or use [Docker](#option-3-download-from-github-releases) as an alternative.

---

## MCP Client Configuration

After installation, add to your MCP client config:

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

### OpenCode

`.opencode/config.json`:
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

### VS Code

`.vscode/mcp.json`:
```json
{
  "mcp": {
    "servers": {
      "fetch": {
        "command": "paimon-mcp-fetch"
      }
    }
  }
}
```

### Cursor / Cline / Windsurf

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

| Variable | Default | Description |
|----------|---------|-------------|
| `PAIMON_MCP_FETCH_TIMEOUT_MS` | 12000 | Request timeout in milliseconds |
| `PAIMON_MCP_FETCH_MAX_REDIRECTS` | 5 | Maximum redirects to follow |
| `PAIMON_MCP_FETCH_MAX_HTML_BYTES` | 2097152 | Max HTML response size (2MB) |
| `PAIMON_MCP_FETCH_MAX_IMAGE_BYTES` | 10485760 | Max image size (10MB) |
| `PAIMON_MCP_FETCH_DISABLE_SSRF` | false | Disable SSRF guard |
| `PAIMON_MCP_FETCH_CACHE_ENABLED` | true | Enable response cache |
| `PAIMON_MCP_FETCH_CACHE_TTL_SECS` | 300 | Cache TTL in seconds |
| `PAIMON_MCP_FETCH_CACHE_MAX_ENTRIES` | 100 | Maximum cache entries |
| `PAIMON_MCP_FETCH_RATE_LIMIT_ENABLED` | true | Enable per-domain rate limiting |
| `PAIMON_MCP_FETCH_RATE_LIMIT_PER_SECOND` | 1.0 | Requests per second per domain |
| `PAIMON_MCP_FETCH_RATE_LIMIT_BURST` | 3 | Max burst size per domain |
| `PAIMON_MCP_FETCH_RETRY_MAX_ATTEMPTS` | 3 | Max retry attempts for transient errors |
| `PAIMON_MCP_FETCH_RETRY_BASE_DELAY_MS` | 500 | Base retry delay (exponential backoff) |
| `PAIMON_MCP_FETCH_RETRY_MAX_DELAY_MS` | 10000 | Max retry delay cap |

---

## Security & Trust

### Why a Go Binary?

Unlike TypeScript (`npx`) or Python (`uvx`) MCP servers, `paimon-mcp-fetch` is a **single compiled binary** with **zero runtime dependencies**. This means:

- **No Node.js, no Python, no Docker required**
- **Startup in ~5ms** (vs 500ms–2s for Node.js)
- **Memory usage ~8–15MB** (vs 50–100MB for Node.js)
- **One file to download and run**

### Addressing the "Unknown Binary" Concern

We understand that running a pre-built binary requires trust. Here's how we address that:

1. **Fully Open Source** — This repository is public under MIT license. You can read every line of code.
2. **Build It Yourself** — Anyone can clone this repo and run `go build` to produce an **identical** binary. See [Build from Source](#option-4-build-from-source).
3. **Reproducible Builds** — Release binaries are built via public GitHub Actions. The workflow is visible in `.github/workflows/`.
4. **Checksum Verification** — Every GitHub Release includes SHA256 checksums for all binaries.
5. **Package Manager Distribution** — Installing via Scoop, Winget, or Homebrew provides an audited installation path.

### Security Features

- **SSRF Protection**: 7-layer defense (URL parsing, scheme validation, hostname blocklist, DNS resolution, private IP filtering, redirect re-validation, stream limits)
- **Robots.txt**: Respected by default with fail-open behavior
- **Size Limits**: Stream-based reading with configurable byte limits
- **Timeouts**: All network calls have context deadlines
- **No Secrets in Logs**: API keys and tokens are never logged

---

## MCP Tool Schema

### `fetch`

Fetches a URL and returns content as markdown.

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
    "ignoreRobotsTxt": false
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

---

## License

MIT
