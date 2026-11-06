package python

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/IvyMurage/otelstarter/internal/project"
)

// basePackages are needed by every instrumented Python service, whatever
// libraries it uses.
var basePackages = []project.Instrumentation{
	{Package: "opentelemetry-distro", Purpose: "SDK and the opentelemetry-instrument launcher"},
	// HTTP only: the gRPC exporter pulls in grpcio, a large compiled package.
	// otel.env sets OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf to match.
	{Package: "opentelemetry-exporter-otlp-proto-http", Purpose: "Sends telemetry to an OpenTelemetry Collector over HTTP"},
	{Package: "opentelemetry-instrumentation-logging", Purpose: "Adds trace and span IDs to log records"},
	{Package: "opentelemetry-instrumentation-system-metrics", Purpose: "Runtime metrics: CPU, memory, garbage collection"},
	{Package: "otelstarter", Purpose: "Domain events: observability.event(\"order.created\", ...)"},
}

var categoryPurpose = map[string]string{
	"MongoDB":     "Database queries",
	"PostgreSQL":  "Database queries",
	"MySQL":       "Database queries",
	"Redis":       "Redis commands",
	"Kafka":       "Kafka messages",
	"RabbitMQ":    "RabbitMQ messages",
	"HTTP client": "Outgoing HTTP requests",
	"ORM":         "ORM queries",
	"Task queue":  "Background tasks",
}

// recommend returns the OpenTelemetry packages for the detected package
// names, marking those already installed. It also returns a note for each
// detected package that has no official instrumentation.
func recommend(names, declared []string) ([]project.Instrumentation, []string) {
	// Several detected packages can share one instrumentation package, for
	// example SQLAlchemy and SQLModel. usedBy collects them per package.
	usedBy := map[string][]string{}
	purpose := map[string]string{}
	var notes []string

	for _, name := range uniqueSorted(names) {
		pkg, ok := knownPackages[name]
		if !ok {
			continue
		}
		if pkg.instrumentation == "" {
			notes = append(notes, fmt.Sprintf(
				"%s has no official OpenTelemetry instrumentation, so its calls will not appear in traces.", name))
			continue
		}
		usedBy[pkg.instrumentation] = append(usedBy[pkg.instrumentation], name)
		purpose[pkg.instrumentation] = purposeOf(pkg)
	}

	// Clone so that setting Installed below never changes basePackages itself.
	recs := slices.Clone(basePackages)
	for _, p := range slices.Sorted(maps.Keys(usedBy)) {
		recs = append(recs, project.Instrumentation{
			Package: p,
			Purpose: fmt.Sprintf("%s (%s)", purpose[p], strings.Join(usedBy[p], ", ")),
		})
	}
	for i := range recs {
		recs[i].Installed = slices.Contains(declared, recs[i].Package)
	}
	return recs, notes
}

func purposeOf(pkg knownPackage) string {
	if pkg.framework != "" {
		return "Incoming HTTP requests: traces and request metrics"
	}
	return categoryPurpose[pkg.category]
}

// exampleStartCommand shows a typical start command for the framework, so the
// user sees where opentelemetry-instrument goes.
func exampleStartCommand(frameworks []string) string {
	switch {
	case slices.Contains(frameworks, "FastAPI"):
		return "uvicorn main:app"
	case slices.Contains(frameworks, "Flask"):
		return "flask run"
	case slices.Contains(frameworks, "Django"):
		return "python manage.py runserver"
	default:
		return "python main.py"
	}
}

// runNotes are known problems with running under opentelemetry-instrument,
// taken from https://opentelemetry.io/docs/zero-code/python/troubleshooting/.
func runNotes(frameworks []string) []string {
	notes := []string{"OpenTelemetry's Python packages need Python 3.10 or newer."}
	if slices.Contains(frameworks, "FastAPI") {
		notes = append(notes, "Recent FastAPI versions log \"FastAPI automatic telemetry configuration failed\" "+
			"when otel.env uses otelstarter's exporter. It is harmless: otelstarter already exports, and the "+
			"message means FastAPI did not add a second exporter (with otlp it would send every span twice). "+
			"To silence it, create the app with FastAPI(telemetry={\"auto_configure\": False}).")
	}
	if slices.Contains(frameworks, "Flask") {
		notes = append(notes, "Flask's debug mode starts a reloader that stops instrumentation from loading. "+
			"Run without debug mode, or pass use_reloader=False to app.run().")
	}
	return notes
}

func uniqueSorted(names []string) []string {
	set := map[string]bool{}
	for _, n := range names {
		if n != "" {
			set[n] = true
		}
	}
	return slices.Sorted(maps.Keys(set))
}
