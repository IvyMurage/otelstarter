package check

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/IvyMurage/otelstarter/internal/adapters/python"
	"github.com/IvyMurage/otelstarter/internal/project"
)

// checkCollector sends an empty OTLP export for each signal. A Collector
// answers 200 and stores nothing, so this proves the endpoint accepts
// telemetry without adding test data to anyone's backend.
func checkCollector(ctx context.Context, base string) Result {
	client := &http.Client{Timeout: 3 * time.Second}
	base = strings.TrimSuffix(base, "/")

	for _, signal := range []string{"traces", "metrics", "logs"} {
		url := base + "/v1/" + signal
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader("{}"))
		if err != nil {
			return fail("Collector at "+base, err.Error(), "")
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return collectorUnreachable(base, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fail(fmt.Sprintf("Collector at %s answered %d for %s", base, resp.StatusCode, signal),
				"Something is listening there, but it did not accept OTLP data. It may not be an OpenTelemetry Collector, or it may not have an OTLP/HTTP receiver.",
				"Check that OTEL_EXPORTER_OTLP_ENDPOINT points at a Collector's OTLP/HTTP port (usually 4318).")
		}
	}
	return pass("Collector at " + base + " accepts traces, metrics and logs")
}

func collectorUnreachable(base string, err error) Result {
	// errors.As looks through wrapped errors for a specific type; here, a
	// network operation error such as "connection refused".
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return fail("Nothing is listening at "+base,
			"Your app's telemetry would be dropped, because no Collector is running at that address.",
			"Start the local stack with: otelstarter run. Or point OTEL_EXPORTER_OTLP_ENDPOINT at your Collector. "+
				"In CI without a Collector, use: otelstarter validate --offline")
	}
	return fail("Cannot reach the Collector at "+base, err.Error(),
		"Check the address in OTEL_EXPORTER_OTLP_ENDPOINT and your network.")
}

// checkPackages reports recommended OpenTelemetry packages missing from the
// project's manifest. Missing essentials fail; others warn, because a
// project may leave out an instrumentation on purpose.
func checkPackages(lang project.Language) []Result {
	var missingEssential, missingOther []string
	for _, inst := range lang.Instrumentation {
		switch {
		case inst.Installed:
		case python.Essential(inst.Package):
			missingEssential = append(missingEssential, inst.Package)
		default:
			missingOther = append(missingOther, inst.Package)
		}
	}

	var results []Result
	if len(missingEssential) > 0 {
		results = append(results, fail("Missing required packages: "+strings.Join(missingEssential, ", "),
			"Without them the app cannot send any telemetry.",
			"Run otelstarter init, then install the project's dependencies."))
	}
	if len(missingOther) > 0 {
		results = append(results, warn("Missing recommended instrumentation: "+strings.Join(missingOther, ", "),
			"Calls through these libraries will not appear in traces or metrics.",
			"Run otelstarter init to add them, or ignore this if you left them out on purpose."))
	}
	if len(results) == 0 {
		results = append(results, pass(fmt.Sprintf("All %d recommended packages are declared", len(lang.Instrumentation))))
	}
	return results
}

// checkLauncher looks for opentelemetry-instrument on PATH, which is where
// it is when the project's virtual environment is active.
func checkLauncher() Result {
	path, err := exec.LookPath("opentelemetry-instrument")
	if err != nil {
		return warn("opentelemetry-instrument is not on your PATH",
			"The app cannot be started with instrumentation from this shell.",
			"Activate your project's virtual environment and install its dependencies (for example: pip install -r requirements.txt).")
	}
	return pass("opentelemetry-instrument: " + path)
}
