package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/IvyMurage/otelstarter/internal/adapters/python"
	"github.com/IvyMurage/otelstarter/internal/detect"
	"github.com/IvyMurage/otelstarter/internal/localstack"
	"github.com/IvyMurage/otelstarter/internal/project"
	"github.com/IvyMurage/otelstarter/internal/setup"
)

// How long the app gets to shut down after Ctrl+C before otelstarter
// interrupts it, and then kills it.
const (
	appGracePeriod = 10 * time.Second
	appKillAfter   = 5 * time.Second
)

func runRun(fs *flag.FlagSet, args []string, s streams) error {
	// Everything after "--" is the app's own command, untouched by our flags.
	ownArgs, appCommand := splitAtDoubleDash(args)
	if _, err := parseArgs(fs, ownArgs, 0); err != nil {
		if errors.Is(err, errUsage) && !slices.Contains(args, "--") && fs.NArg() > 0 {
			fmt.Fprintf(s.stderr, "To start your app too, put -- before its command, for example:\n"+
				"  otelstarter run -- uvicorn main:app\n")
		}
		return err
	}

	// Prepare the app before starting Docker, so mistakes show up at once.
	var app *exec.Cmd
	if len(appCommand) > 0 {
		var err error
		if app, err = prepareApp(appCommand, s); err != nil {
			return err
		}
	}

	// ctx is cancelled when the user presses Ctrl+C (or the process gets SIGTERM).
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	if err := localstack.CheckDocker(ctx); err != nil {
		return err
	}
	stack, err := localstack.New(s.stderr)
	if err != nil {
		return err
	}

	if localstack.Running(ctx) {
		// Another otelstarter run, usually in another terminal, owns the
		// stack. Share it, and leave stopping it to that session: starting
		// again would remove its containers.
		fmt.Fprintf(s.stdout, "Using the local stack that is already running (started by another otelstarter run).\n"+
			"  Grafana    %s\n\n", localstack.GrafanaURL)
		if app == nil {
			fmt.Fprintln(s.stdout, "Nothing else to start. Stop the stack with Ctrl+C where it was started.")
			return nil
		}
		fmt.Fprintf(s.stdout, "Starting your app: %s\nPress Ctrl+C to stop the app. The shared stack keeps running.\n\n",
			strings.Join(app.Args, " "))
		return runApp(ctx, app)
	}

	fmt.Fprintln(s.stdout, "Starting the local observability stack. The first run downloads about 1 GB of images.")
	// defer runs when runRun returns, however it returns, so the stack is
	// always cleaned up.
	defer stopStack(stack, s)
	if err := stack.Start(ctx); err != nil {
		if ctx.Err() != nil {
			return nil // the user pressed Ctrl+C while starting
		}
		return err
	}

	fmt.Fprintf(s.stdout, "\nReady.\n"+
		"  Collector  %s   (OTLP; your app sends here)\n"+
		"  Grafana    %s   (Explore: Tempo for traces, Loki for logs, Prometheus for metrics)\n"+
		"  Files      %s\n\n",
		localstack.CollectorURL, localstack.GrafanaURL, stack.Dir)

	if app == nil {
		fmt.Fprintln(s.stdout, "Start your app in another terminal, or next time use: otelstarter run -- <your start command>")
		fmt.Fprintln(s.stdout, "Press Ctrl+C to stop the stack.")
		<-ctx.Done()
		return nil
	}

	fmt.Fprintf(s.stdout, "Starting your app: %s\nPress Ctrl+C to stop the app and the stack.\n\n", strings.Join(app.Args, " "))
	appErr := runApp(ctx, app)
	if ctx.Err() != nil || !isTerminal(s.stdin) {
		// Stopped with Ctrl+C, or running in a script or CI where nobody
		// could press Ctrl+C later: stop the stack now.
		return appErr
	}

	// The app exited by itself. Keep the stack so its telemetry, often the
	// most useful clue when it crashed, can still be explored in Grafana.
	status := "normally"
	if appErr != nil {
		status = "with an error (" + strings.TrimPrefix(appErr.Error(), "your app exited: ") + ")"
	}
	fmt.Fprintf(s.stdout, "\nYour app exited %s. The stack is still running, so its telemetry is still in Grafana at %s.\n"+
		"Press Ctrl+C to stop the stack, then run otelstarter run -- ... again.\n", status, localstack.GrafanaURL)
	<-ctx.Done()
	return nil
}

// splitAtDoubleDash splits args at the first "--".
func splitAtDoubleDash(args []string) (before, after []string) {
	if i := slices.Index(args, "--"); i >= 0 {
		return args[:i], args[i+1:]
	}
	return args, nil
}

// prepareApp builds the app's command with otel.env loaded and, for Python,
// opentelemetry-instrument in front.
func prepareApp(command []string, s streams) (*exec.Cmd, error) {
	envVars, err := setup.ReadEnvFile(".")
	if err != nil {
		return nil, err
	}

	if proj, err := detect.Detect("."); err == nil && hasLanguage(proj.Languages, "Python") {
		var wrapped bool
		command, wrapped = python.WrapCommand(command)
		if _, err := exec.LookPath("opentelemetry-instrument"); wrapped && err != nil {
			return nil, errors.New("opentelemetry-instrument was not found. Activate your project's virtual " +
				"environment and install its packages (for example: pip install -r requirements.txt), then try again")
		}
	}

	if _, err := exec.LookPath(command[0]); err != nil {
		return nil, fmt.Errorf("cannot find %q. Check the command, or activate your virtual environment", command[0])
	}

	cmd := exec.Command(command[0], command[1:]...)
	// Print each domain event to the terminal too (otelstarter SDK 0.2.0+).
	// Listed with the file's settings, so the shell can still override it.
	envVars = append(envVars, "OTELSTARTER_CONSOLE_EVENTS=true")
	cmd.Env = mergeEnv(os.Environ(), envVars)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = s.stdin, s.stdout, s.stderr
	return cmd, nil
}

func hasLanguage(langs []project.Language, name string) bool {
	for _, l := range langs {
		if l.Name == name {
			return true
		}
	}
	return false
}

// mergeEnv adds fileVars to environ. A variable already set in the shell
// wins over the file, as with Docker Compose, so a one-off override like
// OTEL_SERVICE_NAME=test otelstarter run -- ... works.
func mergeEnv(environ, fileVars []string) []string {
	set := map[string]bool{}
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		set[name] = true
	}
	merged := slices.Clone(environ)
	for _, kv := range fileVars {
		name, _, _ := strings.Cut(kv, "=")
		if !set[name] {
			merged = append(merged, kv)
		}
	}
	return merged
}

// runApp runs the app until it exits or ctx is cancelled. On Ctrl+C the
// terminal already sends the interrupt to the app too, so we first give it
// time to shut down by itself, and only then interrupt or kill it.
func runApp(ctx context.Context, app *exec.Cmd) error {
	if err := app.Start(); err != nil {
		return fmt.Errorf("cannot start your app: %w", err)
	}

	// A channel carries one value from the goroutine below to this function:
	// the app's exit result.
	exited := make(chan error, 1)
	go func() { exited <- app.Wait() }()

	select {
	case err := <-exited:
		if err != nil {
			return fmt.Errorf("your app exited: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	for _, step := range []struct {
		wait   time.Duration
		action func() error
	}{
		{appGracePeriod, func() error { return app.Process.Signal(os.Interrupt) }},
		{appKillAfter, app.Process.Kill},
	} {
		select {
		case <-exited:
			return nil
		case <-time.After(step.wait):
			_ = step.action()
		}
	}
	<-exited
	return nil
}

func stopStack(stack *localstack.Stack, s streams) {
	fmt.Fprintln(s.stdout, "\nStopping the local stack...")
	// A fresh context: the original one is already cancelled after Ctrl+C.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := stack.Stop(ctx); err != nil {
		fmt.Fprintf(s.stderr, "Could not stop the stack: %v. Stop it with: docker compose -p otelstarter down\n", err)
	}
}
