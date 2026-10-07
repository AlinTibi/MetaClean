package main

import (
	"MetaClean/internal/model"
	"MetaClean/internal/report"
	"bytes"
	"strings"
	"testing"
)

func TestCleanResultRefreshesQueueAndBothReports(t *testing.T) {
	app := NewApp()
	original := &model.FileEntry{ID: "f1", Path: "source.jpg", Status: model.StatusSensitiveMetadata,
		Metadata: []model.MetadataEntry{{Tag: "Artist", Value: "Fixture author", Sensitive: true}}, Sensitive: []model.Category{model.CategoryAuthor}}
	app.files["f1"] = original
	app.order = []string{"f1"}
	actual := &model.FileEntry{ID: "f1", Path: "output/source_clean.jpg", CleanedPath: "output/source_clean.jpg", Status: model.StatusClean, Metadata: []model.MetadataEntry{}}
	app.applyCleanResult(model.CleanFileResult{FileID: "f1", Success: true, Inspection: actual})
	if app.GetFiles()[0] != actual || original.MetadataCount() != 1 {
		t.Fatal("queue must replace the immutable scan, preserving the original snapshot")
	}
	var csv, json bytes.Buffer
	if err := report.WriteMetadataCSV(app.sortedFiles(), &csv); err != nil {
		t.Fatal(err)
	}
	if err := report.WritePrivacyReportJSON(app.sortedFiles(), &json); err != nil {
		t.Fatal(err)
	}
	for _, content := range []string{csv.String(), json.String()} {
		if strings.Contains(content, "Fixture author") || !strings.Contains(content, "source_clean.jpg") {
			t.Fatalf("stale report: %s", content)
		}
	}
}

func TestCleanFailureRefreshesErrorAndDoesNotReinsertRemovedFile(t *testing.T) {
	app := NewApp()
	app.files["f1"] = &model.FileEntry{ID: "f1", Metadata: []model.MetadataEntry{{Tag: "Artist", Value: "stale"}}}
	app.order = []string{"f1"}
	actual := &model.FileEntry{ID: "f1", Path: "output.jpg", Status: model.StatusError, Error: "cannot read", CleanError: "write ran; verification failed"}
	app.applyCleanResult(model.CleanFileResult{FileID: "f1", Inspection: actual})
	var buf bytes.Buffer
	if err := report.WritePrivacyReportJSON(app.sortedFiles(), &buf); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "stale") || !strings.Contains(buf.String(), "verification failed") {
		t.Fatal(buf.String())
	}
	app.ClearFiles()
	app.applyCleanResult(model.CleanFileResult{FileID: "f1", Inspection: actual})
	if len(app.GetFiles()) != 0 {
		t.Fatal("late progress restored a removed entry")
	}
}

func TestUnchangedFailureKeepsOriginalInspection(t *testing.T) {
	app := NewApp()
	original := &model.FileEntry{ID: "f1", Path: "read-only.docx"}
	app.files["f1"] = original
	app.applyCleanResult(model.CleanFileResult{FileID: "f1", Error: "format is read-only"})
	if app.files["f1"].Path != original.Path || app.files["f1"].CleanError != "format is read-only" || original.CleanError != "" {
		t.Fatal("unchanged file lost its scan or failure details")
	}
}
