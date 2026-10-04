package exiftool

import "testing"

func TestParseJSONExtractsTags(t *testing.T) {
	data := []byte(`[{
		"SourceFile": "photo.jpg",
		"File:FileSize": "1234 bytes",
		"File:FileType": "JPEG",
		"ExifTool:ExifToolVersion": 13.59,
		"EXIF:Make": "Canon",
		"EXIF:GPSLatitude": 45.5,
		"XMP-dc:Creator": "Jane Doe"
	}]`)

	file, err := ParseJSON(data)
	if err != nil {
		t.Fatalf("ParseJSON failed: %v", err)
	}

	if file.SourceFile != "photo.jpg" {
		t.Errorf("expected SourceFile=photo.jpg, got %q", file.SourceFile)
	}

	// File:* and ExifTool:* groups are filesystem/tool info, not embedded
	// metadata, so they must be excluded from Tags.
	if len(file.Tags) != 3 {
		t.Fatalf("expected 3 tags (EXIF:Make, EXIF:GPSLatitude, XMP-dc:Creator), got %d: %+v", len(file.Tags), file.Tags)
	}

	byTag := map[string]RawTag{}
	for _, tag := range file.Tags {
		byTag[tag.Tag] = tag
	}

	if byTag["Make"].Group != "EXIF" || byTag["Make"].Value != "Canon" {
		t.Errorf("unexpected Make tag: %+v", byTag["Make"])
	}
	if byTag["Creator"].Group != "XMP-dc" || byTag["Creator"].Value != "Jane Doe" {
		t.Errorf("unexpected Creator tag: %+v", byTag["Creator"])
	}
	if byTag["GPSLatitude"].Value != "45.5" {
		t.Errorf("expected numeric GPSLatitude to stringify as 45.5, got %q", byTag["GPSLatitude"].Value)
	}
}

func TestParseJSONCapturesWarningAndError(t *testing.T) {
	data := []byte(`[{
		"SourceFile": "broken.jpg",
		"ExifTool:Error": "File format error"
	}]`)

	file, err := ParseJSON(data)
	if err != nil {
		t.Fatalf("ParseJSON failed: %v", err)
	}
	if file.Error != "File format error" {
		t.Errorf("expected Error to be captured, got %q", file.Error)
	}
}

func TestParseJSONEmptyArray(t *testing.T) {
	_, err := ParseJSON([]byte(`[]`))
	if err == nil {
		t.Fatal("expected an error for an empty results array")
	}
}

func TestParseJSONInvalid(t *testing.T) {
	_, err := ParseJSON([]byte(`not json`))
	if err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
}

func TestParseJSONArrayValue(t *testing.T) {
	data := []byte(`[{"SourceFile":"x.jpg","XMP-dc:Subject":["beach","sunset"]}]`)
	file, err := ParseJSON(data)
	if err != nil {
		t.Fatalf("ParseJSON failed: %v", err)
	}
	if len(file.Tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(file.Tags))
	}
	if file.Tags[0].Value != `["beach","sunset"]` {
		t.Errorf("expected array value to be JSON-rendered, got %q", file.Tags[0].Value)
	}
}
