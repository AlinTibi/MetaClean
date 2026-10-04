package cleaner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOutputPathBasic(t *testing.T) {
	dir := t.TempDir()
	got := OutputPath(`C:\photos\My Vacation (2024).jpg`, dir, "_clean")
	want := filepath.Join(dir, "My Vacation (2024)_clean.jpg")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestOutputPathUnicodeAndSpaces(t *testing.T) {
	dir := t.TempDir()
	got := OutputPath(`C:\docs\résumé café 日本語.docx`, dir, "_clean")
	want := filepath.Join(dir, "résumé café 日本語_clean.docx")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestOutputPathCollisionAvoidance(t *testing.T) {
	dir := t.TempDir()
	// Pre-create the name OutputPath would naturally pick.
	mustWriteFile(t, filepath.Join(dir, "photo_clean.jpg"))

	got := OutputPath(filepath.Join("input", "photo.jpg"), dir, "_clean")
	want := filepath.Join(dir, "photo_clean (2).jpg")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// With both the base and " (2)" variant taken, it must skip to (3).
	mustWriteFile(t, want)
	got2 := OutputPath(filepath.Join("input", "photo.jpg"), dir, "_clean")
	want2 := filepath.Join(dir, "photo_clean (3).jpg")
	if got2 != want2 {
		t.Errorf("got %q, want %q", got2, want2)
	}
}

func TestBackupPathCollisionAvoidance(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "report.pdf")
	mustWriteFile(t, original)
	mustWriteFile(t, filepath.Join(dir, "report.bak.pdf"))

	got := BackupPath(original)
	want := filepath.Join(dir, "report.bak(2).pdf")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to create fixture file %s: %v", path, err)
	}
}
