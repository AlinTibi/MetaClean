package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"MetaClean/internal/exiftool"
	"MetaClean/internal/model"
)

// Engine wraps an exiftool.Runner to scan files into model.FileEntry
// values.
type Engine struct {
	Runner *exiftool.Runner
}

func NewEngine(runner *exiftool.Runner) *Engine {
	return &Engine{Runner: runner}
}

// Scan inspects a single file and returns a fully populated FileEntry. It
// never returns an error for a file-level problem (unsupported format,
// missing file, exiftool failure); those are reported through the
// FileEntry's Status/Error fields instead, since a batch scan must not
// abort partway through because one file is bad.
func (e *Engine) Scan(ctx context.Context, id string, path string) *model.FileEntry {
	entry := &model.FileEntry{
		ID:   id,
		Path: path,
		Name: filepath.Base(path),
		Ext:  strings.ToLower(filepath.Ext(path)),
	}

	info, err := os.Stat(path)
	if err != nil {
		entry.Status = model.StatusError
		entry.Error = "cannot access file: " + err.Error()
		return entry
	}
	entry.Size = info.Size()

	if !IsSupported(path) {
		entry.Status = model.StatusUnsupported
		return entry
	}
	entry.Writable = IsWritable(path)

	raw, err := e.Runner.ReadJSON(ctx, path)
	if err != nil {
		entry.Status = model.StatusError
		entry.Error = describeReadError(err)
		return entry
	}

	parsed, err := exiftool.ParseJSON(raw)
	if err != nil {
		entry.Status = model.StatusError
		entry.Error = err.Error()
		return entry
	}
	if parsed.Error != "" {
		entry.Status = model.StatusError
		entry.Error = parsed.Error
		return entry
	}

	populateEntry(entry, parsed)
	return entry
}

func populateEntry(entry *model.FileEntry, parsed *exiftool.RawFile) {
	sensitiveSet := map[model.Category]bool{}

	for _, t := range parsed.Tags {
		category, sensitive, removable := Classify(t.Group, t.Tag)
		me := model.MetadataEntry{
			Group:     t.Group,
			Tag:       t.Tag,
			Label:     humanizeTag(t.Tag),
			Value:     t.Value,
			Category:  category,
			Sensitive: sensitive,
			Removable: removable,
		}
		entry.Metadata = append(entry.Metadata, me)
		if sensitive {
			sensitiveSet[category] = true
		}
	}

	for _, c := range model.AllCategories {
		if sensitiveSet[c] {
			entry.Sensitive = append(entry.Sensitive, c)
		}
	}

	switch {
	case len(entry.Sensitive) > 0:
		entry.Status = model.StatusSensitiveMetadata
	case len(entry.Metadata) > 0:
		entry.Status = model.StatusMetadataFound
	default:
		entry.Status = model.StatusClean
	}
}

// humanizeTag turns an ExifTool CamelCase tag name like "GPSLatitude" into
// a more readable label "GPS Latitude".
func humanizeTag(tag string) string {
	var b strings.Builder
	runes := []rune(tag)
	for i, r := range runes {
		if i > 0 {
			prevLower := isLower(runes[i-1])
			curUpper := isUpper(r)
			nextLower := i+1 < len(runes) && isLower(runes[i+1])
			if (curUpper && prevLower) || (curUpper && nextLower && isUpper(runes[i-1])) {
				b.WriteByte(' ')
			}
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func isLower(r rune) bool { return r >= 'a' && r <= 'z' }

func describeReadError(err error) string {
	var runErr *exiftool.RunError
	if errors.As(err, &runErr) {
		return runErr.Error()
	}
	return err.Error()
}
