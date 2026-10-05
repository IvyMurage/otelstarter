package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	// Table-driven tests: each case is one row, and the loop below runs them all.
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{"no arguments prints usage", nil, exitUsage, "", "Usage:"},
		{"help command", []string{"help"}, exitOK, "Commands:", ""},
		{"--help flag", []string{"--help"}, exitOK, "Commands:", ""},
		{"version", []string{"version"}, exitOK, "otelstarter dev", ""},
		{"unknown command", []string{"deploy"}, exitUsage, "", `Unknown command "deploy"`},
		{"command --help", []string{"version", "--help"}, exitOK, "", "Usage: otelstarter version"},
		{"unknown flag", []string{"version", "--bogus"}, exitUsage, "", "flag provided but not defined"},
		{"stray argument", []string{"version", "extra"}, exitUsage, "", `Too many arguments for otelstarter version: unexpected "extra"`},
		{"unimplemented command", []string{"init"}, exitError, "", "not implemented yet"},
		{"analyze with two paths", []string{"analyze", "a", "b"}, exitUsage, "", `unexpected "b"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(tt.args, &stdout, &stderr)

			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %q)", code, tt.wantCode, stderr.String())
			}
			assertContains(t, "stdout", stdout.String(), tt.wantStdout)
			assertContains(t, "stderr", stderr.String(), tt.wantStderr)
		})
	}
}

func TestHelpListsEveryMVPCommand(t *testing.T) {
	var stdout bytes.Buffer
	Run([]string{"help"}, &stdout, &bytes.Buffer{})

	for _, name := range []string{"init", "analyze", "run", "validate", "doctor", "version"} {
		assertContains(t, "help", stdout.String(), "  "+name+" ")
	}
}

// assertContains passes when want is empty, so table rows can skip a stream.
func assertContains(t *testing.T, stream, got, want string) {
	t.Helper()
	if want != "" && !strings.Contains(got, want) {
		t.Errorf("%s = %q, want it to contain %q", stream, got, want)
	}
}
