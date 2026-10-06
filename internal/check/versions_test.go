package check

import (
	"strings"
	"testing"

	"github.com/IvyMurage/otelstarter/internal/adapters/python"
	"github.com/IvyMurage/otelstarter/internal/project"
)

func pins(pairs ...string) project.Language {
	var lang project.Language
	for i := 0; i < len(pairs); i += 2 {
		lang.Pins = append(lang.Pins, project.Pin{Package: pairs[i], Version: pairs[i+1]})
	}
	return lang
}

func TestCheckVersions(t *testing.T) {
	tests := []struct {
		name      string
		lang      project.Language
		want      []Status
		wantTitle string
	}{
		{"nothing pinned", pins(), nil, ""},
		{
			"both families consistent",
			pins("opentelemetry-distro", "0.66b0", "opentelemetry-instrumentation-flask", "0.66b0",
				"opentelemetry-exporter-otlp-proto-http", "1.45.0", "opentelemetry-sdk", "1.45.0"),
			[]Status{Pass}, "versions match: 0.66b0, 1.45.0",
		},
		{
			"beta mismatch",
			pins("opentelemetry-distro", "0.66b0", "opentelemetry-instrumentation-flask", "0.60b0"),
			[]Status{Fail}, "beta packages are pinned to different versions: 0.60b0 (opentelemetry-instrumentation-flask), 0.66b0 (opentelemetry-distro)",
		},
		{
			"stable mismatch",
			pins("opentelemetry-api", "1.40.0", "opentelemetry-sdk", "1.45.0"),
			[]Status{Fail}, "stable packages are pinned",
		},
		{
			"same package pinned differently in two manifests",
			pins("opentelemetry-distro", "0.60b0", "opentelemetry-distro", "0.66b0"),
			[]Status{Fail}, "0.60b0 (opentelemetry-distro), 0.66b0 (opentelemetry-distro)",
		},
		{
			"many at one version are counted, the odd one is named",
			pins("opentelemetry-distro", "0.66b0", "opentelemetry-instrumentation-fastapi", "0.66b0",
				"opentelemetry-instrumentation-logging", "0.66b0", "opentelemetry-instrumentation-httpx", "0.60b0"),
			[]Status{Fail}, "0.60b0 (opentelemetry-instrumentation-httpx), 0.66b0 (3 packages)",
		},
		{
			"both families wrong",
			pins("opentelemetry-distro", "0.66b0", "opentelemetry-instrumentation-flask", "0.60b0",
				"opentelemetry-api", "1.40.0", "opentelemetry-sdk", "1.45.0"),
			[]Status{Fail, Fail}, "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := checkVersions(tt.lang)

			var got []Status
			for _, r := range results {
				got = append(got, r.Status)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("statuses = %v, want %v (results %+v)", got, tt.want, results)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("status %d = %v, want %v", i, got[i], tt.want[i])
				}
			}
			if tt.wantTitle != "" && !strings.Contains(results[0].Title, tt.wantTitle) {
				t.Errorf("title = %q, want it to contain %q", results[0].Title, tt.wantTitle)
			}
			if tt.want != nil && tt.want[0] == Fail && !strings.Contains(results[0].Fix, "0.66b0") && !strings.Contains(results[0].Fix, "1.45.0") {
				t.Errorf("fix = %q, want it to name the tested version", results[0].Fix)
			}
		})
	}
}

func TestOlderVersion(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"0.1.0", "0.2.0", true},
		{"0.2.0", "0.2.0", false},
		{"0.3.0", "0.2.0", false},
		{"0.2.0", "0.10.0", true}, // compared as numbers, not text
		{"1.0", "1.0.1", true},
		{"0.2.0rc1", "0.3.0", false}, // not a plain release: never reported
	}
	for _, tt := range tests {
		if got := olderVersion(tt.a, tt.b); got != tt.want {
			t.Errorf("olderVersion(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestCheckSDKVersion(t *testing.T) {
	current := python.TestedSDKVersion()

	if got := checkSDKVersion(pins("otelstarter", "0.0.1")); len(got) != 1 || got[0].Status != Warn ||
		!strings.Contains(got[0].Fix, "otelstarter=="+current) {
		t.Errorf("old pin: got %+v, want a warning whose fix names otelstarter==%s", got, current)
	}
	for _, v := range []string{current, "99.0.0"} {
		if got := checkSDKVersion(pins("otelstarter", v)); len(got) != 0 {
			t.Errorf("pin %s: got %+v, want nothing", v, got)
		}
	}
	if got := checkSDKVersion(pins()); len(got) != 0 {
		t.Errorf("no pin: got %+v, want nothing", got)
	}
}

// The SDK's 0.x version must not be mistaken for an OpenTelemetry beta
// package pinned to a different version.
func TestCheckVersionsIgnoresTheSDK(t *testing.T) {
	results := checkVersions(pins("opentelemetry-distro", "0.66b0", "otelstarter", "0.2.0"))

	if len(results) != 1 || results[0].Status != Pass {
		t.Errorf("checkVersions() = %+v, want a single pass", results)
	}
}
