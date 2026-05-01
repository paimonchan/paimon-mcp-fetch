#!/usr/bin/env pwsh
#requires -Version 5.1
<#
.SYNOPSIS
    Install paimon-mcp-fetch — Web content fetching MCP server.
.DESCRIPTION
    Downloads the latest release binary for your OS/architecture and installs it to
    ~/.local/bin (or ~/bin on Windows) and adds it to PATH if needed.
.NOTES
    Run with: irm https://raw.githubusercontent.com/user/paimon-mcp-fetch/main/install.ps1 | iex
#>

$ErrorActionPreference = "Stop"

$Repo = "user/paimon-mcp-fetch"
$BinaryName = "paimon-mcp-fetch"

function Write-Info($msg) { Write-Host "[install] $msg" -ForegroundColor Cyan }
function Write-Ok($msg) { Write-Host "[install] $msg" -ForegroundColor Green }
function Write-Warn($msg) { Write-Host "[install] $msg" -ForegroundColor Yellow }

# Detect OS and architecture
$GoOS = "windows"
$GoArch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }

# Detect ARM64 Windows
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    $GoArch = "arm64"
}

$Suffix = "$GoOS-$GoArch"
$AssetName = "$BinaryName-$Suffix.exe"

# Determine install directory
$InstallDir = if ($env:LOCALAPPDATA) {
    Join-Path $env:LOCALAPPDATA "bin"
} else {
    Join-Path $env:USERPROFILE "bin"
}

if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$InstallPath = Join-Path $InstallDir "$BinaryName.exe"

Write-Info "Detected platform: $Suffix"
Write-Info "Install directory: $InstallDir"

# Get latest release
$ApiUrl = "https://api.github.com/repos/$Repo/releases/latest"
Write-Info "Fetching latest release info..."
try {
    $Release = Invoke-RestMethod -Uri $ApiUrl -UseBasicParsing -TimeoutSec 30
} catch {
    Write-Error "Failed to fetch release info: $_"
    exit 1
}

$Version = $Release.tag_name
Write-Info "Latest version: $Version"

# Find asset
$Asset = $Release.assets | Where-Object { $_.name -eq $AssetName }
if (-not $Asset) {
    Write-Error "Could not find asset '$AssetName' in release $Version."
    Write-Error "Available assets: $($Release.assets.name -join ', ')"
    exit 1
}

# Download
Write-Info "Downloading $AssetName..."
$TempFile = [System.IO.Path]::GetTempFileName() + ".exe"
try {
    Invoke-WebRequest -Uri $Asset.browser_download_url -OutFile $TempFile -UseBasicParsing -TimeoutSec 120
} catch {
    Write-Error "Download failed: $_"
    exit 1
}

# Verify checksum (optional)
$ChecksumAsset = $Release.assets | Where-Object { $_.name -eq "$AssetName.sha256" }
if ($ChecksumAsset) {
    Write-Info "Verifying checksum..."
    $ChecksumFile = [System.IO.Path]::GetTempFileName()
    Invoke-WebRequest -Uri $ChecksumAsset.browser_download_url -OutFile $ChecksumFile -UseBasicParsing -TimeoutSec 30
    $ExpectedChecksum = (Get-Content $ChecksumFile -Raw).Trim().Split()[0]
    $ActualChecksum = (Get-FileHash -Path $TempFile -Algorithm SHA256).Hash.ToLower()
    if ($ExpectedChecksum -ne $ActualChecksum) {
        Write-Error "Checksum mismatch! Expected: $ExpectedChecksum, Got: $ActualChecksum"
        exit 1
    }
    Write-Ok "Checksum verified."
    Remove-Item $ChecksumFile -Force
}

# Install
Write-Info "Installing to $InstallPath..."
Move-Item -Path $TempFile -Destination $InstallPath -Force

# Add to PATH if needed
$CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($CurrentPath -notlike "*$InstallDir*") {
    Write-Info "Adding $InstallDir to your PATH..."
    $NewPath = "$CurrentPath;$InstallDir"
    [Environment]::SetEnvironmentVariable("PATH", $NewPath, "User")
    Write-Warn "PATH updated. Please restart your terminal or run: `$env:PATH = [Environment]::GetEnvironmentVariable('PATH', 'User')`"
}

Write-Ok "paimon-mcp-fetch $Version installed successfully!"
Write-Info "Binary location: $InstallPath"
Write-Info "Add this to your MCP client config:"
Write-Host @"

  {
    "mcpServers": {
      "fetch": {
        "command": "$BinaryName"
      }
    }
  }

"@ -ForegroundColor Gray
