package check

import (
	"net/url"
	"strings"
)

// defaultEndpoint is what the OpenTelemetry SDKs use for OTLP over HTTP when
// OTEL_EXPORTER_OTLP_ENDPOINT is not set.
const defaultEndpoint = "http://localhost:4318"

func endpoint(env map[string]string) string {
	if e := env["OTEL_EXPORTER_OTLP_ENDPOINT"]; e != "" {
		return e
	}
	return defaultEndpoint
}

func checkServiceName(env map[string]string) Result {
	if env["OTEL_SERVICE_NAME"] == "" {
		return fail("OTEL_SERVICE_NAME is not set",
			"Telemetry is reported as \"unknown_service\", so you cannot tell this app apart from others.",
			"Add OTEL_SERVICE_NAME=<your-app-name> to otel.env.")
	}
	return pass("Service name: " + env["OTEL_SERVICE_NAME"])
}

// checkProtocol catches the most confusing failure found while building
// init: without this setting the Python SDK looks for the gRPC exporter,
// which otelstarter does not install, and silently sends nothing.
func checkProtocol(env map[string]string) Result {
	switch p := env["OTEL_EXPORTER_OTLP_PROTOCOL"]; p {
	case "http/protobuf":
		return pass("Protocol: http/protobuf")
	case "":
		return fail("OTEL_EXPORTER_OTLP_PROTOCOL is not set",
			"The Python SDK then uses gRPC, but only the HTTP exporter is installed, so no telemetry is sent.",
			"Add OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf to otel.env.")
	default:
		return fail("OTEL_EXPORTER_OTLP_PROTOCOL is "+p,
			"otelstarter installs the HTTP exporter (opentelemetry-exporter-otlp-proto-http), which needs http/protobuf.",
			"Set OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf in otel.env.")
	}
}

func checkEndpointURL(env map[string]string) Result {
	e := endpoint(env)
	u, err := url.Parse(e)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fail("OTEL_EXPORTER_OTLP_ENDPOINT is not a valid URL: "+e,
			"The exporter cannot send anywhere without an http:// or https:// address.",
			"Set it to your Collector's address, for example OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318.")
	}
	if strings.HasSuffix(u.Path, "/v1/traces") || strings.HasSuffix(u.Path, "/v1/metrics") || strings.HasSuffix(u.Path, "/v1/logs") {
		return fail("OTEL_EXPORTER_OTLP_ENDPOINT includes a signal path: "+e,
			"This setting is the base address; the SDK adds /v1/traces, /v1/metrics and /v1/logs itself, so requests go to the wrong place.",
			"Remove the /v1/... part from OTEL_EXPORTER_OTLP_ENDPOINT.")
	}
	return pass("Endpoint: " + e)
}

func checkDeprecated(env map[string]string) []Result {
	if _, ok := env["OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED"]; ok {
		return []Result{warn("otel.env sets the deprecated OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED",
			"It stops opentelemetry-instrumentation-logging from installing its own log handler, and will be removed from the SDK.",
			"Delete that line from otel.env. Logs are still exported without it.")}
	}
	return nil
}

// secretWords mark a setting whose value is probably a secret.
var secretWords = []string{"TOKEN", "SECRET", "PASSWORD", "PASSWD", "API_KEY", "APIKEY", "PRIVATE_KEY", "CREDENTIAL"}

// secretHeaderWords mark an OTLP header that carries credentials.
var secretHeaderWords = []string{"authorization", "api-key", "apikey", "token", "bearer", "secret"}

// IsSecret reports whether a setting probably holds a secret. doctor uses it
// to mask values in its report.
func IsSecret(name, value string) bool {
	upper := strings.ToUpper(name)
	for _, w := range secretWords {
		if strings.Contains(upper, w) {
			return true
		}
	}
	if strings.HasPrefix(upper, "OTEL_EXPORTER_OTLP_") && strings.HasSuffix(upper, "HEADERS") {
		lower := strings.ToLower(value)
		for _, w := range secretHeaderWords {
			if strings.Contains(lower, w) {
				return true
			}
		}
	}
	return false
}

// checkSecrets warns about secrets in otel.env, which is meant to be
// committed. Values are never printed.
func checkSecrets(env map[string]string) []Result {
	var results []Result
	for _, name := range SortedNames(env) {
		if IsSecret(name, env[name]) {
			results = append(results, warn(name+" in otel.env looks like a secret",
				"otel.env is meant to be committed to version control, so anything in it is visible to everyone with access to the repository.",
				"Remove it from otel.env and set it in your shell, your deployment platform's secret settings, or a secret manager."))
		}
	}
	return results
}
