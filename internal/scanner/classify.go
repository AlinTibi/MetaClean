// Package scanner inspects files with the bundled ExifTool engine and
// classifies their metadata for privacy review.
package scanner

import (
	"strings"

	"MetaClean/internal/model"
)

// SupportedExts lists the file extensions MetaClean knows how to inspect,
// per the formats ExifTool's bundled build reliably supports for this
// tool's v1 scope.
var SupportedExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".tif":  true,
	".tiff": true,
	".webp": true,
	".heic": true,
	".heif": true,
	".pdf":  true,
	".docx": true,
	".xlsx": true,
	".pptx": true,
}

// IsSupported reports whether path's extension is one MetaClean can
// inspect.
func IsSupported(path string) bool {
	return SupportedExts[strings.ToLower(extOf(path))]
}

// WritableExts lists the extensions the bundled ExifTool build can
// actually write to (`exiftool -listwf`). Notably, Office Open XML
// formats (.docx/.xlsx/.pptx) are NOT in this list: ExifTool can read
// their metadata but cannot remove it. MetaClean inspects these formats
// but refuses to attempt cleaning them, rather than claiming success (or
// producing a confusing "Can't write DOCX files" error from deep inside
// ExifTool) for something it cannot actually do.
var WritableExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".tif":  true,
	".tiff": true,
	".webp": true,
	".heic": true,
	".heif": true,
	".pdf":  true,
}

// IsWritable reports whether MetaClean can clean (not just inspect)
// path's format.
func IsWritable(path string) bool {
	return WritableExts[strings.ToLower(extOf(path))]
}

func extOf(path string) string {
	i := strings.LastIndexByte(path, '.')
	if i < 0 {
		return ""
	}
	return path[i:]
}

// categoryRule matches a tag name against a category using simple,
// auditable substring/exact rules — easy to verify and to extend.
type categoryRule struct {
	category      model.Category
	exact         map[string]bool
	containsAny   []string
	removableArgs []string // the exiftool args this category maps to, if any
}

// classificationRules is evaluated in order; the first match wins. GPS is
// checked before generic "contains" rules since GPS tag names are
// otherwise easy to mis-bucket (e.g. "GPSDateStamp" should read as GPS,
// not Timestamps).
var classificationRules = []categoryRule{
	{
		category:      model.CategoryGPS,
		containsAny:   []string{"gps"},
		removableArgs: []string{"-gps:all="},
	},
	{
		category: model.CategoryAuthor,
		exact: map[string]bool{
			"author": true, "creator": true, "by-line": true, "artist": true,
			"xpauthor": true, "ownername": true, "rights": true, "credit": true,
			"lastmodifiedby": true,
		},
		removableArgs: []string{"-author=", "-creator=", "-by-line=", "-artist=", "-xpauthor=", "-ownername=", "-lastmodifiedby="},
	},
	{
		category: model.CategoryCamera,
		exact: map[string]bool{
			"make": true, "model": true, "serialnumber": true, "lensmodel": true,
			"lensserialnumber": true, "lensmake": true, "bodyserialnumber": true,
			"internalserialnumber": true, "cameraserialnumber": true,
		},
		removableArgs: []string{
			"-make=", "-model=", "-serialnumber=", "-lensmodel=", "-lensserialnumber=",
			"-lensmake=", "-bodyserialnumber=", "-internalserialnumber=", "-cameraserialnumber=",
		},
	},
	{
		category:      model.CategorySoftware,
		containsAny:   []string{"software", "creatortool", "application", "producer", "toolkit"},
		removableArgs: []string{"-software=", "-creatortool=", "-producer=", "-applicationversion=", "-toolkit="},
	},
	{
		category: model.CategoryCompany,
		exact: map[string]bool{
			"company": true, "manager": true,
		},
		removableArgs: []string{"-company=", "-manager="},
	},
	{
		category:      model.CategoryComments,
		containsAny:   []string{"comment", "description", "caption", "subject", "keyword"},
		removableArgs: []string{"-comment=", "-usercomment=", "-description=", "-imagedescription=", "-caption-abstract=", "-xmp:description="},
	},
	{
		category:      model.CategoryTimestamps,
		containsAny:   []string{"date", "time"},
		removableArgs: []string{"-xmp:createdate=", "-xmp:modifydate=", "-xmp:metadatadate="},
	},
}

// CategoryRemovalArgs returns the exiftool tag-deletion arguments used to
// strip a given category, for Custom and Privacy cleaning profiles. It is
// the single source of truth shared by classification and cleaning, so
// "what we detect as sensitive" and "what Custom mode can remove" can
// never drift apart.
func CategoryRemovalArgs(category model.Category) []string {
	for _, rule := range classificationRules {
		if rule.category == category {
			return rule.removableArgs
		}
	}
	return nil
}

// Classify assigns a Category and sensitivity/removability flags to a raw
// (group, tag) pair discovered by ExifTool. It never executes exiftool;
// it is a pure function so classification rules can be unit tested
// directly against representative tag names.
func Classify(group, tag string) (category model.Category, sensitive bool, removable bool) {
	lower := strings.ToLower(tag)

	for _, rule := range classificationRules {
		if rule.exact[lower] {
			return rule.category, true, true
		}
		for _, substr := range rule.containsAny {
			if strings.Contains(lower, substr) {
				return rule.category, true, true
			}
		}
	}

	if group == "Composite" {
		// Derived/computed values (e.g. GPSPosition): informative only,
		// ExifTool cannot remove them directly (removing the underlying
		// source tags removes these too).
		return model.CategoryOther, false, false
	}

	return model.CategoryOther, false, false
}
