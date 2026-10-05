package cli

import (
	"bytes"
	"reflect"
	"testing"
)

func TestSplitAtDoubleDash(t *testing.T) {
	tests := []struct {
		args       []string
		wantBefore []string
		wantAfter  []string
	}{
		{nil, nil, nil},
		{[]string{"--", "uvicorn", "main:app"}, []string{}, []string{"uvicorn", "main:app"}},
		{[]string{"--help"}, []string{"--help"}, nil},
		{[]string{"--", "app", "--", "x"}, []string{}, []string{"app", "--", "x"}}, // only the first -- splits
	}
	for _, tt := range tests {
		before, after := splitAtDoubleDash(tt.args)
		if len(before) != len(tt.wantBefore) || !reflect.DeepEqual(after, tt.wantAfter) {
			t.Errorf("splitAtDoubleDash(%q) = %q, %q; want %q, %q", tt.args, before, after, tt.wantBefore, tt.wantAfter)
		}
	}
}

func TestMergeEnvShellWins(t *testing.T) {
	environ := []string{"PATH=/bin", "OTEL_SERVICE_NAME=from-shell"}
	fileVars := []string{"OTEL_SERVICE_NAME=from-file", "OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318"}

	got := mergeEnv(environ, fileVars)

	want := []string{"PATH=/bin", "OTEL_SERVICE_NAME=from-shell", "OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("mergeEnv() = %q, want %q", got, want)
	}
}

func TestRunWithoutDoubleDashExplainsIt(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"run", "uvicorn", "main:app"}, noInput(), &bytes.Buffer{}, &stderr)

	if code != exitUsage {
		t.Errorf("exit code = %d, want %d", code, exitUsage)
	}
	assertContains(t, "stderr", stderr.String(), "otelstarter run -- uvicorn main:app")
}

// The app is checked before Docker is touched, so this test needs no Docker.
func TestRunAppWithoutEnvFileAsksForInit(t *testing.T) {
	t.Chdir(t.TempDir()) // run from an empty folder; restored after the test

	var stderr bytes.Buffer
	code := Run([]string{"run", "--", "uvicorn", "main:app"}, noInput(), &bytes.Buffer{}, &stderr)

	if code != exitError {
		t.Errorf("exit code = %d, want %d", code, exitError)
	}
	assertContains(t, "stderr", stderr.String(), "otel.env not found")
	assertContains(t, "stderr", stderr.String(), "Run otelstarter init first")
}
