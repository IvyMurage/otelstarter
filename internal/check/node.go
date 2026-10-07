package check

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/IvyMurage/otelstarter/internal/adapters/node"
	"github.com/IvyMurage/otelstarter/internal/project"
)

// checkNode reports whether the Node.js that will run the app is new enough
// for @opentelemetry/auto-instrumentations-node (^18.19.0 || >=20.6.0). It
// only runs "node --version", in the project folder so version managers
// such as nvm or fnm that read .nvmrc there apply.
func checkNode(ctx context.Context, dir string) Result {
	path, err := exec.LookPath("node")
	if err != nil {
		return warn("Node.js was not found on your PATH",
			"OpenTelemetry for Node needs Node.js 18.19+ or 20.6+ to run your app.",
			"Install Node.js 20 or newer and make sure node is on your PATH.")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	cmd.Dir = dir
	out, err := cmd.Output()
	version := strings.TrimPrefix(strings.TrimSpace(string(out)), "v")
	var major, minor int
	if _, scanErr := fmt.Sscanf(version, "%d.%d", &major, &minor); err != nil || scanErr != nil {
		return warn("Cannot tell which Node.js version "+path+" is",
			"OpenTelemetry for Node needs Node.js 18.19+ or 20.6+.",
			"Run node --version to check it yourself.")
	}

	supported := (major == 18 && minor >= 19) || (major == 20 && minor >= 6) || major > 20
	if !supported {
		return fail(fmt.Sprintf("Node.js %s is too old for OpenTelemetry (%s)", version, path),
			"@opentelemetry/auto-instrumentations-node needs Node.js 18.19+ or 20.6+.",
			"Install Node.js 20 or newer, for example with nvm: nvm install 20 && nvm use 20.")
	}
	return pass(fmt.Sprintf("Node.js %s (%s)", version, path))
}

// checkNodeModules reports whether the instrumentation package is installed,
// not just declared: run loads it from node_modules.
func checkNodeModules(dir string) Result {
	if !node.Installed(dir) {
		return warn(node.RegisterModule+" is not installed",
			"otelstarter run loads it from node_modules, so the app would start without instrumentation.",
			"Install your dependencies, for example: npm install.")
	}
	return pass("Instrumentation installed in node_modules")
}

// checkNodeAdvice turns the analysis notes that limit what reaches the
// backend (no logging library, ES modules, frameworks the bundle cannot
// instrument) into warnings.
func checkNodeAdvice(lang project.Language) []Result {
	var results []Result
	for _, note := range lang.Notes {
		title := note
		if i := strings.Index(note, ". "); i > 0 {
			title = note[:i] // the first sentence, as the headline
		}
		results = append(results, warn(title, note, ""))
	}
	return results
}
