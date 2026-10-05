package setup

import (
	"os"
	"path/filepath"
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

func TestReadEnvFileReadsWhatRenderWrites(t *testing.T) {
	dir := t.TempDir()
	content := renderEnvFile("shop", []project.EnvVar{{Name: "EXTRA", Value: "1", Comment: "Extra."}})
	if err := os.WriteFile(filepath.Join(dir, EnvFileName), content, 0o644); err != nil {
		t.Fatal(err)
	}

	vars, err := ReadEnvFile(dir)
	if err != nil {
		t.Fatalf("ReadEnvFile() error = %v", err)
	}
	want := []string{
		"OTEL_SERVICE_NAME=shop",
		"OTEL_RESOURCE_ATTRIBUTES=deployment.environment.name=local",
		"OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318",
		"OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf",
		"EXTRA=1",
	}
	if strings.Join(vars, "\n") != strings.Join(want, "\n") {
		t.Errorf("ReadEnvFile() =\n%v\nwant\n%v", vars, want)
	}
}

func TestReadEnvFileErrors(t *testing.T) {
	tests := map[string]string{
		"missing file":     "",
		"line without =":   "OTEL_SERVICE_NAME\n",
		"invalid name":     "1BAD=x\n",
		"name with spaces": "MY VAR=x\n",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if content != "" {
				if err := os.WriteFile(filepath.Join(dir, EnvFileName), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := ReadEnvFile(dir); err == nil {
				t.Error("ReadEnvFile() = nil error, want an error")
			}
		})
	}
}
