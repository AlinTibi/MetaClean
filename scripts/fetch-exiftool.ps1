<#
.SYNOPSIS
    Downloads the pinned, official Windows ExifTool build into tools/exiftool/
    for local development (`wails dev`, `go test`, manual runs) and for the
    release workflow.

.DESCRIPTION
    MetaClean never bundles the ExifTool binary in source control. This
    script downloads the official Windows distribution directly from its
    publisher, Phil Harvey (exiftool.org) — whose own download page links
    directly to this exact SourceForge release file, his chosen file host
    for the Windows build, not a third-party mirror.

    Integrity is checked against the SHA-256 Phil Harvey himself publishes
    at https://exiftool.org/checksums.txt for this exact archive, verified
    BEFORE extraction — so a redirected HTML page, a corrupted download, or
    a tampered file is rejected outright rather than silently unpacked.

    The official zip does not place "exiftool(-k).exe" and "exiftool_files/"
    at its root: it extracts into a subdirectory named after the archive
    (e.g. "exiftool-13.59_64/"). This script searches the extracted tree
    for those two items rather than assuming a fixed path, so a layout
    change in a future release doesn't silently break it.

    This script makes no changes outside tools/exiftool/ and nothing it
    downloads is ever committed. The release workflow
    (.github/workflows/release.yml) runs this exact same script.

.NOTES
    If this script fails to reach SourceForge (some networks/CI runners
    geo-gate or rate-limit it), download exiftool-13.59_64.zip yourself
    from https://exiftool.org, verify its SHA-256 against
    https://exiftool.org/checksums.txt, and extract "exiftool(-k).exe"
    (renamed to exiftool.exe) plus "exiftool_files/" into tools/exiftool/.

    .github/workflows/test-exiftool-fetch.yml runs this script standalone
    on a real GitHub Actions windows-latest runner (workflow_dispatch) as
    a CI-safe way to verify it end to end, independent of this script's
    own author's local network.
#>

$ErrorActionPreference = 'Stop'

# Pin: ExifTool 13.59, Windows 64-bit build, as published at
# https://exiftool.org (download page links directly to this SourceForge
# release file).
$ExifToolVersion = '13.59'
$ZipFileName = "exiftool-$ExifToolVersion`_64.zip"
$DownloadUrl = "https://sourceforge.net/projects/exiftool/files/$ZipFileName/download"

# SHA-256 of exiftool-13.59_64.zip itself (the archive, before
# extraction), copied from https://exiftool.org/checksums.txt — the
# checksum list Phil Harvey publishes alongside the official release.
$PinnedZipSha256 = '44b512b25af500724ba579d0a53c8fc5851628b692dd5e5d94ae4a15c2cba9ec'

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
$downloadOk = $false
for ($attempt = 1; $attempt -le 3; $attempt++) {
    try {
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $zipPath -UserAgent 'Mozilla/5.0' -MaximumRedirection 10
        $downloadOk = $true
        break
    } catch {
        $statusCode = $_.Exception.Response?.StatusCode
        Write-Warning "Attempt $attempt failed: [$($_.Exception.GetType().FullName)] $($_.Exception.Message) (HTTP status: $statusCode)"
        Start-Sleep -Seconds (2 * $attempt)
    }
}
if (-not $downloadOk) {
    Write-Error "Failed to download ExifTool from $DownloadUrl after 3 attempts.`nSee the NOTES in this script for a manual fallback."
    exit 1
}

# The official SourceForge download endpoint has been observed to return
# an HTML cookie-consent/mirror-selection page instead of the archive for
# some networks. Catch that explicitly, with a clear error, instead of
# trusting whatever bytes were returned as if they were the real zip.
$headerBytes = New-Object byte[] 200
$stream = [System.IO.File]::OpenRead($zipPath)
try {
    $readCount = $stream.Read($headerBytes, 0, $headerBytes.Length)
} finally {
    $stream.Close()
}
$isZip = $readCount -ge 4 -and $headerBytes[0] -eq 0x50 -and $headerBytes[1] -eq 0x4B -and ($headerBytes[2] -eq 0x03 -or $headerBytes[2] -eq 0x05 -or $headerBytes[2] -eq 0x07)
if (-not $isZip) {
    $preview = [System.Text.Encoding]::ASCII.GetString($headerBytes, 0, $readCount)
    Write-Error "Downloaded file is not a ZIP archive (no PK magic bytes) — SourceForge likely returned an HTML page instead of the file.`nFirst bytes: $preview`nSee the NOTES in this script for a manual fallback."
    exit 1
}

$actualZipHash = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash.ToLower()
if ($actualZipHash -ne $PinnedZipSha256) {
    Write-Error "SHA-256 mismatch for $ZipFileName !`nExpected: $PinnedZipSha256`nActual:   $actualZipHash`nRefusing to extract an archive that doesn't match the official published checksum (https://exiftool.org/checksums.txt)."
    exit 1
}
Write-Output "SHA-256 verified against https://exiftool.org/checksums.txt"

Write-Output "Extracting ..."
Expand-Archive -Path $zipPath -DestinationPath $tempDir -Force

# The official zip extracts into a subdirectory (e.g.
# "exiftool-13.59_64/"), not directly to the zip root. Search for the two
# items we need rather than assuming a fixed path, so a layout change in
# a future release doesn't silently break this script.
$foundExe = Get-ChildItem -Path $tempDir -Recurse -File -Filter 'exiftool(-k).exe' | Select-Object -First 1
$foundFilesDir = Get-ChildItem -Path $tempDir -Recurse -Directory -Filter 'exiftool_files' | Select-Object -First 1

if ($null -eq $foundExe) {
    Write-Error "Could not find 'exiftool(-k).exe' anywhere in the extracted archive. Contents:`n$(Get-ChildItem -Path $tempDir -Recurse -Name | Out-String)"
    exit 1
}
if ($null -eq $foundFilesDir) {
    Write-Error "Could not find an 'exiftool_files' directory anywhere in the extracted archive. Contents:`n$(Get-ChildItem -Path $tempDir -Recurse -Name | Out-String)"
    exit 1
}

New-Item -ItemType Directory -Force -Path $toolsDir | Out-Null
Copy-Item $foundExe.FullName (Join-Path $toolsDir 'exiftool.exe')
Copy-Item -Recurse $foundFilesDir.FullName (Join-Path $toolsDir 'exiftool_files')

Remove-Item -Recurse -Force $tempDir

$version = & (Join-Path $toolsDir 'exiftool.exe') -ver
if ($version.Trim() -ne $ExifToolVersion) {
    Write-Error "Installed exiftool.exe reports version '$version', expected '$ExifToolVersion'."
    exit 1
}
Write-Output "Installed ExifTool $version into $toolsDir (archive SHA-256 verified)"
