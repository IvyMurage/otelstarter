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
