# Changelog

All notable changes are recorded here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- Python domain-event SDK (`sdk/python`, package `otelstarter`): `observability.event(name, domain=, operation=, outcome=, entity_type=, entity_id=)` emits one OpenTelemetry log-based event linked to the current trace. No payloads, never raises, depends only on `opentelemetry-api`. Not yet on PyPI.
- `otelstarter validate [--offline] [path]` and `otelstarter doctor [path]`: one set of checks (project, `otel.env`, protocol, endpoint, deprecated settings, secrets, Collector reachability, declared packages, launcher on PATH), each problem with why and fix. `validate` exits 1 on failure; `doctor` adds an environment report with secrets hidden.
- `otelstarter run [-- <app command>]`: starts a local OpenTelemetry Collector and Grafana LGTM (Tempo, Loki, Prometheus) in Docker, and optionally your app with `otel.env` loaded and `opentelemetry-instrument` added. Ctrl+C stops everything. The Collector drops FastAPI's per-request ASGI send/receive spans.
- `otelstarter init [path]` for Python projects: creates `otel.env` with standard OpenTelemetry settings and adds pinned packages to `requirements.txt` (pip), or prints the exact `uv add` / `poetry add` / `pdm add` / `pipenv install` command. Shows the plan first, asks before writing, supports `--dry-run` and `--yes`, and is safe to run repeatedly.
- `otelstarter analyze` recommends OpenTelemetry instrumentation for Python projects: four base packages plus one per detected framework or library, marks those already installed, and shows the `opentelemetry-instrument` start command.
- `otelstarter analyze [path]`: detects Python, Node.js and Go projects. For Python it reports the package manager, framework (FastAPI, Flask, Django), observability-relevant dependencies and installed OpenTelemetry packages.
- CLI skeleton with `init`, `analyze`, `run`, `validate`, `doctor` and `version` commands.

### Changed

- `otel.env` no longer sets `OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED`. It is deprecated and stopped the logging instrumentation from installing its own handler. If an earlier `init` wrote it, delete that line.
- `init` now suggests `otelstarter run -- <command>` as the start command.
- The recommended exporter is now `opentelemetry-exporter-otlp-proto-http` instead of `opentelemetry-exporter-otlp`, which avoids installing `grpcio`.
- Building from source needs Go 1.25 or newer (was 1.24), for `os.Root.Rename`.
