package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), perm); err != nil {
		t.Fatal(err)
	}
}

func mustReadFile(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func buildPlanFor(t *testing.T, dir string) Plan {
	t.Helper()
	plan, err := BuildPlan(dir)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	return plan
}

func TestApplyThenRerunIsNoOp(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "fastapi\n", 0o600)

	if err := Apply(buildPlanFor(t, dir)); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	reqs := mustReadFile(t, dir, "requirements.txt")
	if !strings.HasPrefix(reqs, "fastapi\n\n# >>> otelstarter") || !strings.Contains(reqs, "opentelemetry-instrumentation-fastapi==") {
		t.Errorf("requirements.txt = %q, want the original line followed by the otelstarter block", reqs)
	}
	if !strings.Contains(mustReadFile(t, dir, "otel.env"), "OTEL_SERVICE_NAME=") {
		t.Error("otel.env was not written")
	}
	if info, _ := os.Stat(filepath.Join(dir, "requirements.txt")); info.Mode().Perm() != 0o600 {
		t.Errorf("requirements.txt permissions = %v, want the original 0600 kept", info.Mode().Perm())
	}
	if matches, _ := filepath.Glob(filepath.Join(dir, "*.otelstarter-tmp")); len(matches) > 0 {
		t.Errorf("temporary files left behind: %v", matches)
	}

	// The second run must find nothing to do: this is what makes init safe
	// to run repeatedly.
	if again := buildPlanFor(t, dir); !again.NothingToDo() {
		t.Errorf("second plan has changes %v and commands %v, want nothing to do", changePaths(again), again.Commands)
	}
}

func TestApplyRefusesFileChangedAfterPlan(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "fastapi\n", 0o644)
	plan := buildPlanFor(t, dir)

	writeFile(t, dir, "requirements.txt", "fastapi\nredis\n", 0o644) // user edits while the prompt is open

	err := Apply(plan)
	if err == nil || !strings.Contains(err.Error(), "requirements.txt changed after the plan was shown") {
		t.Fatalf("Apply() error = %v, want a 'changed after the plan' error", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "otel.env")); !os.IsNotExist(statErr) {
		t.Error("otel.env was written even though Apply refused; changes must be all or nothing")
	}
}

func TestApplyRefusesSymlinkedFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "real-requirements.txt", "fastapi\n", 0o644)
	if err := os.Symlink("real-requirements.txt", filepath.Join(dir, "requirements.txt")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	plan := buildPlanFor(t, dir)

	err := Apply(plan)
	if err == nil || !strings.Contains(err.Error(), "is a symbolic link") {
		t.Fatalf("Apply() error = %v, want a symbolic link error", err)
	}
	if got := mustReadFile(t, dir, "real-requirements.txt"); got != "fastapi\n" {
		t.Errorf("link target changed to %q", got)
	}
}
