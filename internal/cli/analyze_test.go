package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzePythonProject(t *testing.T) {
	dir := t.TempDir() // a fresh folder, deleted automatically after the test
	writeFile(t, dir, "requirements.txt", "fastapi==0.115.0\nasyncpg\nhttpx\n")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"analyze", dir}, noInput(), &stdout, &stderr)

	if code != exitOK {
		t.Fatalf("exit code = %d, want %d (stderr: %q)", code, exitOK, stderr.String())
	}
	for _, want := range []string{
		"Project: " + dir,
		"Language:         Python",
		"Package manager:  pip",
		"Framework:        FastAPI",
		"    HTTP client      httpx",
		"    PostgreSQL       asyncpg",
		"OpenTelemetry:    not installed",
		"Recommended instrumentation (✓ installed, + to add):",
		"    + opentelemetry-distro ",
		"    + opentelemetry-instrumentation-fastapi ",
		"Start with:       opentelemetry-instrument uvicorn main:app",
	} {
		assertContains(t, "stdout", stdout.String(), want)
	}
}

func TestAnalyzeUnsupportedLanguageSaysSo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app\n")

	var stdout bytes.Buffer
	code := Run([]string{"analyze", dir}, noInput(), &stdout, &bytes.Buffer{})

	if code != exitOK {
		t.Fatalf("exit code = %d, want %d", code, exitOK)
	}
	assertContains(t, "stdout", stdout.String(), "detection for Go is not supported yet")
}

func TestAnalyzeErrors(t *testing.T) {
	tests := []struct {
		name       string
		dir        string
		wantStderr string
	}{
		{"empty folder", t.TempDir(), "no supported project found"},
		{"missing folder", filepath.Join(t.TempDir(), "nope"), "cannot open the project folder"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			code := Run([]string{"analyze", tt.dir}, noInput(), &bytes.Buffer{}, &stderr)

			if code != exitError {
				t.Errorf("exit code = %d, want %d", code, exitError)
			}
			assertContains(t, "stderr", stderr.String(), tt.wantStderr)
		})
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
