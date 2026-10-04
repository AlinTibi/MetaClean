package exiftool

import "testing"

func TestRunErrorMessageIncludesStderr(t *testing.T) {
	err := &RunError{Args: []string{"-json", "x.jpg"}, ExitCode: 1, Stderr: "Error: File not found - x.jpg"}
	msg := err.Error()
	if msg != "exiftool exited with code 1: Error: File not found - x.jpg" {
		t.Errorf("unexpected error message: %q", msg)
	}
}

func TestRunErrorMessageWithoutStderr(t *testing.T) {
	err := &RunError{ExitCode: 2}
	msg := err.Error()
	if msg != "exiftool exited with code 2" {
		t.Errorf("unexpected error message: %q", msg)
	}
}
