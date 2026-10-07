package cleaner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"MetaClean/internal/exiftool"
	"MetaClean/internal/model"
	"MetaClean/internal/scanner"
)

// Engine cleans files using a scanner.Engine (for the before/after
// verification pass) backed by the same exiftool.Runner.
type Engine struct {
	Scanner *scanner.Engine
	Runner  *exiftool.Runner
}

func NewEngine(scannerEngine *scanner.Engine, runner *exiftool.Runner) *Engine {
	return &Engine{Scanner: scannerEngine, Runner: runner}
}

// ProgressFunc is called after each file in a batch finishes (success or
// failure), so the caller can stream per-file progress to the UI.
type ProgressFunc func(index int, total int, result model.CleanFileResult)

// CleanBatch cleans every entry in entries according to req, in order,
// honoring ctx cancellation between files (an in-flight exiftool call is
// also killed immediately on cancellation, since it runs via
// exec.CommandContext). It never stops the whole batch because one file
// failed; failures are reported per-file.
func (e *Engine) CleanBatch(ctx context.Context, req model.CleanRequest, entries []*model.FileEntry, onProgress ProgressFunc) []model.CleanFileResult {
	results := make([]model.CleanFileResult, 0, len(entries))

	if req.OutputFolder != "" {
		_ = os.MkdirAll(req.OutputFolder, 0o755)
	}

	tagArgs := BuildArgs(req)

	for i, entry := range entries {
		if ctx.Err() != nil {
			result := model.CleanFileResult{FileID: entry.ID, Success: false, Error: "cancelled"}
			results = append(results, result)
			if onProgress != nil {
				onProgress(i, len(entries), result)
			}
			continue
		}

		result := e.cleanOne(ctx, req, entry, tagArgs)
		results = append(results, result)
		if onProgress != nil {
			onProgress(i, len(entries), result)
		}
	}

	return results
}

func (e *Engine) cleanOne(ctx context.Context, req model.CleanRequest, entry *model.FileEntry, tagArgs []string) model.CleanFileResult {
	result := model.CleanFileResult{FileID: entry.ID, BeforeCount: entry.MetadataCount()}

	if len(tagArgs) == 0 {
		result.Error = "no metadata categories selected to remove"
		return result
	}

	if !scanner.IsWritable(entry.Path) {
		result.Error = fmt.Sprintf(
			"MetaClean can inspect %s files but ExifTool cannot write/remove metadata from this format; only read-only inspection is supported for it",
			entry.Ext,
		)
		return result
	}

	if req.ReplaceOriginal {
		return e.cleanInPlace(ctx, req, entry, tagArgs, result)
	}
	return e.cleanToCopy(ctx, req, entry, tagArgs, result)
}

// cleanToCopy is the default, safe path: the original is never opened for
// writing. ExifTool writes the cleaned result to a temp file, which is
// only renamed into place once ExifTool reports success — so a failure or
// interruption never leaves a half-written file at the final name.
func (e *Engine) cleanToCopy(ctx context.Context, req model.CleanRequest, entry *model.FileEntry, tagArgs []string, result model.CleanFileResult) model.CleanFileResult {
	outputPath := OutputPath(entry.Path, req.OutputFolder, req.Suffix)
	tempPath := outputPath + ".mcwrite.tmp"
	_ = os.Remove(tempPath)

	err := e.Runner.WriteArgs(ctx, entry.Path, tempPath, tagArgs)
	if err != nil {
		_ = os.Remove(tempPath)
		result.Error = describeCleanError(err)
		return result
	}

	if err := os.Rename(tempPath, outputPath); err != nil {
		_ = os.Remove(tempPath)
		result.Error = fmt.Sprintf("cleaned file could not be finalized: %v", err)
		return result
	}

	result.OutputPath = outputPath
	result.Success = e.verify(ctx, outputPath, &result)
	return result
}

// cleanInPlace is "Replace original" mode: it copies the original to an
// explicit backup first, then lets ExifTool modify the file in place.
// ExifTool itself also keeps its own "<name>_original" backup (we never
// pass -overwrite_original), so there are two independent recovery paths.
func (e *Engine) cleanInPlace(ctx context.Context, req model.CleanRequest, entry *model.FileEntry, tagArgs []string, result model.CleanFileResult) model.CleanFileResult {
	backupPath := BackupPath(entry.Path)
	if err := copyFile(entry.Path, backupPath); err != nil {
		result.Error = fmt.Sprintf("could not create backup, aborting before touching the original: %v", err)
		return result
	}
	result.BackupPath = backupPath

	if err := e.Runner.WriteArgsInPlace(ctx, entry.Path, tagArgs); err != nil {
		// A failed or cancelled write may have changed the file. Re-read it
		// without the cancelled write context; never display the old scan.
		refreshCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		result.OutputPath = entry.Path
		e.verify(refreshCtx, entry.Path, &result)
		verificationError := result.Error
		result.Error = "in-place cleaning failed; inspect the current file and use the backup if needed: " + describeCleanError(err)
		if verificationError != "" {
			result.Error += "; " + verificationError
		}
		result.Inspection.CleanError = result.Error
		return result
	}

	result.OutputPath = entry.Path
	result.Success = e.verify(ctx, entry.Path, &result)
	return result
}

// verify re-scans the cleaned file and records what metadata (if any)
// remains, so the UI can show an honest before/after comparison instead
// of assuming the removal was complete. It reports whether verification
// itself succeeded: ExifTool reporting success on the write is not
// sufficient on its own to call the file successfully cleaned — if
// MetaClean can't re-read the result, it cannot honestly claim success,
// even though remaining metadata (including zero) is a perfectly valid
// verified outcome.
func (e *Engine) verify(ctx context.Context, path string, result *model.CleanFileResult) bool {
	verified := e.Scanner.Scan(ctx, result.FileID, path)
	verified.CleanedPath = path
	result.Inspection = verified
	if verified.Status == model.StatusError {
		result.Error = "metadata removal ran, but MetaClean could not verify the result: " + verified.Error
		verified.CleanError = result.Error
		return false
	}
	result.AfterCount = verified.MetadataCount()
	result.Remaining = verified.Metadata
	return true
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	tmp := dst + ".mcbackup.tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

func describeCleanError(err error) string {
	var runErr *exiftool.RunError
	if errors.As(err, &runErr) {
		return runErr.Error()
	}
	return err.Error()
}
