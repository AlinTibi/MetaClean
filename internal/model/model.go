// Package model holds the data types shared across MetaClean's backend
// packages (exiftool, scanner, cleaner, report) and exposed to the frontend
// through app.go.
package model

// FileStatus summarizes a file's metadata situation at a glance.
type FileStatus string

const (
	StatusPending            FileStatus = "pending"
	StatusClean               FileStatus = "clean"
	StatusMetadataFound       FileStatus = "metadata_found"
	StatusSensitiveMetadata   FileStatus = "sensitive_metadata_found"
	StatusUnsupported         FileStatus = "unsupported"
	StatusError               FileStatus = "error"
)

// Category groups metadata tags for display and for selecting what a
// cleaning profile should remove.
type Category string

const (
	CategoryGPS         Category = "gps"
	CategoryAuthor      Category = "author"
	CategoryCamera      Category = "camera"
	CategorySoftware    Category = "software"
	CategoryCompany     Category = "company"
	CategoryComments    Category = "comments"
	CategoryTimestamps  Category = "timestamps"
	CategoryOther       Category = "other"
)

// AllCategories lists every category in the order they should be presented
// (and the order Custom-mode checkboxes appear in).
var AllCategories = []Category{
	CategoryGPS,
	CategoryAuthor,
	CategoryCamera,
	CategorySoftware,
	CategoryCompany,
	CategoryComments,
	CategoryTimestamps,
	CategoryOther,
}

// MetadataEntry is a single tag discovered in a file.
type MetadataEntry struct {
	Group     string   `json:"group"`     // ExifTool group, e.g. "EXIF", "XMP-dc", "IPTC"
	Tag       string   `json:"tag"`       // ExifTool tag name, e.g. "GPSLatitude"
	Label     string   `json:"label"`     // Human-readable label
	Value     string   `json:"value"`     // String representation of the value
	Category  Category `json:"category"`
	Sensitive bool     `json:"sensitive"`
	Removable bool     `json:"removable"` // false for structural tags ExifTool can't/won't delete
}

// FileEntry is one row in the queue: a file plus everything MetaClean knows
// about its metadata.
type FileEntry struct {
	ID         string          `json:"id"`
	Path       string          `json:"path"`
	Name       string          `json:"name"`
	Ext        string          `json:"ext"`
	Size       int64           `json:"size"`
	Status     FileStatus      `json:"status"`
	Error      string          `json:"error,omitempty"`
	Writable   bool            `json:"writable"`
	Metadata   []MetadataEntry `json:"metadata"`
	Sensitive  []Category      `json:"sensitiveCategories"`
	CleanedPath string         `json:"cleanedPath,omitempty"`
}

// MetadataCount returns how many metadata entries were found.
func (f *FileEntry) MetadataCount() int {
	return len(f.Metadata)
}

// Profile selects which metadata a cleaning run should target.
type Profile string

const (
	ProfilePrivacy Profile = "privacy"
	ProfileAll     Profile = "all"
	ProfileCustom  Profile = "custom"
)

// CleanRequest describes one batch cleaning operation submitted by the
// frontend.
type CleanRequest struct {
	FileIDs          []string   `json:"fileIds"`
	Profile          Profile    `json:"profile"`
	CustomCategories []Category `json:"customCategories,omitempty"`
	OutputFolder     string     `json:"outputFolder"`
	Suffix           string     `json:"suffix"`
	ReplaceOriginal  bool       `json:"replaceOriginal"`
	PreserveTimestamps bool     `json:"preserveTimestamps"`
}

// CleanFileResult is the outcome of cleaning a single file.
type CleanFileResult struct {
	FileID        string          `json:"fileId"`
	Success       bool            `json:"success"`
	Error         string          `json:"error,omitempty"`
	OutputPath    string          `json:"outputPath,omitempty"`
	BackupPath    string          `json:"backupPath,omitempty"`
	BeforeCount   int             `json:"beforeCount"`
	AfterCount    int             `json:"afterCount"`
	Remaining     []MetadataEntry `json:"remaining,omitempty"`
}

// Settings persists user preferences between runs.
type Settings struct {
	DefaultProfile     Profile `json:"defaultProfile"`
	OutputFolder       string  `json:"outputFolder"`
	Suffix             string  `json:"suffix"`
	ReplaceOriginal    bool    `json:"replaceOriginal"`
	PreserveTimestamps bool    `json:"preserveTimestamps"`
	LastOutputFolder   string  `json:"lastOutputFolder"`
	Theme              string  `json:"theme"`
}

// DefaultSettings returns the built-in defaults used the first time
// MetaClean runs (before any settings file exists).
func DefaultSettings() Settings {
	return Settings{
		DefaultProfile:     ProfilePrivacy,
		Suffix:             "_clean",
		ReplaceOriginal:    false,
		PreserveTimestamps: true,
		Theme:              "dark",
	}
}
