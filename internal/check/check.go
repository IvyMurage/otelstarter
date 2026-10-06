// Package check runs the checks behind `otelstarter validate` and
// `otelstarter doctor`: is the project set up correctly, and can its
// telemetry reach a Collector?
//
// Checks only read files and, unless offline, send empty OTLP requests. They
// never change anything.
package check

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/IvyMurage/otelstarter/internal/detect"
	"github.com/IvyMurage/otelstarter/internal/project"
	"github.com/IvyMurage/otelstarter/internal/setup"
)

// Status is the outcome of one check.
type Status int

// iota numbers the constants 0, 1, 2 in order.
const (
	Pass Status = iota
	Warn
	Fail
)

// Result is one line of the checklist. Why and Fix are set for Warn and Fail.
type Result struct {
	Status Status
	Title  string
	Why    string
	Fix    string
}

// Options controls which checks run.
type Options struct {
	Dir     string
	Offline bool // skip checks that use the network
}

// Run runs every check that applies and returns the results in order. Later
// checks are skipped when an earlier one makes them meaningless, for example
// configuration checks when there is no otel.env.
func Run(ctx context.Context, opts Options) []Result {
	var results []Result
	add := func(r ...Result) { results = append(results, r...) }

	proj, err := detect.Detect(opts.Dir)
	if err != nil {
		return []Result{fail("Project", err.Error(), "")}
	}
	if len(proj.Languages) == 0 {
		return []Result{fail("Project", detect.NoProjectError(proj.Path).Error(), "")}
	}
	lang, ok := findLanguage(proj, "Python")
	if !ok {
		return []Result{fail("No Python project found in "+proj.Path,
			"otelstarter validates Python projects only for now.",
			"Run otelstarter validate from your project's root folder, or pass its path.")}
	}
	add(pass(fmt.Sprintf("Python project (%s)", describe(lang))))

	envVars, err := setup.ReadEnvFile(proj.Path)
	if err != nil {
		add(fail("otel.env", err.Error(), "Run otelstarter init to create it."))
	} else {
		env := toMap(envVars)
		add(pass("otel.env"))
		add(checkServiceName(env))
		add(checkProtocol(env))
		add(checkEndpointURL(env))
		add(checkDeprecated(env)...)
		add(checkSecrets(env)...)
		if !opts.Offline {
			add(checkCollector(ctx, endpoint(env)))
		}
	}

	add(checkPackages(lang)...)
	add(checkVersions(lang)...)
	add(checkLauncher())
	add(checkPython(ctx, proj.Path))
	return results
}

// Count returns how many results have each problem status.
func Count(results []Result) (failures, warnings int) {
	for _, r := range results {
		switch r.Status {
		case Fail:
			failures++
		case Warn:
			warnings++
		}
	}
	return failures, warnings
}

func pass(title string) Result { return Result{Status: Pass, Title: title} }

func warn(title, why, fix string) Result {
	return Result{Status: Warn, Title: title, Why: why, Fix: fix}
}

func fail(title, why, fix string) Result {
	return Result{Status: Fail, Title: title, Why: why, Fix: fix}
}

func findLanguage(proj project.Project, name string) (project.Language, bool) {
	for _, l := range proj.Languages {
		if l.Name == name {
			return l, true
		}
	}
	return project.Language{}, false
}

func describe(lang project.Language) string {
	parts := slices.Clone(lang.Frameworks)
	if lang.PackageManager != "" {
		parts = append(parts, lang.PackageManager)
	}
	if len(parts) == 0 {
		return "no framework detected"
	}
	return strings.Join(parts, ", ")
}

// toMap turns "NAME=value" strings into a map. A later line wins, as when
// the shell reads the file.
func toMap(vars []string) map[string]string {
	m := map[string]string{}
	for _, kv := range vars {
		name, value, _ := strings.Cut(kv, "=")
		m[name] = value
	}
	return m
}

// SortedNames returns the map's keys in order, so output is stable.
func SortedNames(env map[string]string) []string {
	return slices.Sorted(maps.Keys(env))
}
