# otelstarter run

## What it does

`otelstarter run` starts a local observability stack in Docker, and optionally your app:

```text
your app ──OTLP :4318──▶ OpenTelemetry Collector ──▶ Grafana LGTM
                         (drops noise, batches)       Tempo  · traces
                                                      Loki   · logs
                                                      Prometheus · metrics
                                                      Grafana UI :3000
```

It runs in the foreground. Press **Ctrl+C** to stop your app and the stack.

## Why

After `otelstarter init`, something has to receive the telemetry. `run` gives you a working Collector and a Grafana with traces, logs and metrics in one command, without writing any Docker or Collector configuration.

## Usage

```bash
otelstarter run                              # start only the stack
otelstarter run -- uvicorn main:app          # start the stack, then your app
otelstarter run -- flask run --port 8000     # anything after -- is your app's command
```

With `--`, run:

1. Reads `otel.env` from the current folder. Variables already set in your shell win, so `OTEL_SERVICE_NAME=test otelstarter run -- …` works.
2. For Python, puts `opentelemetry-instrument` in front of your command, unless it is already in it.
3. Starts the stack and waits until it is ready, so no early telemetry is lost.
4. Starts your app. When the app exits, or you press Ctrl+C, the stack is stopped.

Activate your virtual environment first, so `opentelemetry-instrument` and your app's command are found. With uv, put `opentelemetry-instrument` in the command yourself: `otelstarter run -- uv run opentelemetry-instrument uvicorn main:app`.

## Example

```text
$ otelstarter run -- uvicorn main:app
Starting the local observability stack. The first run downloads about 1 GB of images.
...
Ready.
  Collector  http://localhost:4318   (OTLP; your app sends here)
  Grafana    http://localhost:3000   (Explore: Tempo for traces, Loki for logs, Prometheus for metrics)
  Files      /home/dev/.cache/otelstarter

Starting your app: opentelemetry-instrument uvicorn main:app
Press Ctrl+C to stop the app and the stack.
```

Open Grafana, go to **Explore** and pick:

- **Tempo** for traces. Search by service name.
- **Loki** for logs. Each log line carries `trace_id` and `span_id`.
- **Prometheus** for metrics, such as `http_server_duration_milliseconds_count`.

## What runs

| Container | Image | Ports (127.0.0.1 only) |
|-----------|-------|------------------------|
| Collector | `otel/opentelemetry-collector:0.162.0` | 4317 (OTLP gRPC), 4318 (OTLP HTTP), 13133 (health) |
| Backend | `grafana/otel-lgtm:0.35.0` | 3000 (Grafana) |

The Collector config:
- limits its memory use
- batches data
- drops the internal "http send" and "http receive" spans that FastAPI's instrumentation adds to every request, which carry no diagnostic value

Your app only talks to the Collector, so a different backend can replace Grafana without touching the app.

The generated `compose.yaml` and `collector.yaml` live in your cache folder (shown as `Files`) and are rewritten on every run.

## Limitations

- **Needs Docker with Compose v2.** The first run downloads about 1 GB, mostly the Grafana LGTM image.
- **Ports 3000, 4317, 4318 and 13133 must be free.** run checks them first and says which one is busy.
- **Telemetry is discarded when the stack stops.** It is for local development, not for keeping data.
- **Python apps send metrics every 60 seconds by default,** so metrics appear in Grafana up to a minute after traffic.
- **Tempo's API in `grafana/otel-lgtm:0.35.0` can return each span of a trace twice.** The Collector and Tempo both count every span once; the doubling happens when the trace is read back. Grafana's trace view is not affected in normal use.
- **Only one stack runs per machine.** Several apps can share it; `OTEL_SERVICE_NAME` tells them apart.

## Security

- **Ports are bound to 127.0.0.1 only.** Grafana in this image has no login, so it must not be reachable from the network.
- **Docker is run with a fixed list of arguments, never through a shell.** Your app's command is run directly, also without a shell.
- **Images are pinned to exact versions.**
