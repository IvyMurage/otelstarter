package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
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

func TestPrepareAppEchoesEventsUnlessTheShellSaysOtherwise(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "otel.env", "OTEL_SERVICE_NAME=shop\n")
	t.Chdir(dir) // prepareApp reads otel.env from the current folder

	app, err := prepareApp([]string{"true"}, streams{})
	if err != nil {
		t.Fatalf("prepareApp() error = %v", err)
	}
	assertEnv(t, app.Env, "OTELSTARTER_CONSOLE_EVENTS=true")
	assertEnv(t, app.Env, "OTEL_SERVICE_NAME=shop")

	t.Setenv("OTELSTARTER_CONSOLE_EVENTS", "false")
	app, err = prepareApp([]string{"true"}, streams{})
	if err != nil {
		t.Fatalf("prepareApp() error = %v", err)
	}
	assertEnv(t, app.Env, "OTELSTARTER_CONSOLE_EVENTS=false")
}

func assertEnv(t *testing.T, env []string, want string) {
	t.Helper()
	if !slices.Contains(env, want) {
		t.Errorf("app environment is missing %q", want)
	}
}

func TestPrepareAppForNode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "otel.env", "OTEL_SERVICE_NAME=shop\n")
	writeFile(t, dir, "package.json", `{"dependencies": {"express": "^4"}}`)
	t.Chdir(dir)

	if _, err := prepareApp([]string{"true"}, streams{}); err == nil ||
		!strings.Contains(err.Error(), "is not installed in this folder") {
		t.Fatalf("prepareApp() without node_modules: error = %v, want a 'not installed' error", err)
	}

	pkgDir := filepath.Join("node_modules", "otelstarter")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, pkgDir, "package.json", "{}")
	t.Setenv("NODE_OPTIONS", "--max-old-space-size=4096")

	app, err := prepareApp([]string{"true"}, streams{})
	if err != nil {
		t.Fatalf("prepareApp() error = %v", err)
	}
	assertEnv(t, app.Env, "NODE_OPTIONS=--max-old-space-size=4096 --require otelstarter/register")
	if app.Args[0] != "true" {
		t.Errorf("command = %q, want it unchanged (Node is instrumented through NODE_OPTIONS)", app.Args)
	}
}
