# paimon-mcp-fetch — TODO

> Web content fetching MCP server built with Go.
> Plan: [`../plans/MCP-001-paimon-mcp-fetch.md`](../plans/MCP-001-paimon-mcp-fetch.md)

---

## Phase 1 — Core (MVP)

- [x] **1.1 Project initialization**
  - [x] `go mod init github.com/<user>/paimon-mcp-fetch`
  - [x] Create folder structure (`cmd/`, `internal/`, `testdata/`)
  - [x] `.gitignore`
  - [x] `README.md` (with Security & Trust section)
  - [x] `Makefile` (Windows-compatible)

- [x] **1.2 Domain layer** (`internal/domain/`)
  - [x] `entity.go` — FetchRequest, FetchResult, ImageResult, ContentBlock, TextOptions, ImageOptions, SecurityOptions
  - [x] `errors.go` — sentinel errors + ErrorType constants + FetchError struct
  - [x] `port.go` — ContentFetcher, ContentExtractor, ImageProcessor, RobotsChecker, CacheStore interfaces
  - [x] `policy.go` — SSRFPolicy, SizePolicy, CachePolicy defaults

- [x] **1.3 Config** (`internal/config/`)
  - [x] `config.go` — env vars with `PAIMON_MCP_FETCH_*` prefix, defaults, validation

- [x] **1.4 HTTP Fetcher adapter** (`internal/adapter/fetcher/`) ✅ REAL
  - [x] `ssrf.go` — 7-layer SSRF guard (URL parse, scheme, hostname, DNS, IPv4-mapped IPv6, private IP, userinfo rejection)
  - [x] `redirect.go` — manual redirect following with per-hop SSRF re-check (max 5)
  - [x] `client.go` — net/http client implementing `domain.ContentFetcher`, stream-based body reading with size limit
  - [x] `ssrf_test.go` — 8 tests (valid URL, invalid scheme, userinfo, localhost, private IP, IPv4-mapped IPv6, isPrivateIP, redirect tracker)

- [x] **1.5 Content Extractor adapter** (`internal/adapter/extractor/`)
  - [x] `extractor.go` — go-readability v2 + html-to-markdown v2 wrapper
  - [x] `extractor_test.go` — tests with fixture HTML files (4 tests, all pass)

- [x] **1.6 Robots Checker adapter** (`internal/adapter/robots/`)
  - [x] `checker.go` — robotstxt wrapper, per-host cache, fail-open with 5s timeout
  - [x] `checker_test.go` — 9 integration tests against httptest server (all pass)

- [x] **1.7 Use Case** (`internal/usecase/`)
  - [x] `fetch.go` — FetchUseCase with constructor injection, 9-step orchestration flow
  - [x] `fetch_test.go` — 4 tests (paginate, remainingImages, validateRequest, cacheKey)

- [x] **1.8 MCP Server adapter** (`internal/adapter/mcpserver/`)
  - [x] `server.go` — mcp-go wiring, `fetch` tool + `fetch` prompt registration, request parsing

- [x] **1.9 Entry point** (`cmd/paimon-mcp-fetch/main.go`)
  - [x] Wire all adapters + use case
  - [x] Graceful shutdown with `signal.NotifyContext`

- [x] **1.10 Integration tests**
  - [x] Fetcher SSRF tests (8 tests)
  - [x] Extractor tests (4 tests)
  - [x] Robots checker tests (9 tests)
  - [x] Use case tests (4 tests)

---

## Phase 2 — Polish + Distribution

- [x] **2.1 Cache** (`internal/adapter/cache/`)
  - [x] `memory.go` — LRU cache with TTL, typed CacheEntry
  - [x] `memory_test.go` — LRU behavior tests

- [x] **2.2 Rate Limiting**
  - [x] Token bucket per domain (custom implementation)
  - [x] Configurable rate + burst

- [x] **2.3 Retry**
  - [x] Exponential backoff for transient errors
  - [x] Max retries config (default 3)

- [x] **2.4 Structured error mapping**
  - [x] Domain error → MCP error with helpful messages

- [x] **2.5 Pagination support**
  - [x] `start_index` + `max_length` in tool response with continuation prompt

- [x] **2.6 GitHub Actions CI/CD**
  - [x] `.github/workflows/release.yml` — auto-build + upload binaries on tag push

- [x] **2.7 GitHub Releases**
  - [x] Cross-platform binaries: Windows (.exe), Linux, macOS (Intel + Apple Silicon)
  - [x] SHA256 checksums for all assets

- [x] **2.8 Install Scripts**
  - [x] `install.ps1` — Windows PowerShell (auto-detect arch, download, install to PATH)
  - [x] `install.sh` — Linux/macOS Bash (auto-detect arch, download, install to ~/.local/bin)

---

## Phase 3 — Image Support + Package Managers

- [x] **3.1 Image Processor adapter** (`internal/adapter/image/`)
  - [x] `processor.go` — image fetch, resize, merge vertically, JPEG encode
  - [x] Support `base64`, `file`, `both` output
  - [x] Support `merged`, `individual`, `both` layout
  - [x] Save to `~/Downloads/paimon-mcp-fetch/YYYY-MM-DD/`

- [x] **3.2 Image tests**
  - [x] `processor_test.go` — tests with httptest server

- [x] **3.3 Scoop** (Windows)
  - [x] Create `paimon-mcp-fetch.json` manifest

- [x] **3.4 Winget** (Windows)
  - [x] Create manifest YAMLs

- [x] **3.5 Homebrew** (macOS / Linux)
  - [x] Create Homebrew formula

---

## Phase 4 — Advanced & Distribution Polish

- [ ] **4.1 JS Rendering** (`//go:build jsrender`)
  - [ ] chromedp integration for dynamic content

- [ ] **4.2 Disk cache**
  - [ ] bbolt or sqlite option

- [ ] **4.3 Observability**
  - [ ] Request count, latency, error rate metrics

- [ ] **4.4 Docker** (Optional alternative distribution)
  - [ ] `Dockerfile` with multi-stage build
  - [ ] Publish to GitHub Container Registry (`ghcr.io/user/paimon-mcp-fetch`)

- [ ] **4.5 Submit False Positive Reports** (Last resort — no budget for code signing)
  - [ ] Submit to Microsoft Defender (https://www.microsoft.com/en-us/wdsi/filesubmission)
  - [ ] Submit to Avast/AVG (https://www.avast.com/false-positive-file-form.php)
  - [ ] Document VirusTotal scan results in README
  - [ ] Note: Code signing certificate skipped due to budget constraints (~$200-500/yr)

---

## Reference Implementations

- Official: https://github.com/modelcontextprotocol/servers/tree/main/src/fetch
- @kazuph/mcp-fetch: https://github.com/kazuph/mcp-fetch

---

## Current Status

**Phase:** 3 — **COMPLETE** ✅  
**All tests passing across 7 packages**  
**Next:** Phase 4 — Advanced (JS rendering, disk cache, observability, Docker, AV reports)
