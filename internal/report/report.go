// Package report renders scan results as CSV metadata reports and JSON
// privacy reports for export.
package report

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"time"

	"MetaClean/internal/model"
)

// WriteMetadataCSV writes one row per discovered metadata tag, across all
// given files, to w.
func WriteMetadataCSV(entries []*model.FileEntry, w io.Writer) error {
	cw := csv.NewWriter(w)

	header := []string{"File", "Status", "Group", "Tag", "Category", "Sensitive", "Value", "Inspection error", "Cleaning error"}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, entry := range entries {
		if len(entry.Metadata) == 0 {
			if err := cw.Write([]string{entry.Path, string(entry.Status), "", "", "", "", "", entry.Error, entry.CleanError}); err != nil {
				return err
			}
			continue
		}
		for _, m := range entry.Metadata {
			row := []string{
				entry.Path,
				string(entry.Status),
				m.Group,
				m.Tag,
				string(m.Category),
				boolStr(m.Sensitive),
				m.Value,
				entry.Error,
				entry.CleanError,
			}
			if err := cw.Write(row); err != nil {
				return err
			}
		}
	}
	cw.Flush()
	return cw.Error()
}

// PrivacyReport is the JSON document produced by WritePrivacyReportJSON.
type PrivacyReport struct {
	GeneratedAt string              `json:"generatedAt"`
	FileCount   int                 `json:"fileCount"`
	Files       []PrivacyReportFile `json:"files"`
}

// PrivacyReportFile is one file's entry in a PrivacyReport.
type PrivacyReportFile struct {
	Error               string                `json:"error,omitempty"`
	CleanError          string                `json:"cleanError,omitempty"`
	Path                string                `json:"path"`
	Status              model.FileStatus      `json:"status"`
	MetadataCount       int                   `json:"metadataCount"`
	SensitiveCategories []model.Category      `json:"sensitiveCategories"`
	Metadata            []model.MetadataEntry `json:"metadata"`
}

// WritePrivacyReportJSON writes a structured privacy report covering
// entries to w, pretty-printed for readability.
func WritePrivacyReportJSON(entries []*model.FileEntry, w io.Writer) error {
	report := PrivacyReport{
		GeneratedAt: time.Now().Format(time.RFC3339),
		FileCount:   len(entries),
	}
	for _, entry := range entries {
		report.Files = append(report.Files, PrivacyReportFile{
			Error:               entry.Error,
			CleanError:          entry.CleanError,
			Path:                entry.Path,
			Status:              entry.Status,
			MetadataCount:       entry.MetadataCount(),
			SensitiveCategories: entry.Sensitive,
			Metadata:            entry.Metadata,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
