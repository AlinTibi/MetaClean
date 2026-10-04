package scanner

import (
	"testing"

	"MetaClean/internal/model"
)

func TestClassifyGPS(t *testing.T) {
	cat, sensitive, removable := Classify("GPS", "GPSLatitude")
	if cat != model.CategoryGPS || !sensitive || !removable {
		t.Errorf("GPSLatitude: got category=%v sensitive=%v removable=%v", cat, sensitive, removable)
	}
}

func TestClassifyAuthor(t *testing.T) {
	cases := []string{"Author", "Creator", "By-line", "Artist"}
	for _, tag := range cases {
		cat, sensitive, _ := Classify("IPTC", tag)
		if cat != model.CategoryAuthor || !sensitive {
			t.Errorf("%s: expected CategoryAuthor+sensitive, got %v/%v", tag, cat, sensitive)
		}
	}
}

func TestClassifyCamera(t *testing.T) {
	cases := []string{"Make", "Model", "SerialNumber", "LensModel"}
	for _, tag := range cases {
		cat, sensitive, _ := Classify("EXIF", tag)
		if cat != model.CategoryCamera || !sensitive {
			t.Errorf("%s: expected CategoryCamera+sensitive, got %v/%v", tag, cat, sensitive)
		}
	}
}

func TestClassifySoftware(t *testing.T) {
	cat, sensitive, _ := Classify("EXIF", "Software")
	if cat != model.CategorySoftware || !sensitive {
		t.Errorf("Software: got %v/%v", cat, sensitive)
	}
}

func TestClassifyCompany(t *testing.T) {
	cat, sensitive, _ := Classify("XMP-pdfx", "Company")
	if cat != model.CategoryCompany || !sensitive {
		t.Errorf("Company: got %v/%v", cat, sensitive)
	}
	cat, sensitive, _ = Classify("XMP-pdfx", "Manager")
	if cat != model.CategoryCompany || !sensitive {
		t.Errorf("Manager: got %v/%v", cat, sensitive)
	}
}

func TestClassifyComments(t *testing.T) {
	cases := []string{"UserComment", "Description", "Caption-Abstract"}
	for _, tag := range cases {
		cat, sensitive, _ := Classify("IPTC", tag)
		if cat != model.CategoryComments || !sensitive {
			t.Errorf("%s: expected CategoryComments+sensitive, got %v/%v", tag, cat, sensitive)
		}
	}
}

func TestClassifyTimestamps(t *testing.T) {
	cat, sensitive, _ := Classify("XMP-xmp", "CreateDate")
	if cat != model.CategoryTimestamps || !sensitive {
		t.Errorf("CreateDate: got %v/%v", cat, sensitive)
	}
}

// TestRemovableTagsAreActuallyTargetedByRemovalArgs is the core
// sensitive/removable correctness contract: for every tag Classify()
// marks removable=true, CategoryRemovalArgs() for that category must
// contain the exact exiftool flag that targets it. A tag flagged
// removable with no corresponding arg is a silent no-op bug — exactly
// what this test exists to catch.
func TestRemovableTagsAreActuallyTargetedByRemovalArgs(t *testing.T) {
	cases := []struct {
		group       string
		tag         string
		wantCat     model.Category
		wantArgFlag string // the exact entry expected in CategoryRemovalArgs
	}{
		{"IPTC", "Rights", model.CategoryAuthor, "-rights="},
		{"IPTC", "Credit", model.CategoryAuthor, "-credit="},
		{"XMP-dc", "Subject", model.CategoryComments, "-subject="},
		{"IPTC", "Keywords", model.CategoryComments, "-keywords="},
		{"XMP-x", "XMPToolkit", model.CategorySoftware, "-xmptoolkit="},
		{"IFD0", "XPComment", model.CategoryComments, "-xpcomment="},
		{"IFD0", "XPSubject", model.CategoryComments, "-xpsubject="},
		{"IFD0", "XPKeywords", model.CategoryComments, "-xpkeywords="},
		{"XMP-xmpMM", "HistorySoftwareAgent", model.CategorySoftware, "-historysoftwareagent="},
		{"PNG", "SoftwareVersion", model.CategorySoftware, "-softwareversion="},
		{"EXIF", "ApplicationVersion", model.CategorySoftware, "-applicationversion="},
		{"EXIF", "Make", model.CategoryCamera, "-make="},
		{"EXIF", "Software", model.CategorySoftware, "-software="},
		{"GPS", "GPSLatitude", model.CategoryGPS, "-gps:all="},
		{"XMP-xmp", "CreateDate", model.CategoryTimestamps, "-xmp:createdate="},
	}

	for _, c := range cases {
		gotCat, sensitive, removable := Classify(c.group, c.tag)
		if gotCat != c.wantCat {
			t.Errorf("%s:%s: expected category %v, got %v", c.group, c.tag, c.wantCat, gotCat)
			continue
		}
		if !sensitive || !removable {
			t.Errorf("%s:%s: expected sensitive=true removable=true, got sensitive=%v removable=%v", c.group, c.tag, sensitive, removable)
			continue
		}

		args := CategoryRemovalArgs(gotCat)
		found := false
		for _, a := range args {
			if a == c.wantArgFlag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s:%s classified removable, but %q is not in CategoryRemovalArgs(%v)=%v", c.group, c.tag, c.wantArgFlag, gotCat, args)
		}
	}
}

func TestClassifySubjectDistanceIsNotMisflaggedAsSensitive(t *testing.T) {
	// SubjectDistance/SubjectArea/SubjectLocation are EXIF focus-metering
	// tags, not the privacy-relevant XMP-dc:Subject (keywords/topic)
	// field. A naive "contains 'subject'" rule would wrongly flag them.
	for _, tag := range []string{"SubjectDistance", "SubjectArea", "SubjectLocation"} {
		cat, sensitive, _ := Classify("EXIF", tag)
		if sensitive {
			t.Errorf("%s: expected NOT sensitive (camera focus metadata, not a privacy field), got category=%v sensitive=true", tag, cat)
		}
	}
}

func TestClassifyApplicationRecordVersionIsNotMisflaggedAsSensitive(t *testing.T) {
	// IPTC's ApplicationRecordVersion is just a format version number
	// (e.g. "2"), not application/software identity. A naive "contains
	// 'application'" rule would wrongly flag it.
	cat, sensitive, _ := Classify("IPTC", "ApplicationRecordVersion")
	if sensitive {
		t.Errorf("ApplicationRecordVersion: expected NOT sensitive (a format version number, not software identity), got category=%v sensitive=true", cat)
	}
}

func TestClassifyNonXMPTimestampIsSensitiveButNotRemovable(t *testing.T) {
	// DateTimeOriginal (EXIF) is not reachable by the Timestamps
	// category's XMP-scoped removal args, so it must not claim
	// removable=true even though it's correctly flagged sensitive.
	cat, sensitive, removable := Classify("EXIF", "DateTimeOriginal")
	if cat != model.CategoryTimestamps {
		t.Errorf("DateTimeOriginal: expected CategoryTimestamps, got %v", cat)
	}
	if !sensitive {
		t.Errorf("DateTimeOriginal: expected sensitive=true")
	}
	if removable {
		t.Errorf("DateTimeOriginal: expected removable=false (not targeted by this category's XMP-only removal args)")
	}
}

func TestClassifyGPSBeforeTimestamp(t *testing.T) {
	// GPSDateStamp must classify as GPS, not Timestamps, even though it
	// contains "date" — this exercises rule ordering.
	cat, _, _ := Classify("GPS", "GPSDateStamp")
	if cat != model.CategoryGPS {
		t.Errorf("GPSDateStamp: expected CategoryGPS, got %v", cat)
	}
}

func TestClassifyUnknownTagIsOtherAndNotRemovable(t *testing.T) {
	cat, sensitive, removable := Classify("EXIF", "ColorSpace")
	if cat != model.CategoryOther {
		t.Errorf("ColorSpace: expected CategoryOther, got %v", cat)
	}
	if sensitive {
		t.Errorf("ColorSpace should not be flagged sensitive")
	}
	if removable {
		t.Errorf("CategoryOther tags should not be marked removable (no safe blanket removal mapping)")
	}
}

func TestIsSupported(t *testing.T) {
	supported := []string{"a.jpg", "A.JPG", "b.jpeg", "c.png", "d.tif", "e.tiff", "f.webp", "g.heic", "h.pdf", "i.docx", "j.xlsx", "k.pptx"}
	for _, name := range supported {
		if !IsSupported(name) {
			t.Errorf("expected %s to be supported", name)
		}
	}
	unsupported := []string{"a.txt", "b.exe", "c", "d.zip"}
	for _, name := range unsupported {
		if IsSupported(name) {
			t.Errorf("expected %s to be unsupported", name)
		}
	}
}

func TestCategoryRemovalArgsNonEmptyForKnownCategories(t *testing.T) {
	known := []model.Category{
		model.CategoryGPS, model.CategoryAuthor, model.CategoryCamera,
		model.CategorySoftware, model.CategoryCompany, model.CategoryComments,
		model.CategoryTimestamps,
	}
	for _, c := range known {
		if len(CategoryRemovalArgs(c)) == 0 {
			t.Errorf("expected removal args for category %v", c)
		}
	}
}

func TestIsWritable(t *testing.T) {
	writable := []string{"a.jpg", "b.png", "c.pdf", "d.tiff", "e.webp", "f.heic"}
	for _, name := range writable {
		if !IsWritable(name) {
			t.Errorf("expected %s to be writable", name)
		}
	}
	// ExifTool can read but not write Office Open XML formats.
	readOnly := []string{"a.docx", "b.xlsx", "c.pptx"}
	for _, name := range readOnly {
		if IsWritable(name) {
			t.Errorf("expected %s to NOT be writable (ExifTool cannot write Office Open XML)", name)
		}
		if !IsSupported(name) {
			t.Errorf("expected %s to still be supported for inspection", name)
		}
	}
}

func TestCategoryRemovalArgsEmptyForOther(t *testing.T) {
	if args := CategoryRemovalArgs(model.CategoryOther); len(args) != 0 {
		t.Errorf("expected no blanket removal args for CategoryOther, got %v", args)
	}
}
