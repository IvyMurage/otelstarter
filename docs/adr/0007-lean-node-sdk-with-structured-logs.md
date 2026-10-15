# ADR 0007: A lean Node.js SDK with structured logs, redaction and query sanitising

**Status:** Accepted
**Date:** 2026-10-07

## Context

PR #15 instrumented Node.js apps with `@opentelemetry/auto-instrumentations-node`, which bundles about 40 instrumentations: +83 MB in `node_modules`. Testing also showed that its register script stops apps exiting on SIGTERM ([upstream #3808](https://github.com/open-telemetry/opentelemetry-js-contrib/issues/3808)), that `console.log` never reaches the backend, and that SQL values are recorded as written.

The maintainer's earlier internal SDK, built for NestJS services, showed a lighter approach. It loads only the instrumentations a service needs, writes structured JSON logs with trace context, redacts secrets and sanitises queries. The maintainer asked for the same principles, implemented fresh for this public project.

## Decision

The `otelstarter` npm package (`sdk/node`, TypeScript) provides:

- **`otelstarter/register`,** loaded with `--require`. It sets up traces, metrics and logs with the three OTLP HTTP/protobuf exporters, and loads only the instrumentations listed in `OTEL_NODE_ENABLED_INSTRUMENTATIONS`, plus http, undici, runtime-node and pino. A missing instrumentation package is reported, not fatal. After flushing on SIGTERM it exits, unless the app has its own SIGTERM handler.
- **`logger`:** structured JSON on stdout (pino), also sent to OpenTelemetry. It uses the fields in [TELEMETRY_CONVENTIONS.md](../TELEMETRY_CONVENTIONS.md).
- **Redaction and query sanitising,** applied to logs and to spans. Spans are cleaned when they end, by wrapping the exporting processor, because instrumentations such as pg set the query after the span starts.
- **`event()`:** domain events with the same rules as the Python SDK.

**Metrics are OpenTelemetry metrics, pushed over OTLP,** not a Prometheus `/metrics` endpoint, so there is one pipeline (maintainer's choice). **Logs go to stdout and to OpenTelemetry** (maintainer's choice). The Python SDK will follow the same conventions.

## Alternatives

- **Keep the all-in-one bundle:** standard, but more than twice the size, plus the SIGTERM and log gaps.
- **`@opentelemetry/sdk-node`:** less of our own code, but it pulls in every exporter including gRPC (+53 MB).
- **Prometheus `/metrics` and stdout-only logs** (the internal SDK's way): lighter still, but logs would not reach the local Grafana, and metrics would need per-app scrape configuration.

## Consequences

- An Express + pg app measured 45 MB of `node_modules` in total, against more than 83 MB for the bundle alone.
- otelstarter now owns a small SDK setup (`sdk.ts`) that must follow OpenTelemetry releases; experimental `0.x` packages are pinned to one minor version.
- `init` and `run` move from the bundle to `otelstarter` plus the detected `@opentelemetry/instrumentation-<name>` packages in a follow-up PR.

## Update, 2026-10-15

That follow-up is done. `init` pins `otelstarter` and one instrumentation package per detected library, each at a tested version built on the same `@opentelemetry/instrumentation` 0.223 as the SDK, so npm installs the core once (Express + pg: 46 MB). winston also gets `@opentelemetry/winston-transport`, without which its logs are not sent. `OTEL_NODE_RESOURCE_DETECTORS` is no longer written: the SDK chooses its detectors itself.

Projects set up with the bundle keep working: `run` uses the bundle's loader while the SDK is not installed, and `validate` warns instead of failing, so their CI does not break.
- Cleaning spans after they end updates the finished span's attribute object in place. That relies on the SDK not freezing it; a test covers it.
