# MetaClean v1.0.1

- Refreshes the inspector, queue counts/status and exported reports from the actual cleaned file. Cleaned-copy rows now inspect the output copy; originals remain untouched.
- Retains per-file failure details after a batch finishes. Interrupted in-place writes are re-inspected, and verification failures clear stale metadata rather than claiming success.
- Resolves bundled ExifTool relative to the executable, including shortcut launches and unrelated working directories. No engine is downloaded at runtime.
- Adds a distinct Windows icon with a privacy shield and eraser.
- Offers Microsoft's official WebView2 download page if the separate runtime is missing; nothing installs automatically.
- Keeps the PDF safety warning in the app and documentation: ExifTool PDF metadata writes may be incremental/reversible. A clean metadata scan does not prove secure PDF sanitization.

The portable release package includes ExifTool 13.59 and its support files; keep the extracted folder together. Microsoft Edge WebView2 Runtime is required separately.
