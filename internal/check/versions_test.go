package check

import (
	"strings"
	"testing"

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
