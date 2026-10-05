// Package python detects the framework and observability-relevant
// dependencies of a Python project by reading its manifest files.
//
// It only reads files. It never runs pip, uv or Python, so analysing an
// untrusted project cannot execute its code.
package python

import (
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/IvyMurage/otelstarter/internal/project"
)

// Manifests are the files whose presence marks a folder as a Python project.
var Manifests = []string{"pyproject.toml", "requirements.txt", "Pipfile", "setup.py"}

// maxManifestSize guards against reading huge files by mistake. Real
// manifests are a few kilobytes.
const maxManifestSize = 1 << 20 // 1 MiB

var frameworks = map[string]string{
	"fastapi": "FastAPI",
	"flask":   "Flask",
	"django":  "Django",
}

// dependencyCategories maps normalised package names to what they are.
// Only list packages whose category is certain: SQLAlchemy is an ORM, but it
// does not tell us which database sits behind it.
var dependencyCategories = map[string]string{
	"psycopg":                "PostgreSQL",
	"psycopg-binary":         "PostgreSQL",
	"psycopg2":               "PostgreSQL",
	"psycopg2-binary":        "PostgreSQL",
	"asyncpg":                "PostgreSQL",
	"pymysql":                "MySQL",
	"mysqlclient":            "MySQL",
	"mysql-connector-python": "MySQL",
	"aiomysql":               "MySQL",
	"redis":                  "Redis",
	"kafka-python":           "Kafka",
	"confluent-kafka":        "Kafka",
	"aiokafka":               "Kafka",
	"pika":                   "RabbitMQ",
	"aio-pika":               "RabbitMQ",
	"requests":               "HTTP client",
	"httpx":                  "HTTP client",
	"aiohttp":                "HTTP client",
	"sqlalchemy":             "ORM",
	"sqlmodel":               "ORM",
	"tortoise-orm":           "ORM",
	"peewee":                 "ORM",
	"celery":                 "Task queue",
}

// Analyze reads the Python manifests in fsys and reports what it found.
func Analyze(fsys fs.FS) (project.Language, error) {
	lang := project.Language{Name: "Python", DependenciesDetected: true}

	var names []string
	usesPoetry := false
	readAny := false

	if data, ok, err := readManifest(fsys, "pyproject.toml"); err != nil {
		return lang, err
	} else if ok {
		deps, poetry, err := parsePyproject(data)
		if err != nil {
			return lang, err
		}
		names = append(names, deps...)
		usesPoetry = poetry
		readAny = true
	}

	if data, ok, err := readManifest(fsys, "requirements.txt"); err != nil {
		return lang, err
	} else if ok {
		names = append(names, parseRequirements(data)...)
		readAny = true
	}

	if data, ok, err := readManifest(fsys, "Pipfile"); err != nil {
		return lang, err
	} else if ok {
		deps, err := parsePipfile(data)
		if err != nil {
			return lang, err
		}
		names = append(names, deps...)
		readAny = true
	}

	if !readAny && fileExists(fsys, "setup.py") {
		lang.Notes = append(lang.Notes,
			"setup.py is not read because it is Python code, and otelstarter never runs project code. "+
				"List your dependencies in pyproject.toml or requirements.txt so they can be detected.")
	}

	lang.PackageManager = packageManager(fsys, usesPoetry)
	lang.Frameworks, lang.Dependencies, lang.OpenTelemetry = classify(names)
	return lang, nil
}

// readManifest returns the file's contents and whether it exists.
func readManifest(fsys fs.FS, name string) ([]byte, bool, error) {
	info, err := fs.Stat(fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("cannot read %s: %w", name, err)
	}
	if info.Size() > maxManifestSize {
		return nil, false, fmt.Errorf("%s is %d bytes, larger than the 1 MiB otelstarter reads. "+
			"Check that it is really a dependency manifest", name, info.Size())
	}
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, false, fmt.Errorf("cannot read %s: %w", name, err)
	}
	return data, true, nil
}

func fileExists(fsys fs.FS, name string) bool {
	_, err := fs.Stat(fsys, name)
	return err == nil
}

// parsePyproject returns dependency names from a PEP 621 [project] table or a
// Poetry [tool.poetry.dependencies] table, and whether Poetry is in use.
// Optional and development dependencies are ignored: they rarely run in
// production, which is what we instrument.
func parsePyproject(data []byte) ([]string, bool, error) {
	// Struct tags (`toml:"..."`) tell the TOML library which key fills each field.
	var doc struct {
		Project struct {
			Dependencies []string `toml:"dependencies"`
		} `toml:"project"`
		Tool struct {
			Poetry *struct {
				Dependencies map[string]any `toml:"dependencies"`
			} `toml:"poetry"`
		} `toml:"tool"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, false, fmt.Errorf("pyproject.toml is not valid TOML: %w. "+
			"Fix the syntax error and run otelstarter analyze again", err)
	}

	var names []string
	for _, req := range doc.Project.Dependencies {
		names = append(names, requirementName(req))
	}

	usesPoetry := doc.Tool.Poetry != nil
	if usesPoetry {
		for name := range doc.Tool.Poetry.Dependencies {
			if name != "python" { // Poetry lists the interpreter version here too
				names = append(names, normalize(name))
			}
		}
	}
	return names, usesPoetry, nil
}

func parsePipfile(data []byte) ([]string, error) {
	var doc struct {
		Packages map[string]any `toml:"packages"`
	}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("Pipfile is not valid TOML: %w. "+
			"Fix the syntax error and run otelstarter analyze again", err)
	}
	var names []string
	for name := range doc.Packages {
		names = append(names, normalize(name))
	}
	return names, nil
}

func parseRequirements(data []byte) []string {
	var names []string
	for _, line := range strings.Split(string(data), "\n") {
		line, _, _ = strings.Cut(line, "#")
		line = strings.TrimSpace(line)
		// Lines starting with "-" are pip options such as -r, -e or --index-url.
		if line == "" || strings.HasPrefix(line, "-") {
			continue
		}
		names = append(names, requirementName(line))
	}
	return names
}

// requirementName extracts the package name from a requirement such as
// "fastapi[all]>=0.110 ; python_version >= '3.9'".
func requirementName(req string) string {
	req = strings.TrimSpace(req)
	if end := strings.IndexAny(req, " \t[<>=!~;@,("); end >= 0 {
		req = req[:end]
	}
	return normalize(req)
}

// normalize applies the PEP 503 rule so "Flask_SQLAlchemy" and
// "flask-sqlalchemy" compare equal: lowercase, and any run of "-", "_" or "."
// becomes a single "-".
func normalize(name string) string {
	var b strings.Builder
	lastWasSeparator := false
	for _, r := range strings.ToLower(name) {
		if r == '-' || r == '_' || r == '.' {
			if !lastWasSeparator {
				b.WriteRune('-')
			}
			lastWasSeparator = true
			continue
		}
		b.WriteRune(r)
		lastWasSeparator = false
	}
	return b.String()
}

// classify sorts package names into frameworks, known dependencies and
// OpenTelemetry packages. Unknown packages are dropped. Results are sorted
// so output is the same on every run.
func classify(names []string) (frameworkNames []string, deps []project.Dependency, otel []string) {
	seen := map[string]bool{}
	for _, name := range names {
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true

		if fw, ok := frameworks[name]; ok {
			frameworkNames = append(frameworkNames, fw)
		}
		if category, ok := dependencyCategories[name]; ok {
			deps = append(deps, project.Dependency{Category: category, Package: name})
		}
		if strings.HasPrefix(name, "opentelemetry-") {
			otel = append(otel, name)
		}
	}

	sort.Strings(frameworkNames)
	sort.Slice(deps, func(i, j int) bool {
		if deps[i].Category != deps[j].Category {
			return deps[i].Category < deps[j].Category
		}
		return deps[i].Package < deps[j].Package
	})
	sort.Strings(otel)
	return frameworkNames, deps, otel
}

// packageManager guesses the tool from lock files. Lock files win over
// manifests because a project can keep a requirements.txt for deployment
// while developing with uv or Poetry.
func packageManager(fsys fs.FS, usesPoetry bool) string {
	switch {
	case fileExists(fsys, "uv.lock"):
		return "uv"
	case fileExists(fsys, "poetry.lock"), usesPoetry:
		return "Poetry"
	case fileExists(fsys, "pdm.lock"):
		return "PDM"
	case fileExists(fsys, "Pipfile"), fileExists(fsys, "Pipfile.lock"):
		return "Pipenv"
	case fileExists(fsys, "requirements.txt"):
		return "pip"
	default:
		return ""
	}
}
