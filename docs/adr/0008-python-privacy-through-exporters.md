# ADR 0008: Python cleans telemetry in its exporters

**Status:** Accepted
**Date:** 2026-10-07

## Context

ADR 0007 gave the Node.js SDK structured logs, redaction and query sanitising, and the maintainer asked for the same in Python, following [TELEMETRY_CONVENTIONS.md](../TELEMETRY_CONVENTIONS.md).

Node cleans spans in a span processor when they end. Python cannot. `Span.end()` freezes the attributes (`_attributes._immutable = True`) **before** any processor hook runs, including `on_end` and the private `_on_ending`. Database instrumentations such as psycopg2 set the query after the span starts, so cleaning at start is too early too.

## Decision

- **The SDK registers a trace exporter and a log exporter named `otelstarter`.** It does this through OpenTelemetry's entry points (`opentelemetry_traces_exporter`, `opentelemetry_logs_exporter`). Each one builds a **cleaned copy** of every span and log record and passes it to the OTLP HTTP/protobuf exporter. `init` selects them in `otel.env`:
  ```bash
  OTEL_TRACES_EXPORTER=otelstarter
  OTEL_LOGS_EXPORTER=otelstarter
  ```
- **An instrumentor switches the root logger's console output to the shared JSON format.** It is registered through the `opentelemetry_instrumentor` entry point, so `opentelemetry-instrument` loads it with no code changes. `OTELSTARTER_JSON_LOGS=false` opts out.
- **`sdk/conventions.json` holds test cases that both SDKs run,** so their rules cannot drift apart.
- **`validate` checks the combination:** it warns when an existing `otel.env` does not select the exporters, and fails when it does but the SDK is missing or older than 0.3.0. OpenTelemetry would then fail to configure and send nothing.

## Alternatives

- **A span processor:** impossible, because attributes are frozen before any hook runs.
- **Changing the frozen attributes in place:** relies on private internals that raise an error once frozen.
- **A custom distro or configurator:** a much larger surface to maintain than two exporters and an instrumentor.

## Consequences

- **The SDK becomes required for Python projects that use these exporters.** The package still depends only on `opentelemetry-api`; the exporters import the SDK and the OTLP exporter, which `init` already installs.
- **This also avoids a duplication that end-to-end testing found.** FastAPI 0.142's built-in telemetry adds its own OTLP exporter to existing providers when `OTEL_TRACES_EXPORTER=otlp`, and FastAPI does not deduplicate exporters. One request's spans arrived twice with `otlp` (6 against 3) and once with `otelstarter`. In exchange, FastAPI logs a warning; it is documented, along with `FastAPI(telemetry={"auto_configure": False})` to silence it.
- **Projects set up before 0.3.0 keep their old `otel.env`,** because `init` never overwrites it. `validate` tells them how to switch.
