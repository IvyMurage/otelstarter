package check

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/IvyMurage/otelstarter/internal/localstack"
	"github.com/IvyMurage/otelstarter/internal/setup"
)

// Entry is one line of doctor's environment report.
type Entry struct {
	Name  string
	Value string
}

// Report is doctor's environment report.
type Report struct {
	Facts    []Entry  // versions, Docker, ports, PATH
	Settings []string // otel.env as NAME=value, secrets masked
	EnvError string   // why otel.env could not be read, if it could not
}

// Environment gathers facts that help diagnose a setup or file a bug report.
// Secret-looking settings are masked, so the report is safe to share.
func Environment(ctx context.Context, dir, version string) Report {
	docker, compose := localstack.DockerVersions(ctx)

	running := localstack.Running(ctx)
	stack := "not running"
	if running {
		stack = "running"
	}

	launcher := "not found on PATH"
	if path, err := exec.LookPath("opentelemetry-instrument"); err == nil {
		launcher = path
	}

	var ports []string
	for _, p := range localstack.Ports {
		state := "free"
		if localstack.PortInUse(p) {
			state = "in use"
		}
		ports = append(ports, fmt.Sprintf("%d %s", p.Number, state))
	}
	portsSummary := strings.Join(ports, ", ")
	if running {
		portsSummary += " (the local stack uses them)"
	}

	entries := []Entry{
		{"otelstarter", version},
		{"Go runtime", runtime.Version()},
		{"OS", runtime.GOOS + "/" + runtime.GOARCH},
		{"Docker", docker},
		{"Docker Compose", compose},
		{"Local stack", stack},
		{"Ports", portsSummary},
		{"opentelemetry-instrument", launcher},
	}

	report := Report{Facts: entries}
	vars, err := setup.ReadEnvFile(dir)
	if err != nil {
		report.EnvError = err.Error()
		return report
	}
	env := toMap(vars)
	for _, name := range SortedNames(env) {
		report.Settings = append(report.Settings, name+"="+masked(name, env[name]))
	}
	return report
}

func masked(name, value string) string {
	if IsSecret(name, value) {
		return "*** (hidden)"
	}
	return value
}
