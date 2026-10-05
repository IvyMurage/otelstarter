# ADR 0003: init writes standard settings and never runs tools

**Status:** Accepted
**Date:** 2026-10-05

## Context

`otelstarter init` is the first command that changes a user's project. The requirements say generated configuration must be deterministic, readable and validated, must not overwrite user configuration without confirmation, and that running `init` repeatedly must be safe. OpenTelemetry's Python packages also require each other at exact versions.

## Decision

1. **Configuration is a plain `otel.env` of standard `OTEL_*` variables.** There is no `otelstarter.yaml`: OpenTelemetry already defines these settings, every SDK reads them, and Docker and Kubernetes load env files natively. This adds no YAML dependency and no new format.
2. **Install the HTTP-only OTLP exporter** (`opentelemetry-exporter-otlp-proto-http`) and set `OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf`. The combined `opentelemetry-exporter-otlp` pulls in `grpcio`, a large compiled package.
3. **Never run package managers or rewrite TOML.** For pip, `init` edits `requirements.txt` inside a marked block. For uv, Poetry, PDM and Pipenv it prints the exact command. Rewriting `pyproject.toml` from Go would lose comments and formatting.
4. **Pin a tested version set** built into otelstarter (`0.66b0` contrib, `1.45.0` SDK and exporter), updated deliberately after an end-to-end test.
5. **Plan, then apply.** `setup.BuildPlan` only reads; `setup.Apply` is the only code that writes to a user's project. Apply checks every file before writing any, refuses symbolic links, aborts if a file changed since the preview, and writes through a temporary file plus rename.
6. **Raise the minimum Go version to 1.25** for `os.Root.Rename`, so the atomic rename stays inside the project folder. Go supports only its two newest releases (1.26 and 1.27 at the time of writing), so 1.24 was already unsupported.

## Alternatives

- **`otelstarter.yaml` plus generated env:** a record of what otelstarter set up, but it duplicates OpenTelemetry's own settings and needs a YAML library. It can be added if `validate` or `run` need it.
- **Run `uv add` and the like after confirmation:** smoother, but otelstarter would start executing tools inside the project, which ADR 0002 avoids.
- **Leave packages unpinned:** simpler, but the result depends on the day it runs, and mismatched versions fail at import time.
- **Keep the full OTLP exporter:** it works without `otel.env`, but every install pays for `grpcio`.

## Consequences

- A project set up by `init` works with any tool that understands OpenTelemetry environment variables.
- Starting the app without loading `otel.env` disables telemetry, with a traceback but without breaking the app. `otelstarter doctor` should detect this.
- Users of uv, Poetry, PDM and Pipenv copy and run one command.
- Every otelstarter release that bumps the version set needs an end-to-end test.
- Contributors need Go 1.25 or newer.
