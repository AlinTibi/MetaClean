package report

import (
	"MetaClean/internal/model"
	"bytes"
	"errors"
	"strings"
	"testing"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk write failed") }

func TestCSVReportsFlushFailure(t *testing.T) {
	if err := WriteMetadataCSV([]*model.FileEntry{{Path: "test.jpg", Status: model.StatusClean}}, failingWriter{}); err == nil {
		t.Fatal("CSV export silently accepted a failed disk write")
	}
}

func TestReportsPreservePartialCleaningAndInspectionErrors(t *testing.T) {
	entries := []*model.FileEntry{{Path: "output.jpg", Status: model.StatusError, Error: "cannot inspect", CleanError: "partial write failed"}}
	var csv, json bytes.Buffer
	if err := WriteMetadataCSV(entries, &csv); err != nil {
		t.Fatal(err)
	}
	if err := WritePrivacyReportJSON(entries, &json); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{csv.String(), json.String()} {
		if !strings.Contains(text, "cannot inspect") || !strings.Contains(text, "partial write failed") {
			t.Fatal("report omitted failure details", text)
		}
	}
}
