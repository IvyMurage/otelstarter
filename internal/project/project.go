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
	Manifests      []string // files that identified the language, e.g. "pyproject.toml"
	PackageManager string   // empty when unknown
	Frameworks     []string
	Dependencies   []Dependency
	OpenTelemetry  []string // OpenTelemetry packages already installed

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
