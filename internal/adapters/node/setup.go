package node

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IvyMurage/otelstarter/internal/project"
)

// SDKVersion is the otelstarter npm package version init pins.
const SDKVersion = "0.1.4"

// pins are the tested versions init pins. All instrumentations use
// @opentelemetry/instrumentation 0.223, the version the SDK is built on, so
// npm installs one copy of it.
var pins = map[string]string{
	sdkPackage: SDKVersion,

	instrumentationPackage("amqplib"):     "0.70.0",
	instrumentationPackage("bunyan"):      "0.68.0",
	instrumentationPackage("express"):     "0.71.0",
	instrumentationPackage("graphql"):     "0.71.0",
	instrumentationPackage("hapi"):        "0.69.0",
	instrumentationPackage("ioredis"):     "0.71.0",
	instrumentationPackage("kafkajs"):     "0.32.0",
	instrumentationPackage("knex"):        "0.67.0",
	instrumentationPackage("koa"):         "0.71.0",
	instrumentationPackage("mongodb"):     "0.76.0",
	instrumentationPackage("mongoose"):    "0.69.0",
	instrumentationPackage("mysql"):       "0.69.0",
	instrumentationPackage("mysql2"):      "0.69.0",
	instrumentationPackage("nestjs-core"): "0.69.0",
	instrumentationPackage("pg"):          "0.75.0",
	instrumentationPackage("redis"):       "0.71.0",
	instrumentationPackage("winston"):     "0.67.0",
	"@opentelemetry/winston-transport":    "0.33.0",
}

// EnvVars are the Node.js-specific settings for otel.env.
func EnvVars(lang project.Language) []project.EnvVar {
	return []project.EnvVar{
		{
			Name:    "OTEL_NODE_ENABLED_INSTRUMENTATIONS",
			Value:   strings.Join(lang.Enabled, ","),
			Comment: "Instrumentations to load: only those for libraries this app uses.",
		},
	}
}

// MissingPackages returns pinned packages, such as "otelstarter@0.1.4",
// that the project does not have yet.
func MissingPackages(lang project.Language) []string {
	var pkgs []string
	for _, inst := range lang.Instrumentation {
		if !inst.Installed {
			pkgs = append(pkgs, inst.Package+"@"+pins[inst.Package])
		}
	}
	return pkgs
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
	return pkg == sdkPackage
}

// UsesBundle reports whether the project declares the instrumentation bundle
// an earlier otelstarter installed. Such a project still sends telemetry
// without the SDK, so a missing SDK is a reason to migrate, not a failure.
func UsesBundle(lang project.Language) bool {
	return slices.Contains(lang.OpenTelemetry, legacyPackage)
}

// launchers are programs that start Node.js, so run instruments the
// command through NODE_OPTIONS.
var launchers = []string{"node", "npm", "npx", "yarn", "pnpm", "bun", "nest", "ts-node", "tsx", "nodemon"}

// LaunchesNode reports whether command starts a Node.js program.
func LaunchesNode(command []string) bool {
	return len(command) > 0 && slices.Contains(launchers, filepath.Base(command[0]))
}

// WithRegister adds --require module to an existing NODE_OPTIONS value,
// keeping whatever was there. Nothing is added when a loader is already
// required, so the app is never instrumented twice.
func WithRegister(nodeOptions, module string) string {
	if strings.Contains(nodeOptions, RegisterModule) || strings.Contains(nodeOptions, legacyRegisterModule) {
		return nodeOptions
	}
	return strings.TrimSpace(nodeOptions + " --require " + module)
}

// Loader returns the module run loads with --require: the SDK's when it is
// installed in dir's node_modules, else the bundle's, for projects set up
// by an earlier otelstarter. ok is false when neither is installed.
func Loader(dir string) (module string, ok bool) {
	switch {
	case installed(dir, sdkPackage):
		return RegisterModule, true
	case installed(dir, legacyPackage):
		return legacyRegisterModule, true
	}
	return "", false
}

// NotInstalled returns the declared packages lang recommends that are
// missing from dir's node_modules, usually because npm install has not
// run since they were added.
func NotInstalled(dir string, lang project.Language) []string {
	var missing []string
	for _, inst := range lang.Instrumentation {
		if inst.Installed && !installed(dir, inst.Package) {
			missing = append(missing, inst.Package)
		}
	}
	return missing
}

// installed checks for the package's package.json: "/register" is mapped to
// a file inside the package through its "exports".
func installed(dir, pkg string) bool {
	_, err := os.Stat(filepath.Join(dir, "node_modules", filepath.FromSlash(pkg), "package.json"))
	return err == nil
}
