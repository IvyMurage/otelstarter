// Package node detects the framework and observability-relevant
// dependencies of a Node.js project by reading package.json.
//
// Like the Python adapter it only reads files: it never runs npm or node.
package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/IvyMurage/otelstarter/internal/project"
)

// Manifests are the files whose presence marks a folder as a Node.js project.
var Manifests = []string{"package.json"}

const maxManifestSize = 1 << 20 // 1 MiB

// The zero-code package and the API it needs.
const (
	autoPackage = "@opentelemetry/auto-instrumentations-node"
	apiPackage  = "@opentelemetry/api"
	// RegisterModule is loaded with --require to start instrumentation.
	RegisterModule = autoPackage + "/register"
)

// alwaysEnabled are switched on for every app: incoming and outgoing HTTP
// (node:http, and undici for fetch) and runtime metrics.
var alwaysEnabled = []string{"http", "undici", "runtime-node"}

// knownPackage describes an npm package that matters for observability.
type knownPackage struct {
	framework string // display name when it is a web framework
	category  string // dependency category otherwise
	// instrumentation is the short name of the bundled instrumentation
	// (OTEL_NODE_ENABLED_INSTRUMENTATIONS uses these). Empty when
	// @opentelemetry/auto-instrumentations-node has none for it.
	instrumentation string
}

// knownPackages is checked against the instrumentations bundled in
// @opentelemetry/auto-instrumentations-node 0.80.0.
var knownPackages = map[string]knownPackage{
	"express":        {framework: "Express", instrumentation: "express"},
	"@nestjs/core":   {framework: "NestJS", instrumentation: "nestjs-core"},
	"koa":            {framework: "Koa", instrumentation: "koa"},
	"@hapi/hapi":     {framework: "hapi", instrumentation: "hapi"},
	"fastify":        {framework: "Fastify"}, // instrumented separately by @fastify/otel
	"pg":             {category: "PostgreSQL", instrumentation: "pg"},
	"mysql":          {category: "MySQL", instrumentation: "mysql"},
	"mysql2":         {category: "MySQL", instrumentation: "mysql2"},
	"redis":          {category: "Redis", instrumentation: "redis"},
	"ioredis":        {category: "Redis", instrumentation: "ioredis"},
	"mongodb":        {category: "MongoDB", instrumentation: "mongodb"},
	"mongoose":       {category: "MongoDB", instrumentation: "mongoose"},
	"kafkajs":        {category: "Kafka", instrumentation: "kafkajs"},
	"amqplib":        {category: "RabbitMQ", instrumentation: "amqplib"},
	"knex":           {category: "ORM", instrumentation: "knex"},
	"@prisma/client": {category: "ORM"},         // instrumented separately by @prisma/instrumentation
	"axios":          {category: "HTTP client"}, // covered by the http instrumentation
	"graphql":        {category: "GraphQL", instrumentation: "graphql"},
	"pino":           {category: "Logging", instrumentation: "pino"},
	"winston":        {category: "Logging", instrumentation: "winston"},
	"bunyan":         {category: "Logging", instrumentation: "bunyan"},
}

// packageJSON is the part of package.json otelstarter reads.
type packageJSON struct {
	Name            string            `json:"name"`
	Type            string            `json:"type"`
	Main            string            `json:"main"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// Analyze reads package.json in fsys and reports what it found.
func Analyze(fsys fs.FS) (project.Language, error) {
	lang := project.Language{Name: "Node.js", DependenciesDetected: true}

	data, err := readManifest(fsys, "package.json")
	if err != nil {
		return lang, err
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return lang, fmt.Errorf("package.json is not valid JSON: %w. Fix the syntax error and run otelstarter again", err)
	}

	lang.AppName = pkg.Name
	lang.PackageManager = packageManager(fsys)
	// Only runtime dependencies: devDependencies (test tools, bundlers)
	// rarely run in production, which is what we instrument.
	names := slices.Sorted(maps.Keys(pkg.Dependencies))

	enabled := slices.Clone(alwaysEnabled)
	hasLogger := false
	for _, name := range names {
		if strings.HasPrefix(name, "@opentelemetry/") {
			lang.OpenTelemetry = append(lang.OpenTelemetry, name)
		}
		known, ok := knownPackages[name]
		if !ok {
			continue
		}
		if known.framework != "" {
			lang.Frameworks = append(lang.Frameworks, known.framework)
		} else {
			lang.Dependencies = append(lang.Dependencies, project.Dependency{Category: known.category, Package: name})
		}
		if known.category == "Logging" {
			hasLogger = true
		}
		if known.instrumentation != "" && !slices.Contains(enabled, known.instrumentation) {
			enabled = append(enabled, known.instrumentation)
		}
		lang.Notes = append(lang.Notes, gapNote(name)...)
	}
	sort.Slice(lang.Dependencies, func(i, j int) bool {
		if lang.Dependencies[i].Category != lang.Dependencies[j].Category {
			return lang.Dependencies[i].Category < lang.Dependencies[j].Category
		}
		return lang.Dependencies[i].Package < lang.Dependencies[j].Package
	})
	lang.Enabled = enabled

	lang.Instrumentation = []project.Instrumentation{
		{Package: apiPackage, Purpose: "OpenTelemetry API", Installed: has(pkg.Dependencies, apiPackage)},
		{Package: autoPackage, Purpose: "Zero-code instrumentation, loaded with --require",
			Installed: has(pkg.Dependencies, autoPackage)},
	}
	lang.Start = startCommand(pkg, lang.PackageManager)
	lang.Activation = fmt.Sprintf(`NODE_OPTIONS="--require %s" %s`, RegisterModule, lang.Start)

	if !hasLogger {
		lang.Notes = append(lang.Notes, "No logging library found (pino, winston or bunyan). OpenTelemetry for Node "+
			"does not export console.log, so your logs will not reach the backend. Use pino or winston to send them, "+
			"with trace IDs.")
	}
	if pkg.Type == "module" {
		lang.Notes = append(lang.Notes, `This is an ES-module app ("type": "module"). otelstarter does not support `+
			"ES modules yet: --require only instruments CommonJS, so frameworks and libraries loaded with import "+
			"are not instrumented. Support is planned.")
	}
	return lang, nil
}

// gapNote explains libraries the bundled package cannot instrument.
func gapNote(name string) []string {
	switch name {
	case "fastify":
		return []string{"Fastify is not instrumented by @opentelemetry/auto-instrumentations-node; incoming " +
			"requests are still traced through the http instrumentation, but without route names. " +
			"Fastify's own plugin, @fastify/otel, adds them."}
	case "@prisma/client":
		return []string{"Prisma queries are not instrumented by @opentelemetry/auto-instrumentations-node; " +
			"Prisma's own @prisma/instrumentation adds them."}
	}
	return nil
}

// startCommand guesses how the app is started, to show where the
// instrumentation goes. A "start" script wins; otherwise node runs "main".
func startCommand(pkg packageJSON, packageManager string) string {
	if _, ok := pkg.Scripts["start"]; ok {
		switch packageManager {
		case "yarn", "pnpm", "bun":
			return packageManager + " start"
		default:
			return "npm start"
		}
	}
	if pkg.Main != "" {
		return "node " + pkg.Main
	}
	return "node index.js"
}

// packageManager reads the lock file.
func packageManager(fsys fs.FS) string {
	for _, lock := range []struct{ file, manager string }{
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"bun.lock", "bun"},
		{"bun.lockb", "bun"},
		{"package-lock.json", "npm"},
	} {
		if _, err := fs.Stat(fsys, lock.file); err == nil {
			return lock.manager
		}
	}
	return "npm"
}

func readManifest(fsys fs.FS, name string) ([]byte, error) {
	info, err := fs.Stat(fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%s not found", name)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", name, err)
	}
	if info.Size() > maxManifestSize {
		return nil, fmt.Errorf("%s is %d bytes, larger than the 1 MiB otelstarter reads", name, info.Size())
	}
	return fs.ReadFile(fsys, name)
}

func has(deps map[string]string, name string) bool {
	_, ok := deps[name]
	return ok
}
