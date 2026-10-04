# Third-Party Notices

MetaClean bundles the official Windows distribution of **ExifTool** as its
metadata engine. ExifTool itself is not modified in any way.

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

MetaClean's release workflow downloads the official pre-built Windows
`exiftool.exe` (with its bundled Perl runtime) directly from its publisher
and packages it unmodified alongside the application; it is never
recompiled, patched, or redistributed as part of MetaClean's own source
code or license.

## Attribution requirement

If you redistribute MetaClean (including the bundled ExifTool binary),
please keep this notice and the license files inside `exiftool/` intact,
per the terms above.

---

MetaClean's own source code is licensed separately under the MIT License
— see `LICENSE` in the repository root.
