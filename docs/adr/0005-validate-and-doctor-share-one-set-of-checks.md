# ADR 0005: validate and doctor share one set of checks

**Status:** Accepted
**Date:** 2026-10-05

## Context

The requirements need `otelstarter validate` to cover dependencies, configuration, instrumentation, the Collector, the environment and telemetry generation (section 8), and `otelstarter doctor` to give actionable errors (section 9). Two independent implementations would duplicate most of their logic and drift apart.

## Decision

- **One check engine** (`internal/check`) returns a list of results, each passed, warning or failed, with *why* and *fix* text for problems.
- **`validate`** prints the results and exits `1` when any check fails. `--offline` skips the network check, for CI.
- **`doctor`** prints the same results, then an environment report:
  - versions
  - Docker and Compose
  - ports and whether the local stack is running
  - where `opentelemetry-instrument` resolves to
  - `otel.env` settings

  Secret-looking values are masked, so the report can be shared.
- **The Collector check sends empty OTLP export requests** (`{}` as JSON) to `/v1/traces`, `/v1/metrics` and `/v1/logs`. A Collector answers `200` and stores nothing. Verified against `otel/opentelemetry-collector:0.162.0`.
- **Checks never change anything and never run pip.** Packages are checked as declared, following ADR 0002.

## Alternatives

- **Split by scope:** static checks in `validate`, runtime checks in `doctor`. Simpler commands, but `validate` would then not cover the Collector and the environment as section 8 requires.
- **Send a real test span:** proves the full path, but leaves test data in the user's backend.
- **Compare package versions now:** needs the manifest parser to record pins. Deferred, since pip already rejects most conflicts.

## Consequences

- A new check is written once and shows up in both commands.
- `validate` in CI needs `--offline` unless a Collector runs in the job.
- Secret detection is heuristic and may miss unusual names.

## Update, 2026-10-06

The version check deferred above has been added. It compares exact `==` pins within each OpenTelemetry version family (beta `0.NNbN`, stable `1.NN.N`), grouped by version shape rather than package name.
