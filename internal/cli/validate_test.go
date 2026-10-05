package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newSetUpProject(t *testing.T, endpoint string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "fastapi\nopentelemetry-distro==0.66b0\nopentelemetry-exporter-otlp-proto-http==1.45.0\n")
	writeFile(t, dir, "otel.env", "OTEL_SERVICE_NAME=shop\nOTEL_EXPORTER_OTLP_ENDPOINT="+endpoint+"\nOTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf\n")
	return dir
}

func runCommand(args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = Run(args, noInput(), &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestValidateExitCodes(t *testing.T) {
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer collector.Close()

	t.Run("warnings only exit 0", func(t *testing.T) {
		// Missing logging and system-metrics instrumentation are warnings.
		code, stdout, stderr := runCommand("validate", newSetUpProject(t, collector.URL))
		if code != exitOK {
			t.Errorf("exit code = %d, want %d\nstdout:\n%s", code, exitOK, stdout)
		}
		assertContains(t, "stdout", stdout, "✓ Collector at "+collector.URL)
		assertContains(t, "stdout", stdout, "0 problems,")
		if stderr != "" {
			t.Errorf("stderr = %q, want nothing: the checklist is the whole output", stderr)
		}
	})

	t.Run("a failure exits 1 with why and fix", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "requirements.txt", "fastapi\n")
		writeFile(t, dir, "otel.env", "OTEL_SERVICE_NAME=shop\n")

		code, stdout, stderr := runCommand("validate", "--offline", dir)
		if code != exitError {
			t.Errorf("exit code = %d, want %d", code, exitError)
		}
		assertContains(t, "stdout", stdout, "✗ OTEL_EXPORTER_OTLP_PROTOCOL is not set")
		assertContains(t, "stdout", stdout, "Why: ")
		assertContains(t, "stdout", stdout, "Fix: Add OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf to otel.env.")
		if stderr != "" {
			t.Errorf("stderr = %q, want nothing", stderr)
		}
	})
}

func TestDoctorPrintsChecksAndReport(t *testing.T) {
	dir := newSetUpProject(t, "http://localhost:4318")
	writeFile(t, dir, "otel.env", "OTEL_SERVICE_NAME=shop\nOTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf\nMY_API_KEY=sk-live-123\n")

	_, stdout, _ := runCommand("doctor", dir)

	for _, want := range []string{"Checks:", "Environment (secrets are hidden", "Go runtime:", "Ports:", "otel.env:", "MY_API_KEY=*** (hidden)"} {
		assertContains(t, "stdout", stdout, want)
	}
	if bytes.Contains([]byte(stdout), []byte("sk-live-123")) {
		t.Error("doctor printed the secret value")
	}
}
