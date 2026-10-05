package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/IvyMurage/otelstarter/internal/setup"
)

func runInit(fs *flag.FlagSet, args []string, s streams) error {
	// fs.Bool returns a pointer: the flag package fills in the value when
	// parseArgs runs, so read it with *dryRun afterwards.
	dryRun := fs.Bool("dry-run", false, "show the plan without changing any files")
	yes := fs.Bool("yes", false, "apply the plan without asking for confirmation")
	rest, err := parseArgs(fs, args, 1)
	if err != nil {
		return err
	}

	plan, err := setup.BuildPlan(pathArg(rest))
	if err != nil {
		return err
	}

	printPlan(s.stdout, plan)
	if plan.NothingToDo() {
		fmt.Fprintln(s.stdout, "Already set up. Nothing to change.")
		return nil
	}
	if len(plan.Changes) == 0 {
		return nil // only commands for the user to run; there is nothing to write
	}
	if *dryRun {
		fmt.Fprintln(s.stdout, "Dry run: no files were changed.")
		return nil
	}

	if !*yes {
		if !isTerminal(s.stdin) {
			return errors.New("init changes files and needs confirmation, but no terminal is attached. " +
				"Run it in a terminal, or pass --yes to apply without asking")
		}
		if !confirm(s.stdin, s.stdout, "Apply these changes? [y/N] ") {
			fmt.Fprintln(s.stdout, "No changes made.")
			return nil
		}
	}

	if err := setup.Apply(plan); err != nil {
		return err
	}
	for _, c := range plan.Changes {
		fmt.Fprintf(s.stdout, "%s %s\n", pastTense(c.Action()), c.Path)
	}
	fmt.Fprintln(s.stdout, "Done. Next, run the commands listed above.")
	return nil
}

func printPlan(w io.Writer, plan setup.Plan) {
	fmt.Fprintf(w, "Project: %s (%s)\n", plan.Path, plan.Description)

	if len(plan.Changes) > 0 {
		fmt.Fprintln(w, "\nFiles to change:")
		for _, c := range plan.Changes {
			fmt.Fprintf(w, "  %-7s %-18s %s\n", c.Action(), c.Path, c.Summary)
		}
		for _, c := range plan.Changes {
			fmt.Fprintf(w, "\n%s %s:\n", c.Path, previewLabel(c))
			for _, line := range c.Preview {
				fmt.Fprintf(w, "  | %s\n", line)
			}
		}
	}

	if len(plan.Commands) > 0 || len(plan.Changes) > 0 {
		fmt.Fprintln(w, "\nThen run:")
		for _, cmd := range plan.Commands {
			fmt.Fprintf(w, "  %s\n", cmd)
		}
		fmt.Fprintf(w, "  %s\n", plan.StartCommand)
		fmt.Fprintln(w, "  (replace the command after -- with your own; run starts a local Collector and Grafana first)")
	}

	if len(plan.Notes) > 0 {
		fmt.Fprintln(w)
		for _, note := range plan.Notes {
			fmt.Fprintf(w, "Note: %s\n", note)
		}
	}
	fmt.Fprintln(w)
}

func previewLabel(c setup.Change) string {
	if c.Action() == "create" {
		return "(new file)"
	}
	return "(lines added)"
}

func pastTense(action string) string {
	if action == "create" {
		return "Created"
	}
	return "Updated"
}

// isTerminal reports whether r is an interactive terminal rather than a
// pipe, a file or a test buffer.
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File) // a type assertion: is this io.Reader really an *os.File?
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// confirm asks a yes/no question. Anything other than "y" or "yes" means no,
// so pressing Enter never changes files.
func confirm(in io.Reader, out io.Writer, question string) bool {
	fmt.Fprint(out, question)
	answer, _ := bufio.NewReader(in).ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}
