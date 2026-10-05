// Package cli parses command-line arguments and dispatches to commands.
//
// The CLI only orchestrates. Detection, configuration, validation and other
// real work belong in their own packages, so handlers here stay thin.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"runtime"
)

// Exit codes follow the common Unix convention: 2 means the user typed the
// command wrong, 1 means the command ran and failed.
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

// Version is replaced at build time with:
//
//	go build -ldflags "-X github.com/Murage-Ivy/otelstarter/internal/cli.Version=v0.1.0"
var Version = "dev"

type command struct {
	name    string
	summary string
	run     func(stdout io.Writer) error
}

// commands is a slice, not a map, so help output keeps this order.
var commands = []command{
	{"init", "Add OpenTelemetry configuration to the current project", notImplemented("init", "Phase 4")},
	{"analyze", "Detect the project's language, framework and dependencies", notImplemented("analyze", "Phase 2")},
	{"run", "Start a local collector and observability backend", notImplemented("run", "Phase 6")},
	{"validate", "Check that the generated configuration works", notImplemented("validate", "Phase 7")},
	{"doctor", "Diagnose common setup problems and suggest fixes", notImplemented("doctor", "Phase 7")},
	{"version", "Print the otelstarter version", printVersion},
}

// Run executes the CLI and returns the process exit code. It takes its
// arguments and output streams as parameters so tests can call it directly.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return exitUsage
	}

	name := args[0]
	if name == "help" || name == "-h" || name == "--help" {
		printUsage(stdout)
		return exitOK
	}

	cmd, ok := findCommand(name)
	if !ok {
		fmt.Fprintf(stderr, "Unknown command %q.\n\nRun 'otelstarter help' to see available commands.\n", name)
		return exitUsage
	}

	// No command takes flags yet; parsing still gives every command --help
	// and rejects stray arguments instead of silently ignoring them.
	flags := flag.NewFlagSet(cmd.name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprintf(stderr, "Usage: otelstarter %s\n\n%s.\n", cmd.name, cmd.summary)
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "otelstarter %s takes no arguments, got %q.\n", cmd.name, flags.Arg(0))
		return exitUsage
	}

	if err := cmd.run(stdout); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitError
	}
	return exitOK
}

func findCommand(name string) (command, bool) {
	for _, c := range commands {
		if c.name == name {
			return c, true
		}
	}
	return command{}, false
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "otelstarter sets up OpenTelemetry logs, metrics and traces for your application.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  otelstarter <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, c := range commands {
		fmt.Fprintf(w, "  %-10s %s\n", c.name, c.summary)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'otelstarter <command> --help' for details on a command.")
}

func printVersion(stdout io.Writer) error {
	fmt.Fprintf(stdout, "otelstarter %s (%s, %s/%s)\n", Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	return nil
}

// notImplemented returns a placeholder handler so every MVP command is
// discoverable now and fails with a clear message until it is built.
func notImplemented(name, phase string) func(io.Writer) error {
	return func(io.Writer) error {
		return fmt.Errorf("'otelstarter %s' is not implemented yet. It is planned for %s of the MVP", name, phase)
	}
}
