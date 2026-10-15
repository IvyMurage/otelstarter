package check

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IvyMurage/otelstarter/internal/project"
)

// fakeCollector answers OTLP paths with 200 and anything else with 404,
// like a real Collector's OTLP/HTTP receiver.
func fakeCollector(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/") {
			w.Write([]byte(`{"partialSuccess":{}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close) // runs when the test ends, like defer
	return server
}

// closedPort returns an address where nothing is listening.
func closedPort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return "http://" + addr
}

func TestCheckCollector(t *testing.T) {
	ok := fakeCollector(t)
	notCollector := httptest.NewServer(http.NotFoundHandler())
	defer notCollector.Close()

	tests := []struct {
		name      string
		endpoint  string
		want      Status
		wantTitle string
	}{
		{"collector accepts OTLP", ok.URL, Pass, "accepts traces, metrics and logs"},
		{"trailing slash is fine", ok.URL + "/", Pass, "accepts"},
		{"something else is listening", notCollector.URL, Fail, "answered 404"},
		{"nothing is listening", closedPort(t), Fail, "Nothing is listening"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkCollector(context.Background(), tt.endpoint)
			if got.Status != tt.want || !strings.Contains(got.Title, tt.wantTitle) {
				t.Errorf("checkCollector() = %v %q, want %v containing %q", got.Status, got.Title, tt.want, tt.wantTitle)
			}
		})
	}
}

func TestCheckPackages(t *testing.T) {
	lang := project.Language{Instrumentation: []project.Instrumentation{
		{Package: "opentelemetry-distro"},
		{Package: "opentelemetry-exporter-otlp-proto-http", Installed: true},
		{Package: "opentelemetry-instrumentation-redis"},
	}}

	results := checkPackages(lang)

	if len(results) != 2 {
		t.Fatalf("checkPackages() = %v, want a failure and a warning", results)
	}
	if results[0].Status != Fail || !strings.Contains(results[0].Title, "opentelemetry-distro") {
		t.Errorf("first result = %+v, want a failure naming opentelemetry-distro", results[0])
	}
	if results[1].Status != Warn || !strings.Contains(results[1].Title, "opentelemetry-instrumentation-redis") {
		t.Errorf("second result = %+v, want a warning naming the redis instrumentation", results[1])
	}
}

// A Node.js project on the bundle an earlier otelstarter installed still
// sends telemetry, so a missing SDK must not fail validate in its CI.
func TestCheckPackagesNodeBundleIsNotAFailure(t *testing.T) {
	lang := project.Language{
		OpenTelemetry:   []string{"@opentelemetry/auto-instrumentations-node"},
		Instrumentation: []project.Instrumentation{{Package: "otelstarter"}},
	}
	for _, r := range checkPackages(lang) {
		if r.Status == Fail {
			t.Errorf("checkPackages() failed: %q, want only a warning", r.Title)
		}
	}
	lang.OpenTelemetry = nil
	if results := checkPackages(lang); results[0].Status != Fail {
		t.Errorf("without the bundle, checkPackages() = %+v, want a failure", results[0])
	}
}

// project writes a set-up Python project whose otel.env points at endpoint.
func setUpProject(t *testing.T, endpoint string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"requirements.txt": "fastapi\nopentelemetry-distro==0.66b0\nopentelemetry-exporter-otlp-proto-http==1.45.0\n" +
			"opentelemetry-instrumentation-logging==0.66b0\nopentelemetry-instrumentation-system-metrics==0.66b0\n" +
			"opentelemetry-instrumentation-fastapi==0.66b0\notelstarter==0.3.0\n",
		"otel.env": "OTEL_SERVICE_NAME=shop\nOTEL_EXPORTER_OTLP_ENDPOINT=" + endpoint + "\nOTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf\n" +
			"OTEL_TRACES_EXPORTER=otelstarter\nOTEL_LOGS_EXPORTER=otelstarter\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func titles(results []Result) string {
	var b strings.Builder
	for _, r := range results {
		b.WriteString(r.Title + "\n")
	}
	return b.String()
}

func TestRunSetUpProjectPasses(t *testing.T) {
	usePython(t, "3.12.4") // independent of the Python on this machine
	dir := setUpProject(t, fakeCollector(t).URL)

	results := Run(context.Background(), Options{Dir: dir})

	if failures, _ := Count(results); failures != 0 {
		t.Errorf("failures = %d, want 0. Results:\n%s", failures, titles(results))
	}
	if !strings.Contains(titles(results), "accepts traces, metrics and logs") {
		t.Errorf("Collector check missing from results:\n%s", titles(results))
	}
}

func TestRunOfflineSkipsCollector(t *testing.T) {
	usePython(t, "3.12.4") // independent of the Python on this machine
	dir := setUpProject(t, closedPort(t))

	results := Run(context.Background(), Options{Dir: dir, Offline: true})

	if failures, _ := Count(results); failures != 0 {
		t.Errorf("failures = %d, want 0 offline. Results:\n%s", failures, titles(results))
	}
	// Match the Collector check's own wording: temp folder names in other
	// titles contain this test's name, which includes "Collector".
	if strings.Contains(titles(results), "accepts traces") || strings.Contains(titles(results), "Nothing is listening") {
		t.Errorf("offline run checked the Collector:\n%s", titles(results))
	}
}

func TestRunStopsEarly(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		wantTitle string
	}{
		{"no project", nil, "no supported project found"},
		{"unsupported language", map[string]string{"go.mod": "module x\n"}, "supports Python and Node.js projects for now"},
		{"no otel.env", map[string]string{"requirements.txt": "fastapi\n"}, "otel.env"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			results := Run(context.Background(), Options{Dir: dir, Offline: true})

			var failed []string
			for _, r := range results {
				if r.Status == Fail {
					failed = append(failed, r.Title+" "+r.Why)
				}
			}
			if !strings.Contains(strings.Join(failed, "\n"), tt.wantTitle) {
				t.Errorf("failures = %q, want one mentioning %q", failed, tt.wantTitle)
			}
		})
	}
}

func TestEnvironmentMasksSecrets(t *testing.T) {
	dir := setUpProject(t, "http://localhost:4318")
	f, err := os.OpenFile(filepath.Join(dir, "otel.env"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("OTEL_EXPORTER_OTLP_HEADERS=Authorization=Bearer sk-live-123\n")
	f.Close()

	report := Environment(context.Background(), dir, "test")

	all := strings.Join(report.Settings, "\n")
	if strings.Contains(all, "sk-live-123") {
		t.Errorf("report contains the secret:\n%s", all)
	}
	if !strings.Contains(all, "OTEL_EXPORTER_OTLP_HEADERS=*** (hidden)") || !strings.Contains(all, "OTEL_SERVICE_NAME=shop") {
		t.Errorf("settings = %q, want the header masked and other settings shown", report.Settings)
	}
}
