package cleaner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// OutputPath computes where a cleaned copy of inputPath should be written:
// <outputFolder>/<base><suffix><ext>, preserving the original filename
// (including spaces/Unicode) other than the inserted suffix. If that path
// already exists, it appends " (2)", " (3)", ... before the extension
// until it finds a free name, so a batch never silently overwrites a
// previous run's output.
func OutputPath(inputPath, outputFolder, suffix string) string {
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)
	candidate := filepath.Join(outputFolder, base+suffix+ext)

	if !pathExists(candidate) {
		return candidate
	}
	for i := 2; ; i++ {
		alt := filepath.Join(outputFolder, fmt.Sprintf("%s%s (%d)%s", base, suffix, i, ext))
		if !pathExists(alt) {
			return alt
		}
	}
}

// BackupPath computes the backup filename used by "Replace original"
// mode: <base>.bak<ext> next to the original, with the same collision-safe
// numbering as OutputPath.
func BackupPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)
	candidate := filepath.Join(dir, base+".bak"+ext)

	if !pathExists(candidate) {
		return candidate
	}
	for i := 2; ; i++ {
		alt := filepath.Join(dir, fmt.Sprintf("%s.bak(%d)%s", base, i, ext))
		if !pathExists(alt) {
			return alt
		}
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
