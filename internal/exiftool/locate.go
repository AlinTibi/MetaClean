package exiftool

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrNotFound is returned when no exiftool executable can be located.
var ErrNotFound = errors.New("exiftool executable not found; expected it bundled next to the application in an \"exiftool\" folder")

const exeName = "exiftool.exe"

// Locate finds the bundled ExifTool executable. It never downloads
// anything and never searches the system PATH, so MetaClean only ever
// runs the exact binary it ships with.
//
// Resolution order:
//  1. METACLEAN_EXIFTOOL_PATH environment variable, if set (explicit
//     developer/operator override).
//  2. "exiftool/exiftool.exe" next to the running executable — the layout
//     used by packaged builds.
//  3. "tools/exiftool/exiftool.exe" found by walking up from the current
//     working directory — a developer convenience so `wails dev` and
//     `go run` work from a source checkout without a packaged build,
//     using the same local-only cache `scripts/fetch-exiftool.ps1`
//     populates.
func Locate() (string, error) {
	if override := os.Getenv("METACLEAN_EXIFTOOL_PATH"); override != "" {
		if fileExists(override) {
			return override, nil
		}
		return "", errors.New("METACLEAN_EXIFTOOL_PATH is set but does not point to a file: " + override)
	}

	if exePath, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exePath), "exiftool", exeName)
		if fileExists(candidate) {
			return candidate, nil
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		if found, ok := searchUpward(cwd, filepath.Join("tools", "exiftool", exeName)); ok {
			return found, nil
		}
	}

	return "", ErrNotFound
}

func searchUpward(start string, relative string) (string, bool) {
	dir := start
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, relative)
		if fileExists(candidate) {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
