package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"sync"

	"MetaClean/internal/cleaner"
	"MetaClean/internal/exiftool"
	"MetaClean/internal/model"
	"MetaClean/internal/report"
	"MetaClean/internal/scanner"
	"MetaClean/internal/settings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the thin Wails binding layer. All real logic lives in
// internal/exiftool, internal/scanner, internal/cleaner, internal/report
// and internal/settings; App only wires the frontend to them.
type App struct {
	ctx context.Context

	runner    *exiftool.Runner
	scanner   *scanner.Engine
	cleaner   *cleaner.Engine
	engineErr string

	mu       sync.Mutex
	files    map[string]*model.FileEntry
	order    []string
	nextID   int
	cleaning bool
	cancel   context.CancelFunc
}

func NewApp() *App {
	return &App{files: make(map[string]*model.FileEntry)}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	runner, err := exiftool.NewRunner()
	if err != nil {
		a.engineErr = err.Error()
	} else {
		a.runner = runner
		a.scanner = scanner.NewEngine(runner)
		a.cleaner = cleaner.NewEngine(a.scanner, runner)
	}

	wailsruntime.OnFileDrop(ctx, func(x, y int, paths []string) {
		entries, err := a.addPaths(paths)
		wailsruntime.EventsEmit(ctx, "files:added", entries, errString(err))
	})
}

// EngineStatus describes whether the bundled ExifTool engine is ready.
type EngineStatus struct {
	Ready   bool   `json:"ready"`
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
	Error   string `json:"error,omitempty"`
}

// GetEngineStatus reports whether the bundled ExifTool was found and
// runs, so the UI can show a clear error instead of failing silently on
// the first scan.
func (a *App) GetEngineStatus() EngineStatus {
	if a.runner == nil {
		return EngineStatus{Ready: false, Error: a.engineErr}
	}
	version, err := a.runner.Version(a.ctx)
	if err != nil {
		return EngineStatus{Ready: false, Path: a.runner.Path, Error: err.Error()}
	}
	return EngineStatus{Ready: true, Path: a.runner.Path, Version: version}
}

// AddFilesDialog shows a native multi-file picker and scans whatever was
// selected.
func (a *App) AddFilesDialog() ([]*model.FileEntry, error) {
	paths, err := wailsruntime.OpenMultipleFilesDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Add files",
	})
	if err != nil || len(paths) == 0 {
		return nil, err
	}
	return a.addPaths(paths)
}

// AddFolderDialog shows a native folder picker and scans every supported
// file found inside it (recursively).
func (a *App) AddFolderDialog() ([]*model.FileEntry, error) {
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Add folder",
	})
	if err != nil || dir == "" {
		return nil, err
	}
	return a.addPaths([]string{dir})
}

// AddPaths scans files or folders given as absolute paths — used for
// drag & drop as well as being reusable from the dialog handlers.
func (a *App) AddPaths(paths []string) ([]*model.FileEntry, error) {
	return a.addPaths(paths)
}

func (a *App) addPaths(paths []string) ([]*model.FileEntry, error) {
	if a.scanner == nil {
		return nil, fmt.Errorf("ExifTool engine is not available: %s", a.engineErr)
	}

	var files []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.IsDir() {
			_ = filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				if scanner.IsSupported(path) {
					files = append(files, path)
				}
				return nil
			})
		} else {
			files = append(files, p)
		}
	}

	var added []*model.FileEntry
	a.mu.Lock()
	for _, p := range files {
		id := a.newID()
		entry := a.scanner.Scan(a.ctx, id, p)
		a.files[id] = entry
		a.order = append(a.order, id)
		added = append(added, entry)
	}
	a.mu.Unlock()

	return added, nil
}

func (a *App) newID() string {
	a.nextID++
	return "f" + strconv.Itoa(a.nextID)
}

// GetFiles returns the current queue in the order files were added.
func (a *App) GetFiles() []*model.FileEntry {
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make([]*model.FileEntry, 0, len(a.order))
	for _, id := range a.order {
		if e, ok := a.files[id]; ok {
			out = append(out, e)
		}
	}
	return out
}

// RemoveFiles drops the given file IDs from the queue.
func (a *App) RemoveFiles(ids []string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	remove := map[string]bool{}
	for _, id := range ids {
		remove[id] = true
		delete(a.files, id)
	}
	kept := a.order[:0:0]
	for _, id := range a.order {
		if !remove[id] {
			kept = append(kept, id)
		}
	}
	a.order = kept
}

// ClearFiles empties the queue.
func (a *App) ClearFiles() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.files = make(map[string]*model.FileEntry)
	a.order = nil
}

// CleanProgress is emitted on the "clean:progress" event after each file.
type CleanProgress struct {
	Index  int                   `json:"index"`
	Total  int                   `json:"total"`
	Result model.CleanFileResult `json:"result"`
}

// StartClean runs a cleaning batch in the background and streams progress
// via "clean:progress", finishing with a "clean:done" event. It returns
// immediately so the UI stays responsive.
func (a *App) StartClean(req model.CleanRequest) error {
	if a.cleaner == nil {
		return fmt.Errorf("ExifTool engine is not available: %s", a.engineErr)
	}

	a.mu.Lock()
	if a.cleaning {
		a.mu.Unlock()
		return fmt.Errorf("a cleaning batch is already running")
	}
	var entries []*model.FileEntry
	for _, id := range req.FileIDs {
		if e, ok := a.files[id]; ok {
			entries = append(entries, e)
		}
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancel = cancel
	a.cleaning = true
	a.mu.Unlock()

	go func() {
		defer cancel()

		results := a.cleaner.CleanBatch(ctx, req, entries, func(index, total int, result model.CleanFileResult) {
			a.applyCleanResult(result)
			wailsruntime.EventsEmit(a.ctx, "clean:progress", CleanProgress{Index: index, Total: total, Result: result})
		})

		// A done event enables Clean immediately: release the batch lock first.
		a.mu.Lock()
		a.cleaning = false
		a.cancel = nil
		a.mu.Unlock()
		wailsruntime.EventsEmit(a.ctx, "clean:done", results)
	}()

	return nil
}

// Replace the queue snapshot before notifying the UI, so inspectors and
// exports describe the same actual file. Unchanged failures retain their scan
// and carry the cleaning error into exports as well as the batch summary.
func (a *App) applyCleanResult(result model.CleanFileResult) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if entry, exists := a.files[result.FileID]; exists {
		if result.Inspection != nil {
			a.files[result.FileID] = result.Inspection
		} else if !result.Success && result.Error != "" {
			current := *entry
			current.CleanError = result.Error
			a.files[result.FileID] = &current
		}
	}
}

// CancelClean stops an in-progress batch after the current file finishes
// (the in-flight exiftool process is killed immediately via context
// cancellation; already-written output files are left as-is).
func (a *App) CancelClean() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
}

// SelectOutputFolder shows a native folder picker.
func (a *App) SelectOutputFolder() (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select output folder",
	})
}

// OpenOutputFolder opens dir in the Windows file explorer by invoking
// explorer.exe directly (no shell interpreter involved).
func (a *App) OpenOutputFolder(dir string) error {
	cmd := exec.Command("explorer", dir)
	return cmd.Start()
}

// ExportMetadataCSV shows a native save dialog and writes a CSV metadata
// report covering every file currently in the queue.
func (a *App) ExportMetadataCSV() (string, error) {
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Export metadata report",
		DefaultFilename: "metaclean-report.csv",
		Filters:         []wailsruntime.FileFilter{{DisplayName: "CSV (*.csv)", Pattern: "*.csv"}},
	})
	if err != nil || path == "" {
		return "", err
	}

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err := report.WriteMetadataCSV(a.sortedFiles(), f); err != nil {
		return "", err
	}
	return path, nil
}

// ExportPrivacyReportJSON shows a native save dialog and writes a JSON
// privacy report covering every file currently in the queue.
func (a *App) ExportPrivacyReportJSON() (string, error) {
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Export privacy report",
		DefaultFilename: "metaclean-privacy-report.json",
		Filters:         []wailsruntime.FileFilter{{DisplayName: "JSON (*.json)", Pattern: "*.json"}},
	})
	if err != nil || path == "" {
		return "", err
	}

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if err := report.WritePrivacyReportJSON(a.sortedFiles(), f); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) sortedFiles() []*model.FileEntry {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]*model.FileEntry, 0, len(a.order))
	for _, id := range a.order {
		if e, ok := a.files[id]; ok {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// GetSettings returns the persisted settings, or defaults on first run.
func (a *App) GetSettings() model.Settings {
	s, err := settings.Load()
	if err != nil {
		return model.DefaultSettings()
	}
	return s
}

// SaveSettings persists settings to disk.
func (a *App) SaveSettings(s model.Settings) error {
	return settings.Save(s)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
