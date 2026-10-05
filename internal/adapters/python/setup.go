package python

import (
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
)

// Markers around the block otelstarter adds to requirements.txt. Removing
// the block undoes the change.
const (
	blockStart = "# >>> otelstarter: OpenTelemetry packages"
	blockEnd   = "# <<< otelstarter"
)

// EnvVars are the Python-specific settings for otel.env.
var EnvVars = []project.EnvVar{
	{
		Name:    "OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED",
		Value:   "true",
		Comment: "Send Python log records through OpenTelemetry.",
	},
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
