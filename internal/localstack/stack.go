// Package localstack runs the local observability stack: an OpenTelemetry
// Collector in front of Grafana's LGTM image (Grafana, Tempo, Loki and
// Prometheus), started with Docker Compose.
//
// Docker is always called with a fixed list of arguments and no shell, so
// nothing a user types can be interpreted as a shell command.
package localstack

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// The //go:embed line below copies these files into the compiled binary, so
// otelstarter needs no separate template files at run time.
//
//go:embed files/compose.yaml files/collector.yaml
var files embed.FS

// Port is a port the stack publishes on 127.0.0.1.
type Port struct {
	Number  int
	Purpose string
}

// Ports must match the "ports:" entries in files/compose.yaml; a test checks.
var Ports = []Port{
	{4317, "OTLP over gRPC"},
	{4318, "OTLP over HTTP"},
	{3000, "Grafana"},
	{13133, "Collector health check"},
}

const (
	CollectorURL = "http://localhost:4318"
	GrafanaURL   = "http://localhost:3000"

	collectorHealthURL = "http://127.0.0.1:13133/"
	grafanaHealthURL   = "http://127.0.0.1:3000/api/health"
	readyTimeout       = 3 * time.Minute
)

// Stack is the local stack. Its Compose files live in Dir, not in the user's
// project.
type Stack struct {
	Dir    string
	Output io.Writer // where Docker's progress messages go
}

// New returns a Stack whose files live in the user's cache folder, for
// example ~/.cache/otelstarter on Linux.
func New(output io.Writer) (*Stack, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find a cache folder for the stack's files: %w", err)
	}
	return &Stack{Dir: filepath.Join(cache, "otelstarter"), Output: output}, nil
}

// CheckDocker makes sure Docker and Docker Compose are installed, running
// and usable by this user.
func CheckDocker(ctx context.Context) error {
	if _, err := exec.LookPath("docker"); err != nil {
		return errors.New("Docker is not installed, or not on your PATH. " +
			"otelstarter run uses Docker to start the Collector and Grafana. " +
			"Install it from https://docs.docker.com/get-docker/")
	}

	var stderr bytes.Buffer
	info := exec.CommandContext(ctx, "docker", "info")
	info.Stderr = &stderr
	if err := info.Run(); err != nil {
		if strings.Contains(stderr.String(), "permission denied") {
			return errors.New("your user cannot use Docker. On Linux, add yourself to the docker group " +
				"with: sudo usermod -aG docker $USER, then log out and back in")
		}
		return errors.New("Docker is installed but not running. Start Docker Desktop, " +
			"or on Linux run: sudo systemctl start docker")
	}

	if err := exec.CommandContext(ctx, "docker", "compose", "version").Run(); err != nil {
		return errors.New("Docker Compose is not available. otelstarter needs the 'docker compose' command (Compose v2). " +
			"See https://docs.docker.com/compose/install/")
	}
	return nil
}

// Start writes the stack's files, starts the containers and waits until the
// Collector and Grafana answer.
func (s *Stack) Start(ctx context.Context) error {
	if err := s.writeFiles(); err != nil {
		return err
	}

	// Remove containers left behind if an earlier run was killed, so they do
	// not hold the ports. Failure here is harmless: there may be nothing to remove.
	_ = s.compose(ctx, io.Discard, "down", "--remove-orphans")

	if err := CheckPortsFree(Ports); err != nil {
		return err
	}
	if err := s.compose(ctx, s.Output, "up", "--detach"); err != nil {
		return fmt.Errorf("docker compose could not start the stack: %w. See Docker's message above", err)
	}

	ctx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()
	for _, url := range []string{collectorHealthURL, grafanaHealthURL} {
		if err := waitUntilReady(ctx, url); err != nil {
			return fmt.Errorf("the stack did not become ready (%s): %w. "+
				"Check the containers with: docker compose -p otelstarter logs", url, err)
		}
	}
	return nil
}

// Stop removes the stack's containers. Telemetry stored in them is discarded.
func (s *Stack) Stop(ctx context.Context) error {
	return s.compose(ctx, s.Output, "down", "--remove-orphans")
}

func (s *Stack) writeFiles() error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", s.Dir, err)
	}
	for _, name := range []string{"compose.yaml", "collector.yaml"} {
		data, err := files.ReadFile("files/" + name)
		if err != nil {
			return err // cannot happen: the files are embedded at build time
		}
		if err := os.WriteFile(filepath.Join(s.Dir, name), data, 0o644); err != nil {
			return fmt.Errorf("cannot write %s: %w", filepath.Join(s.Dir, name), err)
		}
	}
	return nil
}

func (s *Stack) compose(ctx context.Context, out io.Writer, args ...string) error {
	full := append([]string{"compose", "--file", filepath.Join(s.Dir, "compose.yaml")}, args...)
	cmd := exec.CommandContext(ctx, "docker", full...)
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}

// CheckPortsFree reports the first port another program is already using.
func CheckPortsFree(ports []Port) error {
	for _, p := range ports {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p.Number))
		if err != nil {
			return fmt.Errorf("port %d (%s) is already in use. Stop the program using it "+
				"(find it with: lsof -i :%d) and run otelstarter run again", p.Number, p.Purpose, p.Number)
		}
		ln.Close()
	}
	return nil
}

// waitUntilReady polls url until it answers 200 OK or ctx ends.
func waitUntilReady(ctx context.Context, url string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		if resp, err := client.Do(req); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		// select waits for whichever happens first: the next tick, or ctx ending.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
