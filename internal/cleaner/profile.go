// Package cleaner builds ExifTool removal commands for MetaClean's
// cleaning profiles and safely executes them.
package cleaner

import (
	"MetaClean/internal/model"
	"MetaClean/internal/scanner"
)

// privacyCategories is the fixed set Privacy Clean targets: metadata that
// commonly identifies a person, device, or organization, while leaving
// the file otherwise fully usable.
var privacyCategories = []model.Category{
	model.CategoryGPS,
	model.CategoryAuthor,
	model.CategoryCamera,
	model.CategorySoftware,
	model.CategoryCompany,
	model.CategoryComments,
}

// BuildArgs returns the exiftool tag-removal arguments for req, in
// deterministic order. It is a pure function over its inputs (no I/O), so
// it can be unit tested exhaustively without exiftool or real files.
func BuildArgs(req model.CleanRequest) []string {
	if req.Profile == model.ProfileAll {
		return []string{"-all="}
	}

	var categories []model.Category
	switch req.Profile {
	case model.ProfilePrivacy:
		categories = append(categories, privacyCategories...)
		if !req.PreserveTimestamps {
			categories = append(categories, model.CategoryTimestamps)
		}
	case model.ProfileCustom:
		categories = append(categories, req.CustomCategories...)
	}

	return argsForCategories(categories)
}

func argsForCategories(categories []model.Category) []string {
	seen := map[string]bool{}
	var args []string
	for _, c := range categories {
		for _, arg := range scanner.CategoryRemovalArgs(c) {
			if !seen[arg] {
				seen[arg] = true
				args = append(args, arg)
			}
		}
	}
	return args
}
