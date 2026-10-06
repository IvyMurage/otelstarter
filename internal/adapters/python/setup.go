package python

import (
	"path/filepath"
	"strings"

	"github.com/IvyMurage/otelstarter/internal/project"
)

// The tested version set. OpenTelemetry's Python packages require each other
// at exact versions, so they must be pinned together. Contrib packages
// (distro and instrumentation-*) share one version; exporters follow the SDK.
// Update both together, after an end-to-end test.
const (
	contribVersion = "0.66b0"
	sdkVersion     = "1.45.0"

	// otelstarterVersion is the domain-event SDK in sdk/python, published to
	// PyPI. A test keeps it equal to sdk/python/src/otelstarter/__init__.py.
	otelstarterVersion = "0.2.0"
)

// Markers around the block otelstarter adds to requirements.txt. Removing
// the block undoes the change.
const (
	blockStart = "# >>> otelstarter: OpenTelemetry packages"
	blockEnd   = "# <<< otelstarter"
)

// EnvVars are the Python-specific settings for otel.env.
//
// Log export needs no setting: opentelemetry-instrumentation-logging installs
// the OpenTelemetry log handler itself. The older
// OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED switch is deprecated, and
// setting it makes that instrumentation skip its own handler.
var EnvVars = []project.EnvVar{
	{
		Name:    "OTEL_PYTHON_LOG_CORRELATION",
		Value:   "true",
		Comment: "Add trace_id and span_id to log messages.",
	},
}

// MissingRequirements returns pinned requirement lines, such as
// "opentelemetry-distro==0.66b0", for recommended packages the project does
// not have yet.
func MissingRequirements(lang project.Language) []string {
	var reqs []string
	for _, inst := range lang.Instrumentation {
		if !inst.Installed {
			reqs = append(reqs, inst.Package+"=="+pinnedVersion(inst.Package))
		}
	}
	return reqs
}

func pinnedVersion(pkg string) string {
	if pkg == "otelstarter" {
		return otelstarterVersion
	}
	if strings.HasPrefix(pkg, "opentelemetry-exporter-") {
		return sdkVersion
	}
	return contribVersion
}

// InstallCommand returns the command that adds reqs with the project's
// package manager. For pip, otelstarter edits requirements.txt itself, so
// the command only installs it.
func InstallCommand(packageManager string, reqs []string) string {
	packages := strings.Join(reqs, " ")
	switch packageManager {
	case "pip":
		return "pip install -r requirements.txt"
	case "uv":
		return "uv add " + packages
	case "Poetry":
		return "poetry add " + packages
	case "PDM":
		return "pdm add " + packages
	case "Pipenv":
		return "pipenv install " + packages
	default:
		return "pip install " + packages
	}
}

// AddToRequirements returns requirements.txt content with reqs added inside
// the otelstarter block, creating the block at the end if it is missing.
func AddToRequirements(existing string, reqs []string) string {
	lines := strings.Join(reqs, "\n") + "\n"

	if start := strings.Index(existing, blockStart+"\n"); start >= 0 {
		if end := strings.Index(existing[start:], blockEnd); end >= 0 {
			insertAt := start + end
			return existing[:insertAt] + lines + existing[insertAt:]
		}
	}

	if existing != "" {
		if !strings.HasSuffix(existing, "\n") {
			existing += "\n"
		}
		existing += "\n"
	}
	return existing + blockStart + "\n" + lines + blockEnd + "\n"
}

// WrapCommand puts opentelemetry-instrument in front of the user's start
// command, unless it is already there (for example
// "uv run opentelemetry-instrument uvicorn main:app"). The second result
// reports whether it was added.
func WrapCommand(command []string) ([]string, bool) {
	for _, arg := range command {
		if filepath.Base(arg) == "opentelemetry-instrument" {
			return command, false
		}
	}
	return append([]string{"opentelemetry-instrument"}, command...), true
}

// Essential reports whether a package is required for any telemetry to be
// sent. Missing instrumentation for one library only loses that library's
// spans; missing these loses everything.
func Essential(pkg string) bool {
	return pkg == "opentelemetry-distro" || pkg == "opentelemetry-exporter-otlp-proto-http"
}

// TestedSDKVersion returns the otelstarter SDK version that init pins.
func TestedSDKVersion() string {
	return otelstarterVersion
}

// TestedVersions returns the version set otelstarter pins and tests with:
// one for beta packages (distro, instrumentation-*) and one for stable
// packages (API, SDK, exporters).
func TestedVersions() (beta, stable string) {
	return contribVersion, sdkVersion
}
