# Changelog

All notable changes are recorded here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- `otelstarter init [path]` for Python projects: creates `otel.env` with standard OpenTelemetry settings and adds pinned packages to `requirements.txt` (pip), or prints the exact `uv add` / `poetry add` / `pdm add` / `pipenv install` command. Shows the plan first, asks before writing, supports `--dry-run` and `--yes`, and is safe to run repeatedly.
- `otelstarter analyze` recommends OpenTelemetry instrumentation for Python projects: four base packages plus one per detected framework or library, marks those already installed, and shows the `opentelemetry-instrument` start command.
- `otelstarter analyze [path]`: detects Python, Node.js and Go projects. For Python it reports the package manager, framework (FastAPI, Flask, Django), observability-relevant dependencies and installed OpenTelemetry packages.
- CLI skeleton with `init`, `analyze`, `run`, `validate`, `doctor` and `version` commands.

### Changed

- The recommended exporter is now `opentelemetry-exporter-otlp-proto-http` instead of `opentelemetry-exporter-otlp`, which avoids installing `grpcio`.
- Building from source needs Go 1.25 or newer (was 1.24), for `os.Root.Rename`.
