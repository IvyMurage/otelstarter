// Package project describes an analysed application in language-neutral terms.
//
// Detection and language adapters fill these types in; the CLI and later
// configuration steps read them. Keeping the model in its own package lets
// adapters use it without importing the detector that calls them.
package project

// Project is everything otelstarter learned about one application folder.
type Project struct {
	Path      string
	Languages []Language
}

// Language is one language ecosystem found in the project. A repository can
// contain more than one, for example a Python API with a package.json for
// frontend tooling.
type Language struct {
	Name           string   // "Python", "Node.js" or "Go"
	AppName        string   // the application's name from its manifest, if declared
	Manifests      []string // files that identified the language, e.g. "pyproject.toml"
	PackageManager string   // empty when unknown
	Frameworks     []string
	Dependencies   []Dependency
	OpenTelemetry  []string // OpenTelemetry packages already installed
	Pins           []Pin    // exact versions declared for OpenTelemetry packages and the otelstarter SDK

	// Instrumentation lists the OpenTelemetry packages the project should
	// have, and Activation shows how to switch them on, e.g.
	// "opentelemetry-instrument uvicorn main:app".
	Instrumentation []Instrumentation
	Activation      string

	// Start is the app's own start command, without instrumentation, e.g.
	// "uvicorn main:app" or "npm start". init prints "otelstarter run -- " + Start.
	Start string

	// Enabled lists the instrumentations to switch on, for languages where
	// one package bundles many (Node.js), e.g. "express", "pg".
	Enabled []string

	// DependenciesDetected is false when no adapter supports this language
	// yet, so "found nothing" is never confused with "did not look".
	DependenciesDetected bool

	// Notes explain gaps the user can fix, such as a manifest that cannot
	// be read.
	Notes []string
}

// Dependency is an installed package that matters for observability.
type Dependency struct {
	Category string // what it is, e.g. "PostgreSQL" or "HTTP client"
	Package  string // the package name as written in the manifest, normalised
}

// Instrumentation is an OpenTelemetry package recommended for the project.
type Instrumentation struct {
	Package   string // e.g. "opentelemetry-instrumentation-fastapi"
	Purpose   string // what it gives the user, e.g. "Database queries (asyncpg)"
	Installed bool
}

// EnvVar is one setting written to the generated otel.env file.
type EnvVar struct {
	Name    string
	Value   string
	Comment string // one line shown above the setting, explaining it
}

// Pin is a package declared with an exact version, such as
// opentelemetry-distro==0.66b0.
type Pin struct {
	Package string
	Version string
}
