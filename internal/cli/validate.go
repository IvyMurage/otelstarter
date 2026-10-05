package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/IvyMurage/otelstarter/internal/check"
)

// problemsFound makes validate and doctor exit with code 1. The details
// are already printed, so Run adds nothing.
type problemsFound struct{ failures int }

func (p problemsFound) Error() string { return fmt.Sprintf("%d problem(s) found", p.failures) }

func runValidate(fs *flag.FlagSet, args []string, s streams) error {
	offline := fs.Bool("offline", false, "skip the Collector check, for CI and machines without a Collector")
	rest, err := parseArgs(fs, args, 1)
	if err != nil {
		return err
	}

	results := check.Run(context.Background(), check.Options{Dir: pathArg(rest), Offline: *offline})
	return printResults(s.stdout, results)
}

func runDoctor(fs *flag.FlagSet, args []string, s streams) error {
	rest, err := parseArgs(fs, args, 1)
	if err != nil {
		return err
	}
	dir := pathArg(rest)
	ctx := context.Background()

	fmt.Fprintln(s.stdout, "Checks:")
	results := check.Run(ctx, check.Options{Dir: dir})
	checkErr := printResults(s.stdout, results)

	fmt.Fprintln(s.stdout, "\nEnvironment (secrets are hidden, so this is safe to paste into a bug report):")
	report := check.Environment(ctx, dir, Version)
	for _, e := range report.Facts {
		fmt.Fprintf(s.stdout, "  %-26s %s\n", e.Name+":", e.Value)
	}
	fmt.Fprintln(s.stdout, "\notel.env:")
	if report.EnvError != "" {
		fmt.Fprintf(s.stdout, "  %s\n", report.EnvError)
	}
	for _, setting := range report.Settings {
		fmt.Fprintf(s.stdout, "  %s\n", setting)
	}
	return checkErr
}

// printResults prints the checklist and returns problemsFound when any
// check failed.
func printResults(w io.Writer, results []check.Result) error {
	for _, r := range results {
		fmt.Fprintf(w, "  %s %s\n", symbol(r.Status), r.Title)
		if r.Why != "" {
			fmt.Fprintf(w, "      Why: %s\n", r.Why)
		}
		if r.Fix != "" {
			fmt.Fprintf(w, "      Fix: %s\n", r.Fix)
		}
	}

	failures, warnings := check.Count(results)
	switch {
	case failures == 0 && warnings == 0:
		fmt.Fprintln(w, "\nAll checks passed.")
	default:
		fmt.Fprintf(w, "\n%s, %s.\n", plural(failures, "problem"), plural(warnings, "warning"))
	}
	if failures > 0 {
		return problemsFound{failures}
	}
	return nil
}

func symbol(s check.Status) string {
	switch s {
	case check.Fail:
		return "✗"
	case check.Warn:
		return "!"
	default:
		return "✓"
	}
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
