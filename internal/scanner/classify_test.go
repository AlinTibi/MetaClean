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
