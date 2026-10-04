# Third-Party Notices

MetaClean bundles **ExifTool** as its metadata engine — Phil Harvey's
software, not MetaClean's. ExifTool itself is not modified in any way.

## ExifTool

- **Author:** Phil Harvey
- **Homepage:** https://exiftool.org
- **Source:** https://github.com/exiftool/exiftool
- **Pinned version bundled with MetaClean releases:** 13.59
- **License:** ExifTool is free software; you may redistribute it and/or
  modify it under the same terms as Perl itself — either the
  [Perl Artistic License](https://dev.perl.org/licenses/artistic.html), or
  the [GNU General Public License](https://www.gnu.org/licenses/gpl-2.0.html),
  at your option. See the `LICENSE` file inside the bundled
  `exiftool/exiftool_files/` folder for the exact license text shipped
  with the binary.

### How MetaClean obtains it, and why

Phil Harvey publishes the official Windows build (with its bundled Perl
runtime) via a file linked directly from https://exiftool.org, hosted on
SourceForge. **MetaClean's build/release pipeline does not download
directly from that link.** SourceForge returns HTTP 403 Forbidden to that
exact download for automated/CI traffic — confirmed by testing from both
a local development machine and a real GitHub Actions `windows-latest`
runner, not a one-off network hiccup. A release pipeline that fails every
time it runs is worse than sourcing the binary a different way.

Instead, `scripts/fetch-exiftool.ps1` (used identically by
`.github/workflows/release.yml` and local development) downloads the
**exiftool-vendored.exe** npm package — a transparent, MIT-licensed,
open-source repackaging maintained by Matthew McEachen / PhotoStructure
(https://github.com/photostructure/exiftool-vendored.exe) that vendors
Phil Harvey's official Windows build **unmodified**. ExifTool itself,
inside that package, keeps its own license (above) — the npm package's
MIT license covers only the wrapper/vendoring project, not ExifTool.

**Official ExifTool 13.59 checksums**, as published by Phil Harvey at
https://exiftool.org/checksums.txt, for reference:

```
SHA2-256(exiftool-13.59_64.zip) = 44b512b25af500724ba579d0a53c8fc5851628b692dd5e5d94ae4a15c2cba9ec
SHA1(exiftool-13.59_64.zip)     = 022b902bb3d01b171d5108e2c4be59491eb64444
MD5 (exiftool-13.59_64.zip)     = df4fa1b736f0b8d39bd09b15fbdb865c
```

These are recorded here as upstream provenance only. They are **not**
checked against the npm-sourced file: the official distribution is a
`.zip` and the npm package is a `.tgz` with different internal packaging,
so their checksums are never expected to match, and comparing them would
be meaningless. What `fetch-exiftool.ps1` actually verifies is the
downloaded npm tarball's SHA-512 against the integrity hash npm's own
registry publishes for that exact package version — i.e. the download
wasn't corrupted or substituted in transit — plus a final sanity check
that the extracted `exiftool.exe -ver` reports exactly `13.59`.

Either way, the binary that ships is the same official, unmodified
Phil Harvey build; only the download channel used to obtain it differs
from what MetaClean's build pipeline would use if SourceForge permitted
automated access.

## Attribution requirement

If you redistribute MetaClean (including the bundled ExifTool binary),
please keep this notice and the license files inside `exiftool/` intact,
per the terms above.

---

MetaClean's own source code is licensed separately under the MIT License
— see `LICENSE` in the repository root.
