package python

import (
	"reflect"
	"testing"

	"github.com/IvyMurage/otelstarter/internal/project"
)

func TestMissingRequirements(t *testing.T) {
	lang := project.Language{Instrumentation: []project.Instrumentation{
		{Package: "opentelemetry-distro", Installed: true},
		{Package: "opentelemetry-exporter-otlp-proto-http"},
		{Package: "opentelemetry-instrumentation-fastapi"},
	}}

	got := MissingRequirements(lang)

	want := []string{
		"opentelemetry-exporter-otlp-proto-http==" + sdkVersion, // exporters follow the SDK version
		"opentelemetry-instrumentation-fastapi==" + contribVersion,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MissingRequirements() = %v, want %v", got, want)
	}
}

func TestInstallCommand(t *testing.T) {
	reqs := []string{"a==1", "b==2"}
	tests := map[string]string{
		"pip":    "pip install -r requirements.txt",
		"uv":     "uv add a==1 b==2",
		"Poetry": "poetry add a==1 b==2",
		"PDM":    "pdm add a==1 b==2",
		"Pipenv": "pipenv install a==1 b==2",
		"":       "pip install a==1 b==2",
	}
	for manager, want := range tests {
		if got := InstallCommand(manager, reqs); got != want {
			t.Errorf("InstallCommand(%q) = %q, want %q", manager, got, want)
		}
	}
}

func TestAddToRequirements(t *testing.T) {
	block := func(lines string) string { return blockStart + "\n" + lines + blockEnd + "\n" }

	tests := []struct {
		name     string
		existing string
		want     string
	}{
		{"empty file", "", block("new==1\n")},
		{"appends after a blank line", "fastapi\n", "fastapi\n\n" + block("new==1\n")},
		{"adds missing final newline", "fastapi", "fastapi\n\n" + block("new==1\n")},
		{
			"adds inside an existing block, keeping lines after it",
			"fastapi\n\n" + block("old==1\n") + "redis\n",
			"fastapi\n\n" + block("old==1\nnew==1\n") + "redis\n",
		},
		{
			"block without end marker gets a new block",
			blockStart + "\nold==1\n",
			blockStart + "\nold==1\n\n" + block("new==1\n"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddToRequirements(tt.existing, []string{"new==1"}); got != tt.want {
				t.Errorf("AddToRequirements() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

// Running init twice must not duplicate packages: once the block is in
// requirements.txt, analysis sees the packages as installed.
func TestRequirementsBlockIsReadBackAsInstalled(t *testing.T) {
	content := AddToRequirements("fastapi\n", []string{
		"opentelemetry-distro==" + contribVersion,
		"opentelemetry-instrumentation-fastapi==" + contribVersion,
	})

	lang, err := Analyze(files(map[string]string{"requirements.txt": content}))
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	for _, inst := range lang.Instrumentation {
		wantInstalled := inst.Package == "opentelemetry-distro" || inst.Package == "opentelemetry-instrumentation-fastapi"
		if inst.Installed != wantInstalled {
			t.Errorf("%s Installed = %v, want %v", inst.Package, inst.Installed, wantInstalled)
		}
	}
}
