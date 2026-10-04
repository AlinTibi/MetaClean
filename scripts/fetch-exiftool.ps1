<#
.SYNOPSIS
    Downloads the pinned, official Windows ExifTool build into tools/exiftool/
    for local development (`wails dev`, `go test`, manual runs).

.DESCRIPTION
    MetaClean never bundles the ExifTool binary in source control. This
    script fetches the exact same pinned distribution the release workflow
    uses — the official Phil Harvey Windows build (exiftool.org), mirrored
    unmodified on npm by the exiftool-vendored.exe package — and extracts
    it into tools/exiftool/, which is gitignored and picked up automatically
    by internal/exiftool.Locate() during local development.

    This script makes no changes outside tools/exiftool/ and nothing it
    downloads is ever committed.
#>

$ErrorActionPreference = 'Stop'

$ExifToolNpmVersion = '13.59.3' # vendors the official ExifTool 13.59 Windows build

$repoRoot = Split-Path -Parent $PSScriptRoot
$toolsDir = Join-Path $repoRoot 'tools\exiftool'
$tempDir = Join-Path $env:TEMP "metaclean-exiftool-fetch"

if (Test-Path $toolsDir) {
    Write-Output "tools/exiftool already exists at $toolsDir — delete it first if you want to re-fetch."
    exit 0
}

New-Item -ItemType Directory -Force -Path $tempDir | Out-Null
$tgz = Join-Path $tempDir 'exiftool-vendored.tgz'
$url = "https://registry.npmjs.org/exiftool-vendored.exe/-/exiftool-vendored.exe-$ExifToolNpmVersion.tgz"

Write-Output "Downloading $url ..."
Invoke-WebRequest -Uri $url -OutFile $tgz

Write-Output "Extracting ..."
tar xzf $tgz -C $tempDir

New-Item -ItemType Directory -Force -Path $toolsDir | Out-Null
Copy-Item -Recurse -Force (Join-Path $tempDir 'package\bin\*') $toolsDir

Remove-Item -Recurse -Force $tempDir

$version = & (Join-Path $toolsDir 'exiftool.exe') -ver
Write-Output "Installed ExifTool $version into $toolsDir"
