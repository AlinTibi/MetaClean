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

	// removableIf, when set, additionally gates whether a match is
	// reported as removable: a tag can be real and sensitive without
	// this category's removableArgs actually being able to strip it
	// (e.g. a non-XMP timestamp tag). When nil, any match is removable.
	removableIf func(group string) bool
}

func isXMPGroup(group string) bool {
	return strings.HasPrefix(group, "XMP")
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
		removableArgs: []string{
			"-author=", "-creator=", "-by-line=", "-artist=", "-xpauthor=", "-ownername=",
			"-lastmodifiedby=", "-rights=", "-credit=",
		},
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
		category: model.CategorySoftware,
		// "applicationversion" is an exact match (not a substring)
		// deliberately: IPTC's ApplicationRecordVersion is just a format
		// version number (e.g. "2"), not application/software identity,
		// and would otherwise be mis-flagged as sensitive by a broad
		// "application" substring rule.
		exact:       map[string]bool{"applicationversion": true},
		containsAny: []string{"software", "creatortool", "producer", "xmptoolkit"},
		removableArgs: []string{
			"-software=", "-softwareversion=", "-historysoftwareagent=", "-creatortool=",
			"-producer=", "-applicationversion=", "-xmptoolkit=",
		},
	},
	{
		category: model.CategoryCompany,
		exact: map[string]bool{
			"company": true, "manager": true,
		},
		removableArgs: []string{"-company=", "-manager="},
	},
	{
		category: model.CategoryComments,
		// "subject"/"keywords" are exact matches (not substrings)
		// deliberately: EXIF camera tags like SubjectDistance/
		// SubjectArea/SubjectLocation contain "subject" but describe
		// focus metering, not a privacy-relevant topic/keyword field,
		// and would otherwise be mis-flagged as sensitive. The Windows
		// Explorer "XP" variants (XPSubject, XPKeywords) are listed
		// separately since they're distinct writable tags, not aliases.
		exact:       map[string]bool{"subject": true, "keywords": true, "xpsubject": true, "xpkeywords": true},
		containsAny: []string{"comment", "description", "caption"},
		removableArgs: []string{
			"-comment=", "-usercomment=", "-xpcomment=", "-description=", "-imagedescription=",
			"-caption-abstract=", "-xmp:description=", "-subject=", "-keywords=", "-xpsubject=", "-xpkeywords=",
		},
	},
	{
		category:    model.CategoryTimestamps,
		containsAny: []string{"date", "time"},
		removableArgs: []string{
			"-xmp:createdate=", "-xmp:modifydate=", "-xmp:metadatadate=",
		},
		// The removal args above only reach XMP-namespaced date/time
		// tags (a deliberate, documented scope: broader date removal is
		// available via "Remove All Metadata"). A non-XMP date/time tag
		// (e.g. EXIF DateTimeOriginal) is still flagged sensitive so the
		// user can see it, but must not be reported as removable by this
		// category, since nothing in removableArgs actually targets it.
		removableIf: isXMPGroup,
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
		matched := rule.exact[lower]
		if !matched {
			for _, substr := range rule.containsAny {
				if strings.Contains(lower, substr) {
					matched = true
					break
				}
			}
		}
		if !matched {
			continue
		}
		if rule.removableIf != nil && !rule.removableIf(group) {
			return rule.category, true, false
		}
		return rule.category, true, true
	}

	if group == "Composite" {
		// Derived/computed values (e.g. GPSPosition): informative only,
		// ExifTool cannot remove them directly (removing the underlying
		// source tags removes these too).
		return model.CategoryOther, false, false
	}

	return model.CategoryOther, false, false
}
