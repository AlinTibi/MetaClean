package cleaner

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"MetaClean/internal/exiftool"
	"MetaClean/internal/model"
	"MetaClean/internal/scanner"
)

// newTestEngine locates the real bundled exiftool and builds a cleaner
// Engine against it, skipping the test when exiftool isn't available
// (e.g. in CI, which does not download it — only the release workflow
// does). Locally, where tools/exiftool/ has been populated by
// scripts/fetch-exiftool.ps1, this runs for real against the actual
// binary, exercising the full read -> clean -> verify pipeline.
func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	runner, err := exiftool.NewRunner()
	if err != nil {
		t.Skipf("skipping: bundled exiftool not available (%v)", err)
	}
	scanEngine := scanner.NewEngine(runner)
	return NewEngine(scanEngine, runner)
}

// newTaggedJPEG writes a tiny valid JPEG, then uses exiftool to embed
// GPS, author and software metadata into it — small, legally-safe,
// generated-at-test-time fixture data (no binary blob committed).
func newTaggedJPEG(t *testing.T, runner *exiftool.Runner, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "sample.jpg")

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create fixture file: %v", err)
	}
	if err := jpeg.Encode(f, img, nil); err != nil {
		f.Close()
		t.Fatalf("failed to encode fixture JPEG: %v", err)
	}
	f.Close()

	ctx := context.Background()
	err = runner.WriteArgsInPlace(ctx, path, []string{
		"-overwrite_original",
		"-GPSLatitude=45.5017",
		"-GPSLatitudeRef=N",
		"-GPSLongitude=-73.5673",
		"-GPSLongitudeRef=W",
		"-Artist=Jane Doe",
		"-Software=TestCam 1.0",
	})
	if err != nil {
		t.Fatalf("failed to tag fixture JPEG: %v", err)
	}
	return path
}

func TestCleanToCopy_RemovesSensitiveMetadataAndPreservesOriginal(t *testing.T) {
	engine := newTestEngine(t)
	dir := t.TempDir()
	srcPath := newTaggedJPEG(t, engine.Runner, dir)

	beforeHash := mustReadBytes(t, srcPath)

	entry := engine.Scanner.Scan(context.Background(), "f1", srcPath)
	if entry.Status != model.StatusSensitiveMetadata {
		t.Fatalf("expected the tagged fixture to report sensitive metadata, got status=%v metadata=%+v", entry.Status, entry.Metadata)
	}
	beforeCount := entry.MetadataCount()
	if beforeCount == 0 {
		t.Fatalf("expected non-zero metadata before cleaning")
	}

	outputDir := filepath.Join(dir, "out")
	req := model.CleanRequest{
		FileIDs:            []string{"f1"},
		Profile:            model.ProfilePrivacy,
		OutputFolder:       outputDir,
		Suffix:             "_clean",
		PreserveTimestamps: true,
	}

	results := engine.CleanBatch(context.Background(), req, []*model.FileEntry{entry}, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	result := results[0]
	if !result.Success {
		t.Fatalf("expected cleaning to succeed, got error: %s", result.Error)
	}

	// The original file must be byte-for-byte untouched in default mode.
	afterHash := mustReadBytes(t, srcPath)
	if string(beforeHash) != string(afterHash) {
		t.Errorf("original file was modified, but default mode must never touch it")
	}

	if result.BeforeCount != beforeCount {
		t.Errorf("BeforeCount mismatch: result=%d scan=%d", result.BeforeCount, beforeCount)
	}
	if result.AfterCount >= result.BeforeCount {
		t.Errorf("expected AfterCount (%d) < BeforeCount (%d) after cleaning", result.AfterCount, result.BeforeCount)
	}

	// Re-scan the cleaned output directly and confirm the sensitive tags
	// are actually gone, not just reduced in count.
	cleaned := engine.Scanner.Scan(context.Background(), "verify", result.OutputPath)
	for _, m := range cleaned.Metadata {
		if m.Tag == "GPSLatitude" || m.Tag == "Artist" || m.Tag == "Software" {
			t.Errorf("expected %s to be removed from the cleaned copy, but it is still present (value=%q)", m.Tag, m.Value)
		}
	}
}

func TestCleanToCopy_ErrorOnMissingSourceFile(t *testing.T) {
	engine := newTestEngine(t)
	dir := t.TempDir()

	entry := &model.FileEntry{ID: "missing", Path: filepath.Join(dir, "does-not-exist.jpg")}
	req := model.CleanRequest{Profile: model.ProfileAll, OutputFolder: dir, Suffix: "_clean"}

	results := engine.CleanBatch(context.Background(), req, []*model.FileEntry{entry}, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Fatalf("expected cleaning a missing file to fail")
	}
	if results[0].Error == "" {
		t.Errorf("expected a non-empty, descriptive error message")
	}

	// Nothing should have been written to the output folder.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected no output file to be created on failure, found: %v", entries)
	}
}

func TestCleanInPlace_BacksUpThenReplacesOriginal(t *testing.T) {
	engine := newTestEngine(t)
	dir := t.TempDir()
	srcPath := newTaggedJPEG(t, engine.Runner, dir)
	originalBytes := mustReadBytes(t, srcPath)

	entry := engine.Scanner.Scan(context.Background(), "f1", srcPath)
	req := model.CleanRequest{
		Profile:         model.ProfilePrivacy,
		ReplaceOriginal: true,
		Suffix:          "_clean",
	}

	results := engine.CleanBatch(context.Background(), req, []*model.FileEntry{entry}, nil)
	result := results[0]
	if !result.Success {
		t.Fatalf("expected in-place cleaning to succeed, got error: %s", result.Error)
	}

	if result.BackupPath == "" {
		t.Fatal("expected a BackupPath to be reported")
	}
	if _, err := os.Stat(result.BackupPath); err != nil {
		t.Fatalf("expected backup file to exist at %s: %v", result.BackupPath, err)
	}
	backupBytes := mustReadBytes(t, result.BackupPath)
	if string(backupBytes) != string(originalBytes) {
		t.Errorf("backup content does not match the original file")
	}

	if result.OutputPath != srcPath {
		t.Errorf("expected OutputPath to be the original path in replace mode, got %s", result.OutputPath)
	}
	cleaned := engine.Scanner.Scan(context.Background(), "verify", srcPath)
	for _, m := range cleaned.Metadata {
		if m.Tag == "GPSLatitude" || m.Tag == "Artist" {
			t.Errorf("expected %s to be removed from the file in place, still present: %q", m.Tag, m.Value)
		}
	}
}

func TestCleanOne_RejectsNonWritableFormatWithClearError(t *testing.T) {
	engine := newTestEngine(t)
	dir := t.TempDir()

	// ExifTool can read DOCX metadata but cannot write/remove it; MetaClean
	// must refuse cleanly rather than let ExifTool fail deep inside with
	// "Can't write DOCX files", or silently do nothing.
	entry := &model.FileEntry{ID: "f1", Path: filepath.Join(dir, "doc.docx"), Ext: ".docx"}
	if err := os.WriteFile(entry.Path, []byte("not a real docx, just needs to exist"), 0o644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}

	req := model.CleanRequest{Profile: model.ProfileAll, OutputFolder: dir, Suffix: "_clean"}
	results := engine.CleanBatch(context.Background(), req, []*model.FileEntry{entry}, nil)

	if results[0].Success {
		t.Fatalf("expected cleaning a .docx to fail (read-only format)")
	}
	if !strings.Contains(results[0].Error, "cannot write") {
		t.Errorf("expected a clear 'cannot write' explanation, got: %q", results[0].Error)
	}

	entries, _ := os.ReadDir(dir)
	nonSource := 0
	for _, e := range entries {
		if e.Name() != "doc.docx" {
			nonSource++
		}
	}
	if nonSource != 0 {
		t.Errorf("expected no output file to be created for a rejected format, found extra entries: %v", entries)
	}
}

func mustReadBytes(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return data
}
