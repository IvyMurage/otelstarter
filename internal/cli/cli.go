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
	"strings"
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
//	go build -ldflags "-X github.com/IvyMurage/otelstarter/internal/cli.Version=v0.1.0"
var Version = "dev"

// errUsage means the command line was wrong. The details have already been
// printed, so Run only sets the exit code.
var errUsage = errors.New("invalid usage")

// streams bundles the input and output a command may use. Tests pass
// buffers instead of the real terminal.
type streams struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

type command struct {
	name    string
	args    string // positional arguments shown in help, e.g. "[path]"
	summary string
	// run registers its own flags on fs, parses args with parseArgs, then
	// does its work.
	run func(fs *flag.FlagSet, args []string, s streams) error
}

// commands is a slice, not a map, so help output keeps this order.
var commands = []command{
	{"init", "[path]", "Add OpenTelemetry configuration to the project", runInit},
	{"analyze", "[path]", "Detect the project's language, framework and dependencies", runAnalyze},
	{"run", "[-- <app start command>]", "Start a local Collector and Grafana, and optionally your app", runRun},
	{"validate", "[path]", "Check the OpenTelemetry setup; exit code 1 on problems", runValidate},
	{"doctor", "[path]", "Run the checks and print an environment report for troubleshooting", runDoctor},
	{"version", "", "Print the otelstarter version", runVersion},
}

// Run executes the CLI and returns the process exit code. It takes its
// arguments and streams as parameters so tests can call it directly.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
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

	fs := flag.NewFlagSet(cmd.name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: otelstarter %s\n\n%s.\n", strings.TrimSpace(cmd.name+" [flags] "+cmd.args), cmd.summary)
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}

	err := cmd.run(fs, args[1:], streams{stdin, stdout, stderr})
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		return exitOK
	case errors.Is(err, errUsage):
		return exitUsage
	case errors.As(err, new(problemsFound)):
		return exitError // the checklist already explained every problem
	default:
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitError
	}
}

// parseArgs parses flags, rejecting unknown flags and more than maxArgs
// positional arguments, and returns the positional arguments.
func parseArgs(fs *flag.FlagSet, args []string, maxArgs int) ([]string, error) {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, err
		}
		return nil, errUsage // the flag package has already printed the problem
	}
	if fs.NArg() > maxArgs {
		fmt.Fprintf(fs.Output(), "Too many arguments for otelstarter %s: unexpected %q.\n\nRun 'otelstarter %s --help' for usage.\n",
			fs.Name(), fs.Arg(maxArgs), fs.Name())
		return nil, errUsage
	}
	return fs.Args(), nil
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

func runVersion(fs *flag.FlagSet, args []string, s streams) error {
	if _, err := parseArgs(fs, args, 0); err != nil {
		return err
	}
	fmt.Fprintf(s.stdout, "otelstarter %s (%s, %s/%s)\n", Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	return nil
}
