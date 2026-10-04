package exiftool

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// RawTag is one "Group:Tag" => value pair exactly as exiftool -json -G1
// reports it, before any privacy classification is applied.
type RawTag struct {
	Group string
	Tag   string
	Value string
}

// RawFile is the parsed form of one element of exiftool's -json output
// array.
type RawFile struct {
	SourceFile string
	Tags       []RawTag
	// Warning/Error hold any ExifTool:Warning / ExifTool:Error messages
	// exiftool embedded in the output (exiftool reports many recoverable
	// problems this way rather than failing the process).
	Warning string
	Error   string
}

// ParseJSON parses the raw stdout of `exiftool -json -G1 ...` for a single
// file (a one-element JSON array) into a RawFile. It is pure and does not
// execute exiftool, so it is fully unit-testable against fixture strings.
func ParseJSON(data []byte) (*RawFile, error) {
	var records []map[string]interface{}
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("parsing exiftool JSON output: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("exiftool returned no results")
	}

	record := records[0]
	file := &RawFile{}

	keys := make([]string, 0, len(record))
	for k := range record {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := record[key]
		group, tag := splitGroupTag(key)

		switch {
		case key == "SourceFile":
			file.SourceFile = stringifyValue(value)
			continue
		case group == "ExifTool" && tag == "Warning":
			file.Warning = stringifyValue(value)
			continue
		case group == "ExifTool" && tag == "Error":
			file.Error = stringifyValue(value)
			continue
		case group == "ExifTool" || group == "File" || group == "System" || group == "ZIP" || group == "":
			// Internal tool info, filesystem attributes (System/File) and
			// ZIP container structure (Office/OOXML files are ZIP
			// archives) are not embedded file metadata; skip them from
			// the inspector.
			continue
		}

		file.Tags = append(file.Tags, RawTag{
			Group: group,
			Tag:   tag,
			Value: stringifyValue(value),
		})
	}

	return file, nil
}

// splitGroupTag splits a "-G1" JSON key like "EXIF:Make" or
// "XMP-dc:Creator" into its group and tag parts. Keys with no colon (like
// "SourceFile") return an empty group.
func splitGroupTag(key string) (group, tag string) {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == ':' {
			return key[:i], key[i+1:]
		}
	}
	return "", key
}

// stringifyValue renders an arbitrary JSON value (string, number, bool,
// array, object) as a display string.
func stringifyValue(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case nil:
		return ""
	case []interface{}:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		return string(b)
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		return string(b)
	}
}
