package setup

import (
	"regexp"
	"strings"
	"testing"

	"github.com/IvyMurage/otelstarter/internal/project"
)

func TestServiceName(t *testing.T) {
	tests := []struct {
		candidates []string
		want       string
	}{
		{[]string{"shop-api"}, "shop-api"},
		{[]string{"Shop API"}, "shop-api"},
		{[]string{"", "my_folder"}, "my_folder"},
		{[]string{"***", "folder"}, "folder"},
		{[]string{"", ""}, "app"},
		{[]string{"evil\nOTEL_EXPORTER_OTLP_ENDPOINT=http://attacker"}, "evil-otel_exporter_otlp_endpoint-http-attacker"},
	}
	for _, tt := range tests {
		if got := serviceName(tt.candidates...); got != tt.want {
			t.Errorf("serviceName(%q) = %q, want %q", tt.candidates, got, tt.want)
		}
	}
}

func TestRenderEnvFile(t *testing.T) {
	content := string(renderEnvFile("shop", []project.EnvVar{{Name: "EXTRA", Value: "1", Comment: "Extra."}}))

	for _, want := range []string{
		"OTEL_SERVICE_NAME=shop\n",
		"OTEL_RESOURCE_ATTRIBUTES=deployment.environment.name=local\n",
		"OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318\n",
		"OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf\n",
		"# Extra.\nEXTRA=1\n",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("otel.env is missing %q", want)
		}
	}

	// Every line must be a comment, blank, or NAME=value with no spaces or
	// quotes, so `set -a && . ./otel.env` in any POSIX shell reads it the same way.
	valid := regexp.MustCompile(`^(#.*|[A-Z_]+=[^\s'"$` + "`" + `]+)?$`)
	for _, line := range strings.Split(strings.TrimSuffix(content, "\n"), "\n") {
		if !valid.MatchString(line) {
			t.Errorf("line %q is not a comment, blank, or plain NAME=value", line)
		}
	}
}
