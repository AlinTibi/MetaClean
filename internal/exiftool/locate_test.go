package exiftool

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fixture"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLocatePackagedFromUnrelatedWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	exe := filepath.Join(root, "fresh extracted portable folder", "MetaClean.exe")
	bundled := filepath.Join(filepath.Dir(exe), "exiftool", exeName)
	writeExecutable(t, bundled)
	unrelated := t.TempDir()
	writeExecutable(t, filepath.Join(unrelated, "tools", "exiftool", exeName))
	t.Chdir(unrelated)
	got, err := locateFromExecutable(exe, "")
	if err != nil || got != bundled {
		t.Fatalf("got %q, %v; expected packaged engine %q", got, err, bundled)
	}
}

func TestLocateSourceBuildFromExecutableDirectory(t *testing.T) {
	root := t.TempDir()
	engine := filepath.Join(root, "tools", "exiftool", exeName)
	writeExecutable(t, engine)
	t.Chdir(t.TempDir())
	got, err := locateFromExecutable(filepath.Join(root, "build", "bin", "MetaClean.exe"), "")
	if err != nil || got != engine {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestLocateDoesNotExecuteWorkingDirectoryFallback(t *testing.T) {
	unrelated := t.TempDir()
	writeExecutable(t, filepath.Join(unrelated, "tools", "exiftool", exeName))
	t.Chdir(unrelated)
	_, err := locateFromExecutable(filepath.Join(t.TempDir(), "MetaClean.exe"), "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected missing packaged engine, got %v", err)
	}
}

func TestLocateExplicitOverride(t *testing.T) {
	engine := filepath.Join(t.TempDir(), "custom.exe")
	writeExecutable(t, engine)
	got, err := locateFromExecutable("unused.exe", engine)
	if err != nil || got != engine {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range []string{"relative.exe", filepath.Join(t.TempDir(), "missing.exe"), t.TempDir()} {
		if _, err := locateFromExecutable("unused.exe", bad); err == nil {
			t.Fatalf("accepted invalid override %q", bad)
		}
	}
}
