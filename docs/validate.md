# otelstarter validate and doctor

## What they do

Both commands run the same checks on a project set up with `otelstarter init`, and explain every problem: what is wrong, why it matters, and how to fix it.

- **`validate`** is the checklist. It exits with code `1` when any check fails, so it can gate CI.
- **`doctor`** runs the same checks, then prints an environment report for troubleshooting or a bug report. Secrets in the report are hidden.

Neither changes anything.

## Usage

```bash
otelstarter validate               # current folder
otelstarter validate ./my-app
otelstarter validate --offline     # skip the Collector check (CI, no Collector running)
otelstarter doctor
```

## Example

```text
$ otelstarter validate
  ✓ Python project (FastAPI, pip)
  ✓ otel.env
  ✓ Service name: shop
  ✗ OTEL_EXPORTER_OTLP_PROTOCOL is not set
      Why: The Python SDK then uses gRPC, but only the HTTP exporter is installed, so no telemetry is sent.
      Fix: Add OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf to otel.env.
  ✓ Endpoint: http://localhost:4318
  ✓ Collector at http://localhost:4318 accepts traces, metrics and logs
  ✓ All 7 recommended packages are declared
  ✓ opentelemetry-instrument: /home/dev/shop/.venv/bin/opentelemetry-instrument

1 problem, 0 warnings.
```

`✓` passed, `!` warning (exit code stays `0`), `✗` problem (exit code `1`).

## The checks

| Check | Fails or warns when |
|-------|---------------------|
| Python project | No supported project, or not a Python one |
| `otel.env` | Missing or has a malformed line |
| Service name | `OTEL_SERVICE_NAME` is empty |
| Protocol | `OTEL_EXPORTER_OTLP_PROTOCOL` is not `http/protobuf`. Without it the SDK looks for the gRPC exporter and sends nothing |
| Endpoint | Not an `http(s)://` URL, or it includes `/v1/traces` and the like, which the SDK adds itself |
| Deprecated settings | `OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED` is set (warning) |
| Secrets | A setting looks like a secret: names containing `TOKEN`, `SECRET`, `PASSWORD`, `API_KEY` and similar, or OTLP headers carrying `Authorization` or an API key (warning) |
| Collector | The endpoint does not accept OTLP on `/v1/traces`, `/v1/metrics` and `/v1/logs`. Skipped with `--offline` |
| Packages | `opentelemetry-distro` or the OTLP exporter is not declared (problem); another recommended instrumentation is not declared (warning) |
| Versions | OpenTelemetry packages pinned with `==` disagree within a version family: beta packages (`0.NNbN`: distro, instrumentation) must all match, as must stable ones (`1.NN.N`: API, SDK, exporters) |
| Python | The Python that will run the app is older than 3.10. That is the one next to `opentelemetry-instrument` if found, else `python3` or `python` on PATH. It runs `--version` in the project folder, so pyenv's `.python-version` applies |
| SDK version | `otelstarter` is pinned to an older version than the one this otelstarter pins (warning). `init` adds missing packages but does not change existing pins |
| Node.js version | The `node` on PATH is older than 18.19 (or 20.0–20.5). Runs `node --version` in the project folder, so `.nvmrc` with nvm or fnm applies |
| Node.js packages installed | Neither `otelstarter` nor the older bundle is in `node_modules`, or a declared instrumentation package is not, usually because `npm install` has not run (warning) |
| Node.js advice | No logging library (console.log is not exported), ES-module app, Fastify/Prisma, which need their own instrumentation, or the app still uses `@opentelemetry/auto-instrumentations-node` (warnings). While it does, a missing `otelstarter` is a warning, not a problem, because the bundle still sends telemetry |
| Launcher | `opentelemetry-instrument` is not on `PATH`, usually because the virtual environment is not active (warning) |

The Collector check sends an **empty** export request (`{}`) for each signal. A Collector answers `200` and stores nothing, so it adds no test data to your backend.

## doctor's environment report

```text
Environment (secrets are hidden, so this is safe to paste into a bug report):
  otelstarter:               v0.1.0
  Go runtime:                go1.27.1
  OS:                        linux/amd64
  Docker:                    29.8.2
  Docker Compose:            5.6.0
  Local stack:               running
  Ports:                     4317 in use, 4318 in use, 3000 in use, 13133 in use (the local stack uses them)
  opentelemetry-instrument:  /home/dev/shop/.venv/bin/opentelemetry-instrument

otel.env:
  OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
  OTEL_EXPORTER_OTLP_HEADERS=*** (hidden)
  ...
```

## Limitations

- **Python and Node.js projects.** The Python-only checks (protocol, version families, launcher, Python version) do not run for Node.js, and the Node.js checks do not run for Python.
- **Packages are checked as declared in the manifest, not as installed.** otelstarter never runs pip. The launcher check is the hint that they are installed in the active environment.
- **Only exact `==` pins are compared.** Ranges such as `>=1.40` and Poetry's `^` versions are not checked.
- **Signal-specific endpoints** (`OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` and so on) are not checked.
- **Secret detection is a heuristic.** It catches common names and headers, not every secret.
