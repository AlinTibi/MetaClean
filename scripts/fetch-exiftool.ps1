<#
.SYNOPSIS
    Downloads the pinned, official Windows ExifTool build into tools/exiftool/
    for local development (`wails dev`, `go test`, manual runs).

.DESCRIPTION
    MetaClean never bundles the ExifTool binary in source control. This
    script downloads the official Windows distribution directly from its
    publisher, Phil Harvey (exiftool.org), whose own download page links
    directly to this exact SourceForge release file — SourceForge is his
    chosen file host for the Windows build, not a third-party mirror of
    unknown provenance. The downloaded executable's SHA-256 is checked
    against a pinned value before anything is extracted into tools/exiftool/,
    which is gitignored and picked up automatically by
    internal/exiftool.Locate() during local development.

    This script makes no changes outside tools/exiftool/ and nothing it
    downloads is ever committed. The release workflow
    (.github/workflows/release.yml) uses the same pinned version and hash.

.NOTES
    If this script fails to reach SourceForge (some networks/CI runners
    geo-gate or rate-limit it), download exiftool-13.59_64.zip yourself
    from https://exiftool.org, verify its SHA-256, and extract
    "exiftool(-k).exe" (renamed to exiftool.exe) plus "exiftool_files/"
    into tools/exiftool/.
#>

$ErrorActionPreference = 'Stop'

# Pin: ExifTool 13.59 Windows 64-bit, as published at https://exiftool.org
# (download page links directly to this SourceForge release file).
$ExifToolVersion = '13.59'
$ZipFileName = "exiftool-$ExifToolVersion`_64.zip"
$DownloadUrl = "https://sourceforge.net/projects/exiftool/files/$ZipFileName/download"

# SHA-256 of the extracted exiftool(-k).exe for this exact pinned version,
# computed once from a verified-authentic copy of the official binary and
# checked on every fetch so a corrupted, tampered, or unexpectedly
# different download is rejected instead of silently installed.
$PinnedExeSha256 = '68c079c32fdae0d6c7130e9a5fb73f8ac9dabdf9ab8da312da4f6c549d6d3385'

$repoRoot = Split-Path -Parent $PSScriptRoot
$toolsDir = Join-Path $repoRoot 'tools\exiftool'
$tempDir = Join-Path $env:TEMP "metaclean-exiftool-fetch"

if (Test-Path $toolsDir) {
    Write-Output "tools/exiftool already exists at $toolsDir — delete it first if you want to re-fetch."
    exit 0
}

if (Test-Path $tempDir) { Remove-Item -Recurse -Force $tempDir }
New-Item -ItemType Directory -Force -Path $tempDir | Out-Null
$zipPath = Join-Path $tempDir $ZipFileName

Write-Output "Downloading $DownloadUrl ..."
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $zipPath -UserAgent 'Mozilla/5.0' -MaximumRedirection 10
} catch {
    Write-Error "Failed to download ExifTool from $DownloadUrl : $_`nSee the NOTES in this script for a manual fallback."
    exit 1
}

Write-Output "Extracting ..."
Expand-Archive -Path $zipPath -DestinationPath $tempDir -Force

$extractedExe = Join-Path $tempDir 'exiftool(-k).exe'
if (-not (Test-Path $extractedExe)) {
    Write-Error "Expected 'exiftool(-k).exe' was not found after extraction; the official zip layout may have changed. Contents: $(Get-ChildItem $tempDir -Name)"
    exit 1
}

$actualHash = (Get-FileHash -Path $extractedExe -Algorithm SHA256).Hash.ToLower()
if ($actualHash -ne $PinnedExeSha256) {
    Write-Error "SHA-256 mismatch for exiftool.exe!`nExpected: $PinnedExeSha256`nActual:   $actualHash`nRefusing to install a binary that doesn't match the pinned, verified hash."
    exit 1
}

New-Item -ItemType Directory -Force -Path $toolsDir | Out-Null
Copy-Item $extractedExe (Join-Path $toolsDir 'exiftool.exe')
Copy-Item -Recurse (Join-Path $tempDir 'exiftool_files') (Join-Path $toolsDir 'exiftool_files')

Remove-Item -Recurse -Force $tempDir

$version = & (Join-Path $toolsDir 'exiftool.exe') -ver
Write-Output "Installed ExifTool $version into $toolsDir (SHA-256 verified)"
