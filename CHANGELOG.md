# Changelog

All notable changes are recorded here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- Node.js SDK (`sdk/node`, npm package `otelstarter`, not yet published): `otelstarter/register` sets up traces, metrics and logs with only the needed instrumentations and still exits on SIGTERM; `logger` writes structured JSON logs with trace context; secrets are redacted and SQL values sanitised in logs and spans; `event()` records domain events. Conventions shared with Python are in `docs/TELEMETRY_CONVENTIONS.md`.
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

- When the app exits by itself, `run` keeps the stack running in a terminal, so the crash's telemetry stays in Grafana. Without a terminal it still stops.
- **Python SDK 0.2.0 (breaking):** problems with an event are logged as warnings on the `otelstarter` logger instead of being raised as `OtelStarterWarning`, which is removed. They now look like the app's own logs and reach the backend.
- `init` pins `otelstarter==0.2.0`.
- `otel.env` no longer sets `OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED`. It is deprecated and stopped the logging instrumentation from installing its own handler. If an earlier `init` wrote it, delete that line.
- `init` now suggests `otelstarter run -- <command>` as the start command.
- The recommended exporter is now `opentelemetry-exporter-otlp-proto-http` instead of `opentelemetry-exporter-otlp`, which avoids installing `grpcio`.
- Building from source needs Go 1.25 or newer (was 1.24), for `os.Root.Rename`.
