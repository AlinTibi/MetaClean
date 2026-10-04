<#
.SYNOPSIS
    Downloads the pinned ExifTool Windows build into tools/exiftool/ for
    local development (`wails dev`, `go test`, manual runs) and for the
    release workflow. Never runs at application runtime.

.DESCRIPTION
    MetaClean never bundles the ExifTool binary in source control, and the
    running application never downloads anything — this script only runs
    during local development setup and in CI/release builds.

    ExifTool is Phil Harvey's software (https://exiftool.org). His
    official Windows build is published via a SourceForge file linked
    directly from that site. Earlier versions of this script downloaded
    that file directly, but SourceForge returns HTTP 403 Forbidden to
    this exact download for automated/CI traffic — confirmed by directly
    testing from a real GitHub Actions windows-latest runner (see
    .github/workflows/test-exiftool-fetch.yml's run history on this
    repository), not just this developer's local network. A release
    pipeline that fails every single time it runs is worse than one that
    sources the binary differently, so this script instead downloads the
    exiftool-vendored.exe npm package: a transparent, open-source
    (MIT-licensed) repackaging that vendors Phil Harvey's official
    Windows build UNMODIFIED, maintained by Matthew McEachen /
    PhotoStructure (github.com/photostructure/exiftool-vendored.exe).
    ExifTool itself, inside that package, retains its own license — see
    THIRD_PARTY_NOTICES.md.

    This script does NOT claim to download directly from exiftool.org/
    SourceForge, and does not verify the npm-sourced binary against the
    official ZIP's checksum — different packaging (tar.gz vs zip) means
    their checksums are never expected to match, so comparing them would
    be meaningless. Instead:
      - the downloaded npm tarball is verified against the SHA-512 npm's
        own registry publishes for this exact package version (detects
        transit corruption or a compromised/substituted download)
      - the official ExifTool 13.59 checksums Phil Harvey publishes at
        https://exiftool.org/checksums.txt are recorded in
        THIRD_PARTY_NOTICES.md purely as upstream provenance reference,
        not as something this script checks the npm artifact against
      - the extracted binary's own "-ver" output is checked against the
        pinned ExifTool version as a basic sanity check that the
        vendored payload is what it claims to be

    This script makes no changes outside tools/exiftool/ and nothing it
    downloads is ever committed. The release workflow
    (.github/workflows/release.yml) runs this exact same script.

.NOTES
    If npm's registry is unreachable, download
    exiftool-vendored.exe-<version>.tgz yourself from
    https://registry.npmjs.org/exiftool-vendored.exe, verify it, and
    extract package/bin/exiftool.exe plus package/bin/exiftool_files/
    into tools/exiftool/.
#>

$ErrorActionPreference = 'Stop'

# Pin: the exiftool-vendored.exe npm package version that vendors
# ExifTool 13.59 exactly (its own version number mirrors the ExifTool
# version it bundles, with a patch suffix).
$ExifToolVersion = '13.59'
$PackageVersion = '13.59.3'
$TarballName = "exiftool-vendored.exe-$PackageVersion.tgz"
$DownloadUrl = "https://registry.npmjs.org/exiftool-vendored.exe/-/$TarballName"

# SHA-512 npm's registry publishes for this exact tarball (its "dist.integrity"
# field), copied from https://registry.npmjs.org/exiftool-vendored.exe/$PackageVersion.
# This verifies the npm download itself was not corrupted or substituted;
# see the file header for why it is not compared against the official
# ExifTool ZIP's own (differently-packaged) checksum.
$PinnedSha512Base64 = 'F5hpk1yVGZSDHjSKsyumyniY56IM3zawKOejOqtyT2r476fvWrUxpBmWT7Ena104TGPt/2+448Q/vrVt+yOzwQ=='

$repoRoot = Split-Path -Parent $PSScriptRoot
$toolsDir = Join-Path $repoRoot 'tools\exiftool'
$tempDir = Join-Path $env:TEMP "metaclean-exiftool-fetch"

if (Test-Path $toolsDir) {
    Write-Output "tools/exiftool already exists at $toolsDir — delete it first if you want to re-fetch."
    exit 0
}

if (Test-Path $tempDir) { Remove-Item -Recurse -Force $tempDir }
New-Item -ItemType Directory -Force -Path $tempDir | Out-Null
$tarballPath = Join-Path $tempDir $TarballName

Write-Output "Downloading $DownloadUrl ..."
$downloadOk = $false
for ($attempt = 1; $attempt -le 3; $attempt++) {
    try {
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $tarballPath -UserAgent 'Mozilla/5.0' -MaximumRedirection 10
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

$expectedBytes = [Convert]::FromBase64String($PinnedSha512Base64)
$expectedHex = -join ($expectedBytes | ForEach-Object { $_.ToString('x2') })
$actualHex = (Get-FileHash -Path $tarballPath -Algorithm SHA512).Hash.ToLower()
if ($actualHex -ne $expectedHex) {
    Write-Error "SHA-512 mismatch for $TarballName !`nExpected: $expectedHex`nActual:   $actualHex`nRefusing to extract a package that doesn't match npm's published integrity hash for exiftool-vendored.exe@$PackageVersion."
    exit 1
}
Write-Output "SHA-512 verified against npm's published package integrity hash."

Write-Output "Extracting ..."
tar xzf $tarballPath -C $tempDir

# npm tarballs always extract under a top-level "package/" directory, but
# search rather than hardcode the rest, so a future repackaging doesn't
# silently break this script.
$foundExe = Get-ChildItem -Path $tempDir -Recurse -File -Filter 'exiftool.exe' | Select-Object -First 1
$foundFilesDir = Get-ChildItem -Path $tempDir -Recurse -Directory -Filter 'exiftool_files' | Select-Object -First 1

if ($null -eq $foundExe) {
    Write-Error "Could not find 'exiftool.exe' anywhere in the extracted package. Contents:`n$(Get-ChildItem -Path $tempDir -Recurse -Name | Out-String)"
    exit 1
}
if ($null -eq $foundFilesDir) {
    Write-Error "Could not find an 'exiftool_files' directory anywhere in the extracted package. Contents:`n$(Get-ChildItem -Path $tempDir -Recurse -Name | Out-String)"
    exit 1
}

New-Item -ItemType Directory -Force -Path $toolsDir | Out-Null
Copy-Item $foundExe.FullName (Join-Path $toolsDir 'exiftool.exe')
Copy-Item -Recurse $foundFilesDir.FullName (Join-Path $toolsDir 'exiftool_files')

Remove-Item -Recurse -Force $tempDir

$version = (& (Join-Path $toolsDir 'exiftool.exe') -ver).Trim()
if ($version -ne $ExifToolVersion) {
    Write-Error "Installed exiftool.exe reports version '$version', expected '$ExifToolVersion'."
    exit 1
}
Write-Output "Installed ExifTool $version into $toolsDir (npm package integrity verified)"
