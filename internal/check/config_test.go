package check

import (
	"strings"
	"testing"
)

func TestCheckProtocol(t *testing.T) {
	tests := map[string]Status{
		"http/protobuf": Pass,
		"":              Fail, // the SDK would fall back to gRPC, which is not installed
		"grpc":          Fail,
	}
	for protocol, want := range tests {
		env := map[string]string{}
		if protocol != "" {
			env["OTEL_EXPORTER_OTLP_PROTOCOL"] = protocol
		}
		if got := checkProtocol(env).Status; got != want {
			t.Errorf("checkProtocol(%q) = %v, want %v", protocol, got, want)
		}
	}
}

func TestCheckEndpointURL(t *testing.T) {
	tests := []struct {
		endpoint  string
		want      Status
		wantTitle string
	}{
		{"", Pass, "Endpoint: http://localhost:4318"}, // the SDK default
		{"http://collector:4318", Pass, ""},
		{"https://otlp.example.com", Pass, ""},
		{"localhost:4318", Fail, "not a valid URL"},
		{"ftp://collector:4318", Fail, "not a valid URL"},
		{"http://localhost:4318/v1/traces", Fail, "includes a signal path"},
	}
	for _, tt := range tests {
		env := map[string]string{}
		if tt.endpoint != "" {
			env["OTEL_EXPORTER_OTLP_ENDPOINT"] = tt.endpoint
		}
		got := checkEndpointURL(env)
		if got.Status != tt.want || !strings.Contains(got.Title, tt.wantTitle) {
			t.Errorf("checkEndpointURL(%q) = %v %q, want %v containing %q", tt.endpoint, got.Status, got.Title, tt.want, tt.wantTitle)
		}
	}
}

func TestCheckServiceName(t *testing.T) {
	if got := checkServiceName(map[string]string{}).Status; got != Fail {
		t.Errorf("missing service name = %v, want Fail", got)
	}
	if got := checkServiceName(map[string]string{"OTEL_SERVICE_NAME": "shop"}).Status; got != Pass {
		t.Errorf("service name set = %v, want Pass", got)
	}
}

func TestCheckDeprecated(t *testing.T) {
	if got := checkDeprecated(map[string]string{"OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED": "true"}); len(got) != 1 || got[0].Status != Warn {
		t.Errorf("deprecated variable = %v, want one warning", got)
	}
	if got := checkDeprecated(map[string]string{"OTEL_SERVICE_NAME": "shop"}); len(got) != 0 {
		t.Errorf("no deprecated variable = %v, want no results", got)
	}
}

func TestIsSecret(t *testing.T) {
	tests := []struct {
		name, value string
		want        bool
	}{
		{"OTEL_EXPORTER_OTLP_HEADERS", "Authorization=Bearer abc", true},
		{"OTEL_EXPORTER_OTLP_TRACES_HEADERS", "x-api-key=abc", true},
		{"OTEL_EXPORTER_OTLP_HEADERS", "x-tenant=team-a", false},
		{"HONEYCOMB_API_KEY", "abc", true},
		{"db_password", "abc", true},
		{"GITHUB_TOKEN", "abc", true},
		{"OTEL_SERVICE_NAME", "token-service", false}, // the value alone does not make a name secret
		{"OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318", false},
	}
	for _, tt := range tests {
		if got := IsSecret(tt.name, tt.value); got != tt.want {
			t.Errorf("IsSecret(%q, %q) = %v, want %v", tt.name, tt.value, got, tt.want)
		}
	}
}

func TestCheckSecretsNeverPrintsValues(t *testing.T) {
	results := checkSecrets(map[string]string{
		"OTEL_EXPORTER_OTLP_HEADERS": "Authorization=Bearer sk-live-123",
		"OTEL_SERVICE_NAME":          "shop",
	})

	if len(results) != 1 || results[0].Status != Warn {
		t.Fatalf("checkSecrets() = %v, want one warning", results)
	}
	r := results[0]
	if strings.Contains(r.Title+r.Why+r.Fix, "sk-live-123") {
		t.Errorf("result %+v contains the secret value", r)
	}
}
