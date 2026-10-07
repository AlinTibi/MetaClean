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
//  3. "tools/exiftool/exiftool.exe" found above the executable directory,
//     for builds in a source checkout. Go tests/go run in a temporary
//     build directory use the explicit override. The CWD is never searched.
func Locate() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return locateFromExecutable(exePath, os.Getenv("METACLEAN_EXIFTOOL_PATH"))
}

func locateFromExecutable(exePath, override string) (string, error) {
	if override != "" {
		if !filepath.IsAbs(override) {
			return "", errors.New("METACLEAN_EXIFTOOL_PATH must be an absolute path")
		}
		if fileExists(override) {
			return override, nil
		}
		return "", errors.New("METACLEAN_EXIFTOOL_PATH is set but does not point to a file: " + override)
	}

	exeDir := filepath.Dir(exePath)
	candidate := filepath.Join(exeDir, "exiftool", exeName)
	if fileExists(candidate) {
		return candidate, nil
	}
	if found, ok := searchUpward(exeDir, filepath.Join("tools", "exiftool", exeName)); ok {
		return found, nil
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
