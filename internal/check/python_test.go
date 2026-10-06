package check

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeExecutable creates a small shell script, standing in for a real
// program such as python3, so tests do not depend on the machine.
func writeExecutable(t *testing.T, dir, name, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake programs are shell scripts")
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// usePython makes python3 on PATH report version. PATH holds nothing else.
func usePython(t *testing.T, version string) {
	t.Helper()
	dir := t.TempDir()
	writeExecutable(t, dir, "python3", "echo 'Python "+version+"'")
	t.Setenv("PATH", dir)
}

func TestCheckPython(t *testing.T) {
	tests := []struct {
		version   string
		want      Status
		wantTitle string
	}{
		{"3.12.4", Pass, "Python 3.12.4"},
		{"3.10.0", Pass, "Python 3.10.0"},
		{"3.9.18", Fail, "Python 3.9.18 is too old"},
		{"3.8.13", Fail, "Python 3.8.13 is too old"},
		{"not-a-version", Warn, "Cannot tell which Python version"},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			usePython(t, tt.version)

			got := checkPython(context.Background(), t.TempDir())

			if got.Status != tt.want || !strings.Contains(got.Title, tt.wantTitle) {
				t.Errorf("checkPython() = %v %q, want %v containing %q", got.Status, got.Title, tt.want, tt.wantTitle)
			}
		})
	}
}

func TestCheckPythonMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if got := checkPython(context.Background(), t.TempDir()); got.Status != Warn {
		t.Errorf("checkPython() = %v %q, want a warning", got.Status, got.Title)
	}
}

// The virtual environment's Python is the one that runs the app, even when
// another python3 comes first on PATH.
func TestCheckPythonPrefersTheVirtualEnvironment(t *testing.T) {
	system, venv := t.TempDir(), t.TempDir()
	writeExecutable(t, system, "python3", "echo 'Python 3.8.13'")
	writeExecutable(t, venv, "python3", "echo 'Python 3.12.4'")
	writeExecutable(t, venv, "opentelemetry-instrument", "true")
	t.Setenv("PATH", system+string(os.PathListSeparator)+venv)

	got := checkPython(context.Background(), t.TempDir())

	if got.Status != Pass || !strings.Contains(got.Title, venv) {
		t.Errorf("checkPython() = %v %q, want a pass using the interpreter in %s", got.Status, got.Title, venv)
	}
}

// pyenv picks the version from .python-version in the current folder, so the
// check must run in the project folder, not wherever otelstarter was started.
func TestCheckPythonRunsInTheProjectFolder(t *testing.T) {
	bin, project := t.TempDir(), t.TempDir()
	writeExecutable(t, bin, "python3", `v=3.8.0; if [ -f .python-version ]; then read -r v < .python-version; fi; echo "Python $v"` /* shell built-ins only: PATH holds nothing else */)
	t.Setenv("PATH", bin)
	if err := os.WriteFile(filepath.Join(project, ".python-version"), []byte("3.12.1"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := checkPython(context.Background(), project); got.Status != Pass {
		t.Errorf("checkPython() = %v %q, want the project's Python 3.12.1", got.Status, got.Title)
	}
}
