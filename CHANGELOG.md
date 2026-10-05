# Changelog

All notable changes are recorded here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- `otelstarter analyze` recommends OpenTelemetry instrumentation for Python projects: four base packages plus one per detected framework or library, marks those already installed, and shows the `opentelemetry-instrument` start command.
- `otelstarter analyze [path]`: detects Python, Node.js and Go projects. For Python it reports the package manager, framework (FastAPI, Flask, Django), observability-relevant dependencies and installed OpenTelemetry packages.
- CLI skeleton with `init`, `analyze`, `run`, `validate`, `doctor` and `version` commands. Only `version` and `help` work so far.
