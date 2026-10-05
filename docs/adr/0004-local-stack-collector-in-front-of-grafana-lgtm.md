# ADR 0004: Local stack is a Collector in front of Grafana LGTM

**Status:** Accepted
**Date:** 2026-10-05

## Context

`otelstarter run` must give developers somewhere to see their telemetry (requirements section 13: OTEL Collector, local environment, basic dashboard) while keeping OTelStarter backend agnostic and the Collector responsible for processing and filtering (architecture section 4).

## Decision

1. **Two containers, started with Docker Compose:**
   - the core `otel/opentelemetry-collector` image (38 MB, which includes the `filter` processor)
   - `grafana/otel-lgtm`: Grafana, Tempo, Loki and Prometheus in one image

   Apps send only to our Collector, so the backend can be swapped without touching apps.
2. **The Collector config is ours.** It limits memory, batches, and drops FastAPI's per-request ASGI `send`/`receive` spans. Zero-code instrumentation cannot exclude these, because `exclude_spans` is a code-only option.
3. **Foreground lifecycle.** `run` stays open; Ctrl+C stops the app and the stack. There is no separate stop command.
4. **`run -- <command>` starts the app.** It loads `otel.env` (shell variables win), adds `opentelemetry-instrument` for Python, waits for the stack to be ready, then starts the app. This removes the "started without otel.env" failure mode found in Phase 4.
5. **Generated files live in the user cache folder,** not the project, and are rewritten on every run.
6. **Ports are bound to 127.0.0.1 only,** and images are pinned. Docker is invoked with fixed arguments and no shell.
7. **`init` no longer writes `OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED`.** The variable is deprecated, and setting it stops `opentelemetry-instrumentation-logging` from installing its own log handler. Verified: without it, logs are still exported with trace IDs.

## Alternatives

- **OpenObserve:** lighter (214 MB), but its current release is a release candidate and it needs a login.
- **Jaeger:** traces only.
- **Collector debug output only:** no dashboard.
- **Background stack with `run --stop`:** keeps data across app restarts, but adds a command and leaves containers running when forgotten.
- **Files in the project folder:** easier to edit, but clutters the project and needs init-style confirmation on every run.
- **Sending straight to the LGTM image's built-in Collector:** one container fewer, but filtering would then depend on that image's internal config, tying OTelStarter to one backend.

## Consequences

- One command gives traces, logs and metrics in Grafana, with logs linked to traces by `trace_id`.
- The first run downloads about 1 GB; later runs start in seconds.
- Telemetry is lost when the stack stops.
- Image versions must be bumped deliberately. CI validates the Collector config with the pinned Collector image.
