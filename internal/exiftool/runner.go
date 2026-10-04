package exiftool

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes a specific exiftool.exe binary. It never touches the
// shell (no cmd.exe, no PowerShell) and never reaches the network.
type Runner struct {
	Path string
}

// NewRunner locates the bundled exiftool executable and returns a Runner
// for it.
func NewRunner() (*Runner, error) {
	path, err := Locate()
	if err != nil {
		return nil, err
	}
	return &Runner{Path: path}, nil
}

// RunError wraps a failed exiftool invocation with its exit code and
// captured stderr, so callers can show the operator something useful
// instead of a bare "exit status 1".
type RunError struct {
	Args     []string
	ExitCode int
	Stderr   string
}

func (e *RunError) Error() string {
	stderr := strings.TrimSpace(e.Stderr)
	if stderr == "" {
		return fmt.Sprintf("exiftool exited with code %d", e.ExitCode)
	}
	return fmt.Sprintf("exiftool exited with code %d: %s", e.ExitCode, stderr)
}

// run executes exiftool with args and returns stdout. A non-zero exit
// code produces a *RunError carrying stderr for diagnostics.
func (r *Runner) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, r.Path, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		exitCode := -1
		var exitErr *exec.ExitError
		if ok := asExitError(err, &exitErr); ok {
			exitCode = exitErr.ExitCode()
		}
		return stdout.Bytes(), &RunError{Args: args, ExitCode: exitCode, Stderr: stderr.String()}
	}
	return stdout.Bytes(), nil
}

func asExitError(err error, target **exec.ExitError) bool {
	if ee, ok := err.(*exec.ExitError); ok {
		*target = ee
		return true
	}
	return false
}

// ReadJSON runs exiftool in metadata-extraction mode for a single file and
// returns the raw JSON it printed (a one-element JSON array).
func (r *Runner) ReadJSON(ctx context.Context, path string) ([]byte, error) {
	return r.run(ctx, "-json", "-G1", "-a", "-charset", "filename=utf8", path)
}

// WriteArgs runs exiftool with caller-supplied tag-removal arguments
// against a single input file, writing the result to outputPath via
// ExifTool's own "-o" copy mode. The original file at path is never
// opened for writing in this mode, so a failure here cannot corrupt or
// truncate it.
func (r *Runner) WriteArgs(ctx context.Context, path string, outputPath string, tagArgs []string) error {
	args := make([]string, 0, len(tagArgs)+5)
	args = append(args, tagArgs...)
	args = append(args, "-o", outputPath, "-charset", "filename=utf8", path)
	_, err := r.run(ctx, args...)
	return err
}

// WriteArgsInPlace runs exiftool with caller-supplied tag-removal
// arguments directly against path, without "-overwrite_original" — so
// ExifTool keeps its own "<name>_original" backup copy in addition to any
// backup the caller already made.
func (r *Runner) WriteArgsInPlace(ctx context.Context, path string, tagArgs []string) error {
	args := make([]string, 0, len(tagArgs)+3)
	args = append(args, tagArgs...)
	args = append(args, "-charset", "filename=utf8", path)
	_, err := r.run(ctx, args...)
	return err
}

// Version returns the bundled exiftool's reported version string.
func (r *Runner) Version(ctx context.Context) (string, error) {
	out, err := r.run(ctx, "-ver")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
