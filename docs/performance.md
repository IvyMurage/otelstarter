# Performance and footprint

What otelstarter costs, measured rather than estimated. These are **first baseline numbers** from one machine and one run, enough to show orders of magnitude. They are not a benchmark to quote. Requirements section 11 asks for proper benchmarks (CPU, memory, latency, throughput) before making performance claims; this page is where they will live.

## Summary

| Part | Cost | Where it runs |
|------|------|---------------|
| CLI (`otelstarter`) | 9.6 MB binary (2.8 MB compressed), 1 Go dependency, about 0.01 s and 7 MB RAM per command | Your machine, during setup |
| Python SDK (`otelstarter` on PyPI) | 9.5 KB wheel; depends only on `opentelemetry-api` | Inside your app |
| Packages `init` adds | About +14 MB on disk and +23 packages, mostly OpenTelemetry's own (protobuf, SDK, semantic conventions) | Inside your app |
| Instrumentation at runtime | About +19 MB RAM per process and about +1 ms CPU per request in the test below | Inside your app |
| Local stack (`run`) | 3.8 GB of images (about 1 GB download); about 715 MB RAM (Grafana LGTM about 686 MB, Collector about 28 MB) | Your machine only, never in production |

## Runtime overhead

**Setup** (2026-10-06):
- Machine: 8-core laptop, Linux.
- App: Python 3.11, FastAPI with uvicorn (1 worker), with the packages from `otelstarter init`.
- Endpoint: `GET /inventory`, which only returns `{"available": 42}`.
- Load: ApacheBench, 3,000 requests at 10 concurrent, after 300 warm-up requests. Each variant ran twice; the runs agreed within 3%.
- Instrumented runs sent all signals to the local Collector from `otelstarter run`.

| | Requests/s | Mean latency | Process memory |
|---|---|---|---|
| Plain | about 2,000 | 4.9 ms | 43 MB |
| Instrumented | about 670 | 15 ms | 62 MB |

**How to read this.** The endpoint does no work, so OpenTelemetry's per-request work is almost everything the server does. That makes this close to a worst case. Server time per request rose from about 0.5 ms to about 1.5 ms: roughly **1 ms of CPU per request**.
- **A typical endpoint** spends tens of milliseconds on a database or another service, so 1 ms is a small share.
- **A trivial endpoint with very high traffic** feels it much more.

**Part of that cost is avoidable.** FastAPI's instrumentation creates two internal "http send/receive" spans per request. The local Collector drops them before storage, but the app still spends CPU creating them.

## Reducing the cost in production

**Sample traces.** Record a share of traces instead of all of them, using standard OpenTelemetry settings:

```bash
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1     # keep 10% of traces
```

- **`parentbased`** keeps decisions consistent across services: if an upstream service sampled a request, this one records it too.
- **Metrics are not affected,** so request rates, errors and latencies stay complete.
- **Logs keep their trace IDs,** but some will point at traces that were not kept.

`otel.env` from `otelstarter init` records everything, which suits local development. Choose a sampling rate per environment when you deploy.

## How to reproduce

```bash
otelstarter run                     # terminal 1: the local stack
# terminal 2, in an app set up with otelstarter init:
uvicorn main:app --port 8791 &                                        # plain
ab -n 3000 -c 10 http://127.0.0.1:8791/inventory
set -a && . ./otel.env && set +a
opentelemetry-instrument uvicorn main:app --port 8791 &               # instrumented
ab -n 3000 -c 10 http://127.0.0.1:8791/inventory
```

Warm each variant up first, for example with `ab -n 300`, and run each more than once.
