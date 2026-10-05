package localstack

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// The Go list of ports and the Compose file must agree, or the port check
// would test the wrong ports.
func TestPortsMatchComposeFile(t *testing.T) {
	compose, err := files.ReadFile("files/compose.yaml")
	if err != nil {
		t.Fatal(err)
	}
	published := strings.Count(string(compose), "- 127.0.0.1:")
	if published != len(Ports) {
		t.Errorf("compose.yaml publishes %d ports, Ports lists %d", published, len(Ports))
	}
	for _, p := range Ports {
		want := fmt.Sprintf("- 127.0.0.1:%d:%d", p.Number, p.Number)
		if !strings.Contains(string(compose), want) {
			t.Errorf("compose.yaml does not publish %q", want)
		}
	}
}

func TestCollectorConfigFiltersAsgiNoise(t *testing.T) {
	config, err := files.ReadFile("files/collector.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(config), `attributes["asgi.event.type"] != nil`) {
		t.Error("collector.yaml no longer filters the FastAPI send/receive spans")
	}
}

func TestCheckPortsFree(t *testing.T) {
	// Occupy a port, then check that it is reported as in use.
	ln, err := net.Listen("tcp", "127.0.0.1:0") // port 0 asks the OS for any free port
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	busy := ln.Addr().(*net.TCPAddr).Port

	err = CheckPortsFree([]Port{{busy, "Grafana"}})
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("port %d (Grafana) is already in use", busy)) {
		t.Errorf("CheckPortsFree() error = %v, want port %d reported as in use", err, busy)
	}

	ln.Close()
	if err := CheckPortsFree([]Port{{busy, "Grafana"}}); err != nil {
		t.Errorf("CheckPortsFree() after closing = %v, want nil", err)
	}
}

func TestCheckDockerMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // an empty folder: no docker binary anywhere

	err := CheckDocker(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Docker is not installed") {
		t.Errorf("CheckDocker() error = %v, want a 'Docker is not installed' error", err)
	}
}

func TestWaitUntilReady(t *testing.T) {
	var calls atomic.Int32 // safe to update from the server's goroutine
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable) // still starting
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := waitUntilReady(ctx, server.URL); err != nil {
		t.Fatalf("waitUntilReady() error = %v", err)
	}
	if calls.Load() != 3 {
		t.Errorf("server called %d times, want 3", calls.Load())
	}
}

func TestWaitUntilReadyGivesUp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	if err := waitUntilReady(ctx, server.URL); err == nil {
		t.Error("waitUntilReady() = nil, want a timeout error")
	}
}

// TestStackStartsAndStops starts the real stack. It needs Docker, downloads
// about 1 GB the first time, and uses ports 3000, 4317, 4318 and 13133, so it
// only runs when OTELSTARTER_DOCKER_TESTS=1.
func TestStackStartsAndStops(t *testing.T) {
	if os.Getenv("OTELSTARTER_DOCKER_TESTS") != "1" {
		t.Skip("set OTELSTARTER_DOCKER_TESTS=1 to run tests that start Docker containers")
	}
	ctx := context.Background()
	if err := CheckDocker(ctx); err != nil {
		t.Fatal(err)
	}
	stack := &Stack{Dir: t.TempDir(), Output: os.Stderr}
	defer stack.Stop(ctx)

	if err := stack.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := waitUntilReady(ctx, grafanaHealthURL); err != nil {
		t.Errorf("Grafana not ready after Start: %v", err)
	}
}
