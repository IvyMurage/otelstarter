# ADR 0001: Write the CLI in Go

**Status:** Accepted
**Date:** 2026-10-05

## Context

OTelStarter configures Python, Node.js and Go applications. The CLI runs on the developer's machine before any telemetry exists. Whatever language the CLI uses, users of the other two languages must still be able to install it easily.

The domain-event API (`observability.event(...)`) runs inside the user's application, so it has to be written in each application language anyway. The CLI language and the SDK language are separate decisions.

## Decision

Write the CLI in Go. Use only the standard library until a dependency clearly pays for itself: `flag` for argument parsing and `testing` for tests.

Repository layout follows Go conventions instead of the `src/` tree sketched in the architecture document:

- `cmd/otelstarter/` holds the `main` package.
- `internal/` holds everything else. Go's compiler stops other modules from importing it, so we are not committing to a public Go API.
- Tests sit next to code as `*_test.go` files instead of a separate `tests/` tree.

The architectural boundaries from `ARCHITECTURE.md` still apply. They become packages under `internal/` once there is code for them:

| Phase | Package |
|-------|---------|
| 2 | `internal/detect` (language, framework, dependencies) |
| 3 | `internal/adapters/python` |
| 4 | `internal/config`, `internal/safety` |
| 6 | `templates/collector` |
| 7 | `internal/validate` |

## Alternatives

- **Python.** Fastest start and matches the Python-first adapter phase. Node.js and Go users would need a Python runtime.
- **Node.js/TypeScript.** Easy `npx` distribution. Python and Go users would need Node.js.
- **A CLI framework such as Cobra.** Adds subcommand trees and shell completion we do not need for six flat commands. Can be revisited if the command set grows.

## Consequences

- Users get one static binary with no runtime to install.
- Contributors need Go installed.
- Language detection must read other ecosystems' files (`pyproject.toml`, `package.json`, `go.mod`) from Go. TOML parsing may need a dependency in Phase 2.
- Domain-event SDKs for Python, Node.js and Go live outside the CLI. Where they live is an open decision for Phase 5.
