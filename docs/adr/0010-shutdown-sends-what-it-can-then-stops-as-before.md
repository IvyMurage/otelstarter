# ADR 0010: Shutdown sends what it can, then stops as before

**Status:** Accepted

## Context

`compat/run.py` runs real Express, NestJS, Flask and FastAPI apps with and without otelstarter. On its first run, every app answered every request the same either way, but shutdown differed:

| App | Collector down: time to stop after SIGTERM |
|-----|-------------------------------------------|
| Node.js (Express, NestJS) | about 9 s, against 0 s without otelstarter |
| gunicorn, 2 workers | still running after 30 s |
| uvicorn, 2 workers | 20 s |

- **Why it was slow:** the exporters retry for up to their 10 s timeout. Python flushes traces, logs and metrics one after another, and stops retrying only once the exporter is shut down, which happens after the flush. Docker kills a container 10 s after SIGTERM, and Kubernetes after 30 s.
- **What a plain Python app loses:** stopped by SIGTERM, it ends without running exit handlers, so the spans and logs still in the batch processors are lost. That is up to 5 s of telemetry on every deploy.
- **Node.js exit status:** after flushing, the SDK exited with code 143 instead of being stopped by the signal.

## Decision

- **A 3-second budget for the final flush.**
  - **Node.js:** the SDK's shutdown races a 3-second timer.
  - **Python:** the three providers are shut down at the same time, in daemon threads, with one 3-second deadline. This runs from an exit handler registered after OpenTelemetry's, so it runs first, and theirs find the providers already shut down.
- **Stop exactly as without otelstarter.** If the app keeps the default SIGTERM behaviour, otelstarter sends the waiting telemetry, restores the default, and raises SIGTERM again. The process is then stopped by the signal, as before.
- **An app's own SIGTERM handler stays in charge.** This includes the handlers of gunicorn and uvicorn. The telemetry is then sent when the process exits.

## Alternatives

- **Lower `OTEL_EXPORTER_OTLP_TIMEOUT` in `otel.env`:** no code, but it would also cut ordinary exports short on a slow network, and Python's three sequential flushes would still add up.
- **Turn SIGTERM into `SystemExit` in Python:** this runs exit handlers too, but it raises an exception inside whatever code is running, and an `except BaseException` in the app could swallow it. Re-raising the signal changes nothing the app can observe.
- **Wait as long as the exporters want:** telemetry is never cut short, but deploys slow down and containers get killed.

## Consequences

- With the Collector down, a Node.js app stops in about 3 s, and a 2-worker gunicorn or uvicorn in about 6.5 s: the workers, then the master.
- When the Collector is reachable, the flush takes milliseconds, and the last batch is no longer lost on SIGTERM.
- Telemetry that cannot be sent within 3 seconds is dropped at shutdown.
