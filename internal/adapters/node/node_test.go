package node

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IvyMurage/otelstarter/internal/project"
)

func files(m map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for name, content := range m {
		fsys[name] = &fstest.MapFile{Data: []byte(content)}
	}
	return fsys
}

func analyze(t *testing.T, m map[string]string) project.Language {
	t.Helper()
	lang, err := Analyze(files(m))
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	return lang
}

func TestAnalyzeExpressApp(t *testing.T) {
	lang := analyze(t, map[string]string{
		"package.json": `{
			"name": "shop",
			"scripts": {"start": "node server.js"},
			"dependencies": {"express": "^4", "pg": "^8", "ioredis": "^5", "pino": "^9", "lodash": "^4"},
			"devDependencies": {"jest": "^29", "mysql2": "^3"}
		}`,
		"pnpm-lock.yaml": "",
	})

	if lang.AppName != "shop" || lang.PackageManager != "pnpm" {
		t.Errorf("AppName, PackageManager = %q, %q, want shop, pnpm", lang.AppName, lang.PackageManager)
	}
	if !reflect.DeepEqual(lang.Frameworks, []string{"Express"}) {
		t.Errorf("Frameworks = %v, want [Express]", lang.Frameworks)
	}
	wantDeps := []project.Dependency{
		{Category: "Logging", Package: "pino"},
		{Category: "PostgreSQL", Package: "pg"},
		{Category: "Redis", Package: "ioredis"},
	}
	if !reflect.DeepEqual(lang.Dependencies, wantDeps) {
		t.Errorf("Dependencies = %v, want %v (devDependencies such as mysql2 are ignored)", lang.Dependencies, wantDeps)
	}
	wantEnabled := []string{"http", "undici", "runtime-node", "express", "ioredis", "pg", "pino"}
	if !reflect.DeepEqual(lang.Enabled, wantEnabled) {
		t.Errorf("Enabled = %v, want %v", lang.Enabled, wantEnabled)
	}
	if lang.Activation != `NODE_OPTIONS="--require otelstarter/register" pnpm start` {
		t.Errorf("Activation = %q", lang.Activation)
	}
	if len(lang.Notes) != 0 {
		t.Errorf("Notes = %q, want none: a logger is present and the app is CommonJS", lang.Notes)
	}
	var pkgs []string
	for _, inst := range lang.Instrumentation {
		pkgs = append(pkgs, inst.Package)
	}
	// One package per library; pino is part of the SDK.
	wantPkgs := []string{"otelstarter", "@opentelemetry/instrumentation-express",
		"@opentelemetry/instrumentation-ioredis", "@opentelemetry/instrumentation-pg"}
	if !reflect.DeepEqual(pkgs, wantPkgs) {
		t.Errorf("Instrumentation = %v, want %v", pkgs, wantPkgs)
	}
}

func TestAnalyzeWinstonNeedsTransport(t *testing.T) {
	lang := analyze(t, map[string]string{"package.json": `{"dependencies": {"winston": "^3"}}`})
	var pkgs []string
	for _, inst := range lang.Instrumentation {
		pkgs = append(pkgs, inst.Package)
	}
	want := []string{"otelstarter", "@opentelemetry/instrumentation-winston", "@opentelemetry/winston-transport"}
	if !reflect.DeepEqual(pkgs, want) {
		t.Errorf("Instrumentation = %v, want %v", pkgs, want)
	}
}

func TestAnalyzeMarksInstalledPackages(t *testing.T) {
	lang := analyze(t, map[string]string{"package.json": `{"dependencies": {
		"otelstarter": "0.1.1", "@opentelemetry/instrumentation-express": "0.71.0", "express": "^4", "pg": "^8"}}`})

	installed := map[string]bool{}
	for _, inst := range lang.Instrumentation {
		installed[inst.Package] = inst.Installed
	}
	want := map[string]bool{"otelstarter": true, "@opentelemetry/instrumentation-express": true,
		"@opentelemetry/instrumentation-pg": false}
	if !reflect.DeepEqual(installed, want) {
		t.Errorf("Installed = %v, want %v", installed, want)
	}
	if !reflect.DeepEqual(lang.OpenTelemetry, []string{"@opentelemetry/instrumentation-express"}) {
		t.Errorf("OpenTelemetry = %v", lang.OpenTelemetry)
	}
}

func TestAnalyzeNotes(t *testing.T) {
	tests := []struct {
		name     string
		pkg      string
		wantNote string
	}{
		{"no logging library", `{"dependencies": {"express": "^4"}}`, "does not export console.log"},
		{"Fastify", `{"dependencies": {"fastify": "^5", "pino": "^9"}}`, "@fastify/otel"},
		{"Prisma", `{"dependencies": {"@prisma/client": "^5", "pino": "^9"}}`, "@prisma/instrumentation"},
		{"bundle from an earlier otelstarter", `{"dependencies": {"@opentelemetry/auto-instrumentations-node": "0.80.0", "pino": "^9"}}`,
			"otelstarter now uses its own SDK"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lang := analyze(t, map[string]string{"package.json": tt.pkg})
			if !strings.Contains(strings.Join(lang.Notes, "\n"), tt.wantNote) {
				t.Errorf("Notes = %q, want one containing %q", lang.Notes, tt.wantNote)
			}
		})
	}
}

func TestAnalyzeESModuleApp(t *testing.T) {
	lang := analyze(t, map[string]string{"package.json": `{"type": "module", "dependencies": {"express": "^4", "pino": "^9"}}`})
	if !lang.ESModule {
		t.Error("ESModule = false for \"type\": \"module\"")
	}
	if len(lang.Notes) != 0 {
		t.Errorf("Notes = %q, want none: ES modules are supported", lang.Notes)
	}
	if analyze(t, map[string]string{"package.json": `{"dependencies": {"pino": "^9"}}`}).ESModule {
		t.Error("ESModule = true without \"type\": \"module\"")
	}
}

func TestStartCommand(t *testing.T) {
	tests := []struct {
		pkg     packageJSON
		manager string
		want    string
	}{
		{packageJSON{Scripts: map[string]string{"start": "x"}}, "npm", "npm start"},
		{packageJSON{Scripts: map[string]string{"start": "x"}}, "yarn", "yarn start"},
		{packageJSON{Main: "src/server.js"}, "npm", "node src/server.js"},
		{packageJSON{}, "npm", "node index.js"},
	}
	for _, tt := range tests {
		if got := startCommand(tt.pkg, tt.manager); got != tt.want {
			t.Errorf("startCommand(%+v, %q) = %q, want %q", tt.pkg, tt.manager, got, tt.want)
		}
	}
}

func TestPackageManagerFromLockFile(t *testing.T) {
	tests := map[string]string{
		"package-lock.json": "npm",
		"yarn.lock":         "yarn",
		"pnpm-lock.yaml":    "pnpm",
		"bun.lock":          "bun",
		"":                  "npm", // no lock file: npm, the default
	}
	for lock, want := range tests {
		m := map[string]string{"package.json": "{}"}
		if lock != "" {
			m[lock] = ""
		}
		if got := analyze(t, m).PackageManager; got != want {
			t.Errorf("lock file %q: PackageManager = %q, want %q", lock, got, want)
		}
	}
}

func TestAnalyzeInvalidPackageJSON(t *testing.T) {
	_, err := Analyze(files(map[string]string{"package.json": "{not json"}))
	if err == nil || !strings.Contains(err.Error(), "package.json is not valid JSON") {
		t.Errorf("Analyze() error = %v, want an invalid JSON error", err)
	}
}

// Every instrumentation init can recommend must have a tested version, or
// the install command would end in "@" with no version.
func TestEveryRecommendedPackageIsPinned(t *testing.T) {
	for name, known := range knownPackages {
		if known.instrumentation == "" || slices.Contains(inSDK, known.instrumentation) {
			continue
		}
		for _, p := range append([]string{instrumentationPackage(known.instrumentation)}, companions[known.instrumentation]...) {
			if pins[p] == "" {
				t.Errorf("%s: %s has no pinned version in setup.go", name, p)
			}
		}
	}
}
