package cleaner

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"MetaClean/internal/exiftool"
	"MetaClean/internal/model"
)

// Simulate an engine that changes an original and then exits with an error.
// The real ExifTool still performs the subsequent inspection.
func TestMain(m *testing.M) {
	if replacement := os.Getenv("METACLEAN_TEST_PARTIAL_WRITE"); replacement != "" {
		data, err := os.ReadFile(replacement)
		if err == nil {
			err = os.WriteFile(os.Args[len(os.Args)-1], data, 0644)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Fprintln(os.Stderr, "fixture write interrupted after changing file")
		}
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestVerifiedInspectionMatchesActualCopyAndOriginal(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(fmt.Sprintf("replace=%t", replace), func(t *testing.T) {
			engine := newTestEngine(t)
			dir := t.TempDir()
			path := newTaggedJPEG(t, engine.Runner, dir)
			before := engine.Scanner.Scan(context.Background(), "f1", path)
			result := engine.CleanBatch(context.Background(), model.CleanRequest{Profile: model.ProfileAll, ReplaceOriginal: replace, OutputFolder: filepath.Join(dir, "out"), Suffix: "_clean"}, []*model.FileEntry{before}, nil)[0]
			if !result.Success || result.Inspection == nil {
				t.Fatal(result)
			}
			actual := engine.Scanner.Scan(context.Background(), "f1", result.OutputPath)
			if result.Inspection.Path != actual.Path || result.Inspection.ID != "f1" || result.AfterCount != actual.MetadataCount() || result.Inspection.Status != actual.Status {
				t.Fatal("result does not describe actual output", result)
			}
			for _, tag := range result.Inspection.Metadata {
				if tag.Tag == "Artist" || tag.Tag == "GPSLatitude" {
					t.Fatal("stale sensitive tag", tag)
				}
			}
		})
	}
}

func TestPartialInPlaceFailureRefreshesRealFileAndPreservesFailure(t *testing.T) {
	engine := newTestEngine(t)
	dir := t.TempDir()
	path := newTaggedJPEG(t, engine.Runner, dir)
	original := mustReadBytes(t, path)
	before := engine.Scanner.Scan(context.Background(), "f1", path)
	clean := filepath.Join(dir, "replacement.jpg")
	if err := engine.Runner.WriteArgs(context.Background(), path, clean, []string{"-all="}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("METACLEAN_TEST_PARTIAL_WRITE", clean)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	engine.Runner = &exiftool.Runner{Path: executable}
	result := engine.CleanBatch(context.Background(), model.CleanRequest{Profile: model.ProfileAll, ReplaceOriginal: true}, []*model.FileEntry{before}, nil)[0]
	if result.Success || result.Inspection == nil || !strings.Contains(result.Error, "in-place cleaning failed") || result.Inspection.CleanError != result.Error {
		t.Fatal(result)
	}
	if result.AfterCount >= before.MetadataCount() {
		t.Fatal("failed write did not refresh changed file")
	}
	if string(mustReadBytes(t, result.BackupPath)) != string(original) {
		t.Fatal("original backup changed")
	}
}

func TestPNGCleaningRefreshesSensitiveMetadata(t *testing.T) {
	engine := newTestEngine(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(file, image.NewRGBA(image.Rect(0, 0, 8, 8)))
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err = engine.Runner.WriteArgsInPlace(context.Background(), path, []string{"-overwrite_original", "-XMP-dc:Creator=FixturePNGAuthor"}); err != nil {
		t.Fatal(err)
	}
	before := engine.Scanner.Scan(context.Background(), "png", path)
	if before.Status != model.StatusSensitiveMetadata {
		t.Fatal("fixture missing metadata", before)
	}
	result := engine.CleanBatch(context.Background(), model.CleanRequest{Profile: model.ProfilePrivacy, OutputFolder: filepath.Join(dir, "out"), Suffix: "_clean"}, []*model.FileEntry{before}, nil)[0]
	if !result.Success || result.Inspection == nil {
		t.Fatal(result)
	}
	for _, tag := range result.Inspection.Metadata {
		if strings.Contains(tag.Value, "FixturePNGAuthor") {
			t.Fatal("PNG author remains", tag)
		}
	}
}

func TestPDFScanCanBeCleanWhileOriginalMetadataRemainsRecoverable(t *testing.T) {
	engine := newTestEngine(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	var pdf strings.Builder
	pdf.WriteString("%PDF-1.4\n")
	objects := []string{"<< /Type /Catalog /Pages 2 0 R >>", "<< /Type /Pages /Kids [3 0 R] /Count 1 >>", "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>", "<< /Author (RecoverableFixtureAuthor) /Title (Metadata test) >>"}
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, pdf.Len())
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 5\n0000000000 65535 f \n")
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size 5 /Root 1 0 R /Info 4 0 R >>\nstartxref\n%d\n%%%%EOF\n", xref)
	if err := os.WriteFile(path, []byte(pdf.String()), 0644); err != nil {
		t.Fatal(err)
	}
	before := engine.Scanner.Scan(context.Background(), "pdf", path)
	if before.Status != model.StatusSensitiveMetadata {
		t.Fatal("PDF author not inspected", before)
	}
	result := engine.CleanBatch(context.Background(), model.CleanRequest{Profile: model.ProfileAll, OutputFolder: filepath.Join(dir, "out"), Suffix: "_clean"}, []*model.FileEntry{before}, nil)[0]
	if !result.Success || result.Inspection == nil {
		t.Fatal(result)
	}
	for _, tag := range result.Inspection.Metadata {
		if strings.Contains(tag.Value, "RecoverableFixtureAuthor") {
			t.Fatal("PDF active author remains", tag)
		}
	}
	if !strings.Contains(string(mustReadBytes(t, result.OutputPath)), "RecoverableFixtureAuthor") {
		t.Fatal("fixture should demonstrate incremental PDF metadata persistence")
	}
}
