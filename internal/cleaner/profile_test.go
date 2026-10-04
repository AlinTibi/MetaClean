package cleaner

import (
	"testing"

	"MetaClean/internal/model"
)

func contains(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

func TestBuildArgsRemoveAll(t *testing.T) {
	args := BuildArgs(model.CleanRequest{Profile: model.ProfileAll})
	if len(args) != 1 || args[0] != "-all=" {
		t.Errorf("expected exactly [-all=], got %v", args)
	}
}

func TestBuildArgsPrivacyPreservesTimestampsByDefault(t *testing.T) {
	args := BuildArgs(model.CleanRequest{Profile: model.ProfilePrivacy, PreserveTimestamps: true})

	if !contains(args, "-gps:all=") {
		t.Errorf("expected GPS removal in privacy profile, got %v", args)
	}
	if !contains(args, "-author=") {
		t.Errorf("expected author removal in privacy profile, got %v", args)
	}
	if contains(args, "-xmp:createdate=") {
		t.Errorf("expected timestamps preserved by default, got %v", args)
	}
}

func TestBuildArgsPrivacyDropsTimestampsWhenNotPreserved(t *testing.T) {
	args := BuildArgs(model.CleanRequest{Profile: model.ProfilePrivacy, PreserveTimestamps: false})
	if !contains(args, "-xmp:createdate=") {
		t.Errorf("expected timestamp removal when PreserveTimestamps=false, got %v", args)
	}
}

func TestBuildArgsCustomOnlySelectedCategories(t *testing.T) {
	args := BuildArgs(model.CleanRequest{
		Profile:          model.ProfileCustom,
		CustomCategories: []model.Category{model.CategoryGPS},
	})
	if !contains(args, "-gps:all=") {
		t.Errorf("expected GPS args, got %v", args)
	}
	if contains(args, "-author=") {
		t.Errorf("did not expect author args when only GPS selected, got %v", args)
	}
}

func TestBuildArgsCustomOtherCategoryIsNoOp(t *testing.T) {
	args := BuildArgs(model.CleanRequest{
		Profile:          model.ProfileCustom,
		CustomCategories: []model.Category{model.CategoryOther},
	})
	if len(args) != 0 {
		t.Errorf("expected no removal args for CategoryOther (no safe blanket mapping), got %v", args)
	}
}

func TestBuildArgsCustomDeduplicatesOverlappingArgs(t *testing.T) {
	// A hypothetical duplicate selection should never produce repeated
	// exiftool flags.
	args := BuildArgs(model.CleanRequest{
		Profile:          model.ProfileCustom,
		CustomCategories: []model.Category{model.CategoryGPS, model.CategoryGPS},
	})
	count := 0
	for _, a := range args {
		if a == "-gps:all=" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected -gps:all= exactly once, got %d occurrences in %v", count, args)
	}
}

func TestBuildArgsEmptyForNoSelection(t *testing.T) {
	args := BuildArgs(model.CleanRequest{Profile: model.ProfileCustom})
	if len(args) != 0 {
		t.Errorf("expected no args for an empty custom selection, got %v", args)
	}
}
