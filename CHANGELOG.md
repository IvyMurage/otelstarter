# Changelog

All notable changes are recorded here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- Python `analyze` detects `pymongo` as MongoDB and recommends the official `opentelemetry-instrumentation-pymongo` package.

- `docs/PROJECT_GUIDE.md`: the whole project in plain English, including what every file does.
- For contributors:
  - a rewritten `CONTRIBUTING.md`, with setup and checks for each part and what happens after a pull request;
  - `docs/adding-a-language.md`;
  - pull request and issue templates;
  - `CODE_OF_CONDUCT.md`.

  The engineering guidelines now speak to every contributor, with a section on using AI assistants.

## [0.1.0] - 2026-10-29

The first release of the CLI. The SDKs are released separately: npm `otelstarter` 0.1.4 and PyPI `otelstarter` 0.3.1.

### Added

- Ready-made CLI downloads. Pushing a `v*` tag builds `otelstarter` for Linux, macOS and Windows (amd64 and arm64) and publishes them, with checksums, as a GitHub release. The README shows a one-line install.
- `otelstarter version` reports the release version when installed with `go install …@vX.Y.Z`. Before, it printed `dev`.
- `compat/run.py`, run in CI: real Express, NestJS 11 and 12, Flask and FastAPI apps, also under gunicorn and uvicorn with workers, must answer exactly the same with otelstarter as without it. They must also stop cleanly with the Collector up or down, send traces, logs and metrics, and leak no email address or password. See `compat/README.md`.
- Release workflow for the Node.js SDK: pushing a `sdk-node-vX.Y.Z` tag tests and publishes it to npm through Trusted Publishing. The first version is published by hand; see `sdk/node/RELEASING.md`. Node.js SDK 0.1.1 is the first release through the workflow, with no code changes.
- Python SDK 0.3.0: structured JSON logs (switched on automatically under `opentelemetry-instrument`), and `otelstarter` trace and log exporters that redact secrets and replace SQL values with `?` before export. `init` selects them in `otel.env` (`OTEL_TRACES_EXPORTER`/`OTEL_LOGS_EXPORTER=otelstarter`) and pins `otelstarter==0.3.0`. `validate` warns when an existing `otel.env` does not use them, and fails when it does but the SDK is missing or too old.
- `sdk/conventions.json`: shared test cases that both SDKs run, so their redaction and sanitising cannot drift apart.
- Node.js SDK (`sdk/node`, npm package `otelstarter`, published as 0.1.0): `otelstarter/register` sets up traces, metrics and logs with only the needed instrumentations and still exits on SIGTERM; `logger` writes structured JSON logs with trace context; secrets are redacted and SQL values sanitised in logs and spans; `event()` records domain events. Conventions shared with Python are in `docs/TELEMETRY_CONVENTIONS.md`.
- `init`, `run`, `validate` and `doctor` support Node.js (CommonJS apps). `init` writes `otel.env` with only the detected instrumentations enabled and prints the exact `npm`/`yarn`/`pnpm`/`bun` install command. `run` adds the OpenTelemetry loader to `NODE_OPTIONS`, keeping existing options. `validate` checks the Node.js version, that the package is installed, and warns about missing logging libraries, ES modules, Fastify and Prisma. Tested end to end with Express and NestJS.
- The local Collector drops Express's built-in `query` and `expressInit` middleware spans.
- `analyze` supports Node.js: frameworks, libraries, logging library, package manager and ES-module apps from `package.json`. It recommends `@opentelemetry/auto-instrumentations-node` with only the detected instrumentations enabled.
- `validate` warns when the `otelstarter` SDK is pinned to an older version than the one `init` now pins.
- `docs/performance.md`: first measured footprint and overhead baseline, plus how to sample traces in production.
- `docs/run.md`: developing with reload (`uvicorn --reload` works under instrumentation; Flask's debug reloader does not), and sharing one stack between terminals.
- `docs/quickstart.md`, and install instructions in the README, including adding `~/go/bin` to `PATH`.
- `run` shares a stack that is already running (for example `otelstarter run` in another terminal), instead of removing its containers.
- `validate` and `doctor` check the Python version that will run the app (3.10+ required). It prefers the virtual environment's interpreter, and runs in the project folder so pyenv's `.python-version` applies.
- Python SDK 0.2.0: with `OTELSTARTER_CONSOLE_EVENTS=true` (set by `run`), each domain event also prints a line to the terminal. It is not exported.
- `analyze` recommends, and `init` adds, the `otelstarter` domain-event SDK (`otelstarter==0.1.0`) alongside the OpenTelemetry packages.
- Release workflow for the Python SDK: pushing a `sdk-python-vX.Y.Z` tag tests, builds and publishes it to PyPI through Trusted Publishing. See `sdk/python/RELEASING.md`.
- `validate` and `doctor` check that OpenTelemetry packages pinned with `==` belong together: all beta packages (`0.NNbN`) at one version, all stable ones (`1.NN.N`) at another.
- Python domain-event SDK (`sdk/python`, package `otelstarter`): `observability.event(name, domain=, operation=, outcome=, entity_type=, entity_id=)` emits one OpenTelemetry log-based event linked to the current trace. No payloads, never raises, depends only on `opentelemetry-api`. Published to PyPI as `otelstarter` 0.1.0.
- `otelstarter validate [--offline] [path]` and `otelstarter doctor [path]`: one set of checks (project, `otel.env`, protocol, endpoint, deprecated settings, secrets, Collector reachability, declared packages, launcher on PATH), each problem with why and fix. `validate` exits 1 on failure; `doctor` adds an environment report with secrets hidden.
- `otelstarter run [-- <app command>]`: starts a local OpenTelemetry Collector and Grafana LGTM (Tempo, Loki, Prometheus) in Docker, and optionally your app with `otel.env` loaded and `opentelemetry-instrument` added. Ctrl+C stops everything. The Collector drops FastAPI's per-request ASGI send/receive spans.
- `otelstarter init [path]` for Python projects: creates `otel.env` with standard OpenTelemetry settings and adds pinned packages to `requirements.txt` (pip), or prints the exact `uv add` / `poetry add` / `pdm add` / `pipenv install` command. Shows the plan first, asks before writing, supports `--dry-run` and `--yes`, and is safe to run repeatedly.
- `otelstarter analyze` recommends OpenTelemetry instrumentation for Python projects: four base packages plus one per detected framework or library, marks those already installed, and shows the `opentelemetry-instrument` start command.
- `otelstarter analyze [path]`: detects Python, Node.js and Go projects. For Python it reports the package manager, framework (FastAPI, Flask, Django), observability-relevant dependencies and installed OpenTelemetry packages.
- CLI skeleton with `init`, `analyze`, `run`, `validate`, `doctor` and `version` commands.

### Changed

- Node.js SDK 0.1.4 and Python SDK 0.3.1, from the new compatibility check:
  - **Node.js:** logs from the app's own pino, winston or bunyan logger are redacted too. Before, `log.info({ password }, 'login')` reached the backend with the password.
  - **Both, at shutdown:** the last telemetry is sent within at most 3 seconds, and SIGTERM then stops the app exactly as without otelstarter. Before, with the Collector down, a Node.js app took about 9 s to stop and a gunicorn or uvicorn worker 20–30 s. A Python app stopped by SIGTERM also lost its last batch.
  - `init` pins `otelstarter@0.1.4` and `otelstarter==0.3.1`.
- Node.js SDK 0.1.3: **ES-module apps** (`"type": "module"`, `.mjs`) are fully instrumented with the same `--require otelstarter/register`, on Node.js 22.14 or newer. Before, an ES-module Express app got HTTP spans but no Express route spans, and requests were named only `GET`. The import hook is only switched on for ES-module apps, so CommonJS apps keep their startup time and memory. `validate` warns when an ES-module app runs on an older Node.js, and `analyze` no longer says ES modules are unsupported. `init` pins `otelstarter@0.1.3`.
- Node.js SDK 0.1.2: `logger` also accepts pino's argument order, `logger.info({ orderId }, 'order placed')` and `logger.error(err, 'payment failed')`. Before, the fields were silently dropped. `init` pins `otelstarter@0.1.2`.
- **Node.js:** `init` now installs the `otelstarter` SDK plus one `@opentelemetry/instrumentation-<name>` package per detected library, instead of `@opentelemetry/api` and the 80 MB `@opentelemetry/auto-instrumentations-node` bundle. `run` loads `otelstarter/register`, so Node.js apps get redaction, query sanitising, JSON logs and a normal exit on SIGTERM. Projects still on the bundle keep working: `run` uses its loader until the SDK is installed, and `validate` warns, without failing, until they move. `otel.env` no longer sets `OTEL_NODE_RESOURCE_DETECTORS`. `validate` also warns when a declared instrumentation package is not in `node_modules`.
- Python `otel.env` no longer sets `OTEL_PYTHON_LOG_CORRELATION`: the JSON logs carry `trace_id` and `span_id`. With plain `otlp` exporters, recent FastAPI versions add their own exporter and send every span twice; the new `otelstarter` exporters avoid that.
- When the app exits by itself, `run` keeps the stack running in a terminal, so the crash's telemetry stays in Grafana. Without a terminal it still stops.
- **Python SDK 0.2.0 (breaking):** problems with an event are logged as warnings on the `otelstarter` logger instead of being raised as `OtelStarterWarning`, which is removed. They now look like the app's own logs and reach the backend.
- `init` pins `otelstarter==0.2.0`.
- `otel.env` no longer sets `OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED`. It is deprecated and stopped the logging instrumentation from installing its own handler. If an earlier `init` wrote it, delete that line.
- `init` now suggests `otelstarter run -- <command>` as the start command.
- The recommended exporter is now `opentelemetry-exporter-otlp-proto-http` instead of `opentelemetry-exporter-otlp`, which avoids installing `grpcio`.
- Building from source needs Go 1.25 or newer (was 1.24), for `os.Root.Rename`.
