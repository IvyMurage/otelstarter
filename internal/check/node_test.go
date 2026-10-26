package check

import (
	"context"
	"strings"
	"testing"

	"github.com/IvyMurage/otelstarter/internal/project"
)

// useNode makes node on PATH report version. PATH holds nothing else.
func useNode(t *testing.T, version string) {
	t.Helper()
	dir := t.TempDir()
	writeExecutable(t, dir, "node", "echo 'v"+version+"'")
	t.Setenv("PATH", dir)
}

func TestCheckNode(t *testing.T) {
	tests := []struct {
		version string
		want    Status
	}{
		{"22.23.1", Pass},
		{"20.6.0", Pass},
		{"20.5.1", Fail}, // 20.x needs 20.6+
		{"18.19.0", Pass},
		{"18.18.2", Fail},
		{"16.20.0", Fail},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			useNode(t, tt.version)
			if got := checkNode(context.Background(), t.TempDir(), false); got.Status != tt.want {
				t.Errorf("checkNode() = %v %q, want %v", got.Status, got.Title, tt.want)
			}
		})
	}
}

func TestCheckNodeMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if got := checkNode(context.Background(), t.TempDir(), false); got.Status != Warn {
		t.Errorf("checkNode() = %v %q, want a warning", got.Status, got.Title)
	}
}

func TestCheckNodeForESModuleApps(t *testing.T) {
	tests := []struct {
		version string
		want    Status
	}{
		{"22.14.0", Pass},
		{"24.1.0", Pass},
		{"23.7.0", Pass},
		{"22.13.1", Warn}, // runs, but without the import hook
		{"20.19.0", Warn},
		{"18.18.2", Fail}, // too old for OpenTelemetry at all
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			useNode(t, tt.version)
			if got := checkNode(context.Background(), t.TempDir(), true); got.Status != tt.want {
				t.Errorf("checkNode() = %v %q, want %v", got.Status, got.Title, tt.want)
			}
		})
	}
}

func TestCheckNodeAdviceUsesFirstSentenceAsTitle(t *testing.T) {
	results := checkNodeAdvice(project.Language{Notes: []string{"No logging library found. Use pino."}})
	if len(results) != 1 || results[0].Status != Warn || results[0].Title != "No logging library found" ||
		!strings.Contains(results[0].Why, "Use pino") {
		t.Errorf("checkNodeAdvice() = %+v", results)
	}
}
