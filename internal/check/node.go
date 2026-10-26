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
// for the otelstarter SDK and OpenTelemetry (^18.19.0 || >=20.6.0), and,
// for an ES-module app, for the SDK's import hook (22.14+). It only runs
// "node --version", in the project folder so version managers such as nvm
// or fnm that read .nvmrc there apply.
func checkNode(ctx context.Context, dir string, esModule bool) Result {
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
			"OpenTelemetry for Node needs Node.js 18.19+ or 20.6+.",
			"Install Node.js 20 or newer, for example with nvm: nvm install 20 && nvm use 20.")
	}
	// worker_threads.isInternalThread, which the SDK needs to run its import
	// hook safely, arrived in 22.14 and 23.7.
	hooksImports := major > 23 || (major == 23 && minor >= 7) || (major == 22 && minor >= 14)
	if esModule && !hooksImports {
		return warn(fmt.Sprintf("Node.js %s instruments only part of an ES-module app (%s)", version, path),
			"The SDK instruments modules loaded with import from Node.js 22.14. On older versions, frameworks and "+
				"libraries the app imports (Express, pg…) are not traced; incoming HTTP requests still are.",
			"Install Node.js 22.14 or newer, for example with nvm: nvm install 22 && nvm use 22.")
	}
	return pass(fmt.Sprintf("Node.js %s (%s)", version, path))
}

// checkNodeModules reports whether the packages are installed, not just
// declared: run loads them from node_modules.
func checkNodeModules(dir string, lang project.Language) Result {
	if _, ok := node.Loader(dir); !ok {
		return warn(node.RegisterModule+" is not installed",
			"otelstarter run loads it from node_modules, so the app would start without instrumentation.",
			"Install your dependencies, for example: npm install.")
	}
	if missing := node.NotInstalled(dir, lang); len(missing) > 0 {
		return warn("Declared but not installed: "+strings.Join(missing, ", "),
			"The SDK skips instrumentation it cannot find, so calls through these libraries will not appear in traces.",
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
