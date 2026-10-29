package python

import (
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IvyMurage/otelstarter/internal/project"
)

// fstest.MapFS is an in-memory file system: file name -> contents. It keeps
// these tests fast and free of temporary folders.
func files(m map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for name, content := range m {
		fsys[name] = &fstest.MapFile{Data: []byte(content)}
	}
	return fsys
}

func TestAnalyze(t *testing.T) {
	tests := []struct {
		name               string
		files              map[string]string
		wantPackageManager string
		wantFrameworks     []string
		wantDeps           []project.Dependency
		wantOTel           []string
	}{
		{
			name: "requirements.txt with versions, extras, markers and comments",
			files: map[string]string{"requirements.txt": `
# web
FastAPI[all]>=0.110 ; python_version >= "3.9"
psycopg2-binary==2.9.9   # database
-r dev-requirements.txt
--index-url https://example.com/simple
Redis~=5.0
`},
			wantPackageManager: "pip",
			wantFrameworks:     []string{"FastAPI"},
			wantDeps: []project.Dependency{
				{Category: "PostgreSQL", Package: "psycopg2-binary"},
				{Category: "Redis", Package: "redis"},
			},
		},
		{
			name: "PEP 621 pyproject with uv lock file",
			files: map[string]string{
				"pyproject.toml": `
[project]
name = "shop"
dependencies = ["flask>=3", "SQLAlchemy", "requests", "opentelemetry-sdk"]

[project.optional-dependencies]
dev = ["httpx"]
`,
				"uv.lock": "",
			},
			wantPackageManager: "uv",
			wantFrameworks:     []string{"Flask"},
			wantDeps: []project.Dependency{
				{Category: "HTTP client", Package: "requests"},
				{Category: "ORM", Package: "sqlalchemy"},
			},
			wantOTel: []string{"opentelemetry-sdk"},
		},
		{
			name: "Poetry pyproject",
			files: map[string]string{"pyproject.toml": `
[tool.poetry.dependencies]
python = "^3.11"
Django = "^5.0"
celery = { version = "^5.3", extras = ["redis"] }
`},
			wantPackageManager: "Poetry",
			wantFrameworks:     []string{"Django"},
			wantDeps:           []project.Dependency{{Category: "Task queue", Package: "celery"}},
		},
		{
			name: "Pipfile",
			files: map[string]string{"Pipfile": `
[packages]
aiohttp = "*"
pika = "*"
`},
			wantPackageManager: "Pipenv",
			wantDeps: []project.Dependency{
				{Category: "HTTP client", Package: "aiohttp"},
				{Category: "RabbitMQ", Package: "pika"},
			},
		},
		{
			name: "same package in two manifests is listed once",
			files: map[string]string{
				"pyproject.toml":   "[project]\ndependencies = [\"redis\"]\n",
				"requirements.txt": "redis==5.0\n",
			},
			wantPackageManager: "pip",
			wantDeps:           []project.Dependency{{Category: "Redis", Package: "redis"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Analyze(files(tt.files))
			if err != nil {
				t.Fatalf("Analyze() error = %v", err)
			}
			if got.PackageManager != tt.wantPackageManager {
				t.Errorf("PackageManager = %q, want %q", got.PackageManager, tt.wantPackageManager)
			}
			// reflect.DeepEqual compares slices element by element.
			if !reflect.DeepEqual(got.Frameworks, tt.wantFrameworks) {
				t.Errorf("Frameworks = %v, want %v", got.Frameworks, tt.wantFrameworks)
			}
			if !reflect.DeepEqual(got.Dependencies, tt.wantDeps) {
				t.Errorf("Dependencies = %v, want %v", got.Dependencies, tt.wantDeps)
			}
			if !reflect.DeepEqual(got.OpenTelemetry, tt.wantOTel) {
				t.Errorf("OpenTelemetry = %v, want %v", got.OpenTelemetry, tt.wantOTel)
			}
		})
	}
}

func TestAnalyzePymongo(t *testing.T) {
	got, err := Analyze(files(map[string]string{
		"requirements.txt": "pymongo==4.10.0\n",
	}))
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if want := []project.Dependency{{Category: "MongoDB", Package: "pymongo"}}; !reflect.DeepEqual(got.Dependencies, want) {
		t.Errorf("Dependencies = %v, want %v", got.Dependencies, want)
	}
	const instrumentation = "opentelemetry-instrumentation-pymongo"
	var found bool
	for _, rec := range got.Instrumentation {
		if rec.Package == instrumentation {
			found = true
			if rec.Purpose != "Database queries (pymongo)" {
				t.Errorf("Purpose = %q, want MongoDB database queries", rec.Purpose)
			}
			if rec.Installed {
				t.Error("new instrumentation is marked installed")
			}
		}
	}
	if !found {
		t.Errorf("recommended instrumentation does not include %q: %v", instrumentation, got.Instrumentation)
	}
	if want := instrumentation + "==0.66b0"; !hasNote(MissingRequirements(got), want) {
		t.Errorf("MissingRequirements() does not include %q", want)
	}
}

func TestAnalyzeSetupPyOnlyExplainsGap(t *testing.T) {
	got, err := Analyze(files(map[string]string{"setup.py": "from setuptools import setup\n"}))
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if !hasNote(got.Notes, "setup.py is not read") {
		t.Errorf("Notes = %q, want a note explaining setup.py is not read", got.Notes)
	}
}

func TestAnalyzeErrors(t *testing.T) {
	tests := []struct {
		name    string
		fsys    fstest.MapFS
		wantErr string
	}{
		{
			name:    "invalid pyproject.toml",
			fsys:    files(map[string]string{"pyproject.toml": "[project\n"}),
			wantErr: "pyproject.toml is not valid TOML",
		},
		{
			name:    "invalid Pipfile",
			fsys:    files(map[string]string{"Pipfile": "packages = = 1\n"}),
			wantErr: "Pipfile is not valid TOML",
		},
		{
			name: "oversized manifest",
			fsys: fstest.MapFS{"requirements.txt": &fstest.MapFile{
				Data: []byte(strings.Repeat("a", maxManifestSize+1)),
			}},
			wantErr: "larger than the 1 MiB",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Analyze(tt.fsys)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Analyze() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := map[string]string{
		"Flask_SQLAlchemy":       "flask-sqlalchemy",
		"zope.interface":         "zope-interface",
		"some__weird--.Name":     "some-weird-name",
		"mysql-connector-python": "mysql-connector-python",
	}
	for in, want := range tests {
		if got := normalize(in); got != want {
			t.Errorf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func hasNote(notes []string, want string) bool {
	for _, n := range notes {
		if strings.Contains(n, want) {
			return true
		}
	}
	return false
}

func TestAnalyzeReadsAppName(t *testing.T) {
	tests := map[string]string{
		"PEP 621": "[project]\nname = \"shop-api\"\n",
		"Poetry":  "[tool.poetry]\nname = \"shop-api\"\n",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			lang, err := Analyze(files(map[string]string{"pyproject.toml": content}))
			if err != nil {
				t.Fatalf("Analyze() error = %v", err)
			}
			if lang.AppName != "shop-api" {
				t.Errorf("AppName = %q, want %q", lang.AppName, "shop-api")
			}
		})
	}
}

func TestParseRequirement(t *testing.T) {
	tests := []struct {
		in   string
		want requirement
	}{
		{"opentelemetry-distro==0.66b0", requirement{"opentelemetry-distro", "0.66b0"}},
		{"Flask[async] == 3.0.3 ; python_version >= '3.10'", requirement{"flask", "3.0.3"}},
		{"redis==5.0.1,<6", requirement{"redis", "5.0.1"}},
		{"httpx>=0.27", requirement{"httpx", ""}},
		{"httpx==0.27.*", requirement{"httpx", ""}},  // a wildcard is a range, not a pin
		{"httpx===0.27.0", requirement{"httpx", ""}}, // arbitrary equality is not a normal pin
		{"fastapi", requirement{"fastapi", ""}},
		{"pika *", requirement{"pika", ""}}, // how Pipfile's "*" reaches the parser
	}
	for _, tt := range tests {
		if got := parseRequirement(tt.in); got != tt.want {
			t.Errorf("parseRequirement(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestAnalyzeRecordsOpenTelemetryPins(t *testing.T) {
	lang, err := Analyze(files(map[string]string{
		"requirements.txt": "fastapi==0.115.0\nopentelemetry-distro==0.66b0\nopentelemetry-sdk>=1.40\n",
		"Pipfile": "[packages]\nopentelemetry-instrumentation-fastapi = \"==0.60b0\"\n" +
			"opentelemetry-exporter-otlp-proto-http = {version = \"==1.45.0\"}\n",
		"pyproject.toml": "[project]\ndependencies = [\"opentelemetry-distro==0.66b0\", \"otelstarter==0.1.0\"]\n", // distro pin twice: listed once
	}))
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	want := []project.Pin{
		{Package: "opentelemetry-distro", Version: "0.66b0"},
		{Package: "opentelemetry-exporter-otlp-proto-http", Version: "1.45.0"},
		{Package: "opentelemetry-instrumentation-fastapi", Version: "0.60b0"},
		{Package: "otelstarter", Version: "0.1.0"},
	}
	if !reflect.DeepEqual(lang.Pins, want) {
		t.Errorf("Pins = %+v, want %+v", lang.Pins, want)
	}
}
