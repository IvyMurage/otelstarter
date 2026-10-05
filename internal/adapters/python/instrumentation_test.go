package python

import (
	"reflect"
	"strings"
	"testing"

	"github.com/IvyMurage/otelstarter/internal/project"
)

func TestRecommend(t *testing.T) {
	names := []string{"fastapi", "asyncpg", "sqlalchemy", "sqlmodel", "aiomysql", "peewee", "numpy", "asyncpg"}
	installed := []string{"opentelemetry-distro", "opentelemetry-instrumentation-asyncpg"}

	got, notes := recommend(names, installed)

	want := []project.Instrumentation{
		{Package: "opentelemetry-distro", Purpose: "SDK and the opentelemetry-instrument launcher", Installed: true},
		{Package: "opentelemetry-exporter-otlp-proto-http", Purpose: "Sends telemetry to an OpenTelemetry Collector over HTTP"},
		{Package: "opentelemetry-instrumentation-logging", Purpose: "Adds trace and span IDs to log records"},
		{Package: "opentelemetry-instrumentation-system-metrics", Purpose: "Runtime metrics: CPU, memory, garbage collection"},
		{Package: "opentelemetry-instrumentation-asyncpg", Purpose: "Database queries (asyncpg)", Installed: true},
		{Package: "opentelemetry-instrumentation-fastapi", Purpose: "Incoming HTTP requests: traces and request metrics (fastapi)"},
		{Package: "opentelemetry-instrumentation-sqlalchemy", Purpose: "ORM queries (sqlalchemy, sqlmodel)"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("recommend() =\n%v\nwant\n%v", got, want)
	}

	for _, pkg := range []string{"aiomysql", "peewee"} {
		if !hasNote(notes, pkg+" has no official OpenTelemetry instrumentation") {
			t.Errorf("notes = %q, want a note about %s", notes, pkg)
		}
	}
}

// recommend copies basePackages before marking them installed. Without the
// copy, one call would leak Installed=true into the next.
func TestRecommendDoesNotChangeBasePackages(t *testing.T) {
	recommend(nil, []string{"opentelemetry-distro"})
	got, _ := recommend(nil, nil)

	if got[0].Installed {
		t.Error("opentelemetry-distro marked installed in a project that does not have it")
	}
}

func TestStartCommandAndNotes(t *testing.T) {
	tests := []struct {
		frameworks     []string
		wantActivation string
		wantFlaskNote  bool
	}{
		{[]string{"FastAPI"}, "opentelemetry-instrument uvicorn main:app", false},
		{[]string{"Flask"}, "opentelemetry-instrument flask run", true},
		{[]string{"Django"}, "opentelemetry-instrument python manage.py runserver", false},
		{nil, "opentelemetry-instrument python main.py", false},
	}
	for _, tt := range tests {
		t.Run(tt.wantActivation, func(t *testing.T) {
			if got := "opentelemetry-instrument " + exampleStartCommand(tt.frameworks); got != tt.wantActivation {
				t.Errorf("activation = %q, want %q", got, tt.wantActivation)
			}
			notes := runNotes(tt.frameworks)
			if got := hasNote(notes, "use_reloader=False"); got != tt.wantFlaskNote {
				t.Errorf("Flask reloader note present = %v, want %v", got, tt.wantFlaskNote)
			}
			if !hasNote(notes, "Python 3.10 or newer") {
				t.Errorf("notes = %q, want the Python version note", notes)
			}
		})
	}
}

// TestKnownPackagesTable guards the lookup table against typos when rows
// are added.
func TestKnownPackagesTable(t *testing.T) {
	for name, pkg := range knownPackages {
		if name != normalize(name) {
			t.Errorf("%q is not normalised; use %q", name, normalize(name))
		}
		if (pkg.framework == "") == (pkg.category == "") {
			t.Errorf("%s: set exactly one of framework or category", name)
		}
		if pkg.category != "" && categoryPurpose[pkg.category] == "" {
			t.Errorf("%s: category %q has no entry in categoryPurpose", name, pkg.category)
		}
		if pkg.instrumentation != "" && !strings.HasPrefix(pkg.instrumentation, "opentelemetry-instrumentation-") {
			t.Errorf("%s: %q does not look like an OpenTelemetry instrumentation package", name, pkg.instrumentation)
		}
	}
}
