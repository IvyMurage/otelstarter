package node

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IvyMurage/otelstarter/internal/project"
)

// The tested versions init pins. The API and the bundle are released
// separately, so each has its own version.
const (
	autoVersion = "0.80.0"
	apiVersion  = "1.9.1"
)

// EnvVars are the Node.js-specific settings for otel.env.
func EnvVars(lang project.Language) []project.EnvVar {
	return []project.EnvVar{
		{
			Name:    "OTEL_NODE_ENABLED_INSTRUMENTATIONS",
			Value:   strings.Join(lang.Enabled, ","),
			Comment: "Instrumentations to load: only those for libraries this app uses.",
		},
		{
			// Without this, startup tries cloud metadata servers (GCP, AWS…)
			// and prints a warning on machines outside those clouds.
			Name:    "OTEL_NODE_RESOURCE_DETECTORS",
			Value:   "env,host,os,process",
			Comment: "Describe the service from the environment and host, without probing cloud metadata servers.",
		},
	}
}

// MissingPackages returns pinned packages, such as
// "@opentelemetry/api@1.9.1", that the project does not have yet.
func MissingPackages(lang project.Language) []string {
	var pkgs []string
	for _, inst := range lang.Instrumentation {
		if !inst.Installed {
			pkgs = append(pkgs, inst.Package+"@"+pinnedVersion(inst.Package))
		}
	}
	return pkgs
}

func pinnedVersion(pkg string) string {
	if pkg == apiPackage {
		return apiVersion
	}
	return autoVersion
}

// InstallCommand returns the command that adds pkgs with exact versions.
// otelstarter never edits package.json itself.
func InstallCommand(packageManager string, pkgs []string) string {
	list := strings.Join(pkgs, " ")
	switch packageManager {
	case "yarn":
		return "yarn add --exact " + list
	case "pnpm":
		return "pnpm add --save-exact " + list
	case "bun":
		return "bun add --exact " + list
	default:
		return "npm install --save-exact " + list
	}
}

// Essential reports whether a package is required for any telemetry.
func Essential(pkg string) bool {
	return pkg == apiPackage || pkg == autoPackage
}

// launchers are programs that start Node.js, so run instruments the
// command through NODE_OPTIONS.
var launchers = []string{"node", "npm", "npx", "yarn", "pnpm", "bun", "nest", "ts-node", "tsx", "nodemon"}

// LaunchesNode reports whether command starts a Node.js program.
func LaunchesNode(command []string) bool {
	return len(command) > 0 && slices.Contains(launchers, filepath.Base(command[0]))
}

// WithRegister adds --require RegisterModule to an existing NODE_OPTIONS
// value, keeping whatever was there.
func WithRegister(nodeOptions string) string {
	flag := "--require " + RegisterModule
	if strings.Contains(nodeOptions, RegisterModule) {
		return nodeOptions
	}
	return strings.TrimSpace(nodeOptions + " " + flag)
}

// Installed reports whether the instrumentation package is installed in
// dir's node_modules. It checks the package itself: "/register" is mapped
// to a file inside the package through its package.json "exports".
func Installed(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "node_modules", filepath.FromSlash(autoPackage), "package.json"))
	return err == nil
}
