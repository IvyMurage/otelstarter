package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runInitIn(t *testing.T, dir string, flags ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	args := append([]string{"init"}, flags...)
	code = Run(append(args, dir), noInput(), &out, &errOut)
	return code, out.String(), errOut.String()
}

func newPipProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "fastapi\n")
	return dir
}

func TestInitDryRunWritesNothing(t *testing.T) {
	dir := newPipProject(t)

	code, stdout, _ := runInitIn(t, dir, "--dry-run")

	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	for _, want := range []string{
		"Files to change:",
		"create  otel.env",
		"update  requirements.txt",
		"  | OTEL_SERVICE_NAME=",
		"  | opentelemetry-instrumentation-fastapi==",
		"pip install -r requirements.txt",
		"otelstarter run -- uvicorn main:app",
		"Dry run: no files were changed.",
	} {
		assertContains(t, "stdout", stdout, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "otel.env")); !os.IsNotExist(err) {
		t.Error("dry run created otel.env")
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "requirements.txt")); string(got) != "fastapi\n" {
		t.Errorf("dry run changed requirements.txt to %q", got)
	}
}

func TestInitWithoutTerminalOrYesRefuses(t *testing.T) {
	dir := newPipProject(t)

	code, _, stderr := runInitIn(t, dir)

	if code != exitError {
		t.Errorf("exit code = %d, want %d", code, exitError)
	}
	assertContains(t, "stderr", stderr, "pass --yes")
	if _, err := os.Stat(filepath.Join(dir, "otel.env")); !os.IsNotExist(err) {
		t.Error("otel.env was written without confirmation")
	}
}

func TestInitYesAppliesThenSecondRunHasNothingToDo(t *testing.T) {
	dir := newPipProject(t)

	code, stdout, stderr := runInitIn(t, dir, "--yes")
	if code != exitOK {
		t.Fatalf("first run exit code = %d (stderr: %q)", code, stderr)
	}
	assertContains(t, "stdout", stdout, "Created otel.env")
	assertContains(t, "stdout", stdout, "Updated requirements.txt")

	code, stdout, _ = runInitIn(t, dir, "--yes")
	if code != exitOK {
		t.Fatalf("second run exit code = %d", code)
	}
	assertContains(t, "stdout", stdout, "Already set up. Nothing to change.")
}

func TestInitUnsupportedProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", "{}")

	code, _, stderr := runInitIn(t, dir, "--yes")

	if code != exitError {
		t.Errorf("exit code = %d, want %d", code, exitError)
	}
	assertContains(t, "stderr", stderr, "supports Python projects only for now")
}

func TestConfirm(t *testing.T) {
	tests := map[string]bool{
		"y\n":   true,
		"YES\n": true,
		" y \n": true,
		"\n":    false, // pressing Enter must never change files
		"n\n":   false,
		"sure":  false,
		"":      false,
	}
	for input, want := range tests {
		var out bytes.Buffer
		if got := confirm(strings.NewReader(input), &out, "Apply? "); got != want {
			t.Errorf("confirm(%q) = %v, want %v", input, got, want)
		}
	}
}
