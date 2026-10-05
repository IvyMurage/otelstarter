# ADR 0002: Detect projects from manifest files only

**Status:** Accepted
**Date:** 2026-10-05

## Context

`otelstarter analyze` must find a project's language, framework and dependencies (FR-001, FR-002). The project may be untrusted, for example a freshly cloned repository.

## Decision

- **Read manifests; never run project code or package managers.** Detection parses `pyproject.toml`, `requirements.txt` and `Pipfile` as data. `setup.py` is skipped because reading it would mean executing it.
- **Open files through `os.Root`.** It stops a symlink in the project from making us read files elsewhere on the machine.
- **Parse TOML with `github.com/pelletier/go-toml/v2`.** Go's standard library has no TOML parser, and `pyproject.toml` is the main modern Python manifest. The library is MIT-licensed, actively maintained, widely used and has no dependencies of its own.
- **Split responsibilities:** `internal/detect` decides which languages are present; `internal/adapters/python` reads Python manifests; `internal/project` holds the shared result types so adapters do not import the detector.
- **No shared adapter interface yet.** Each adapter is a plain function. An interface will be extracted once a second adapter shows what they share.

## Alternatives

- **Ask pip or uv for the installed packages.** More accurate for transitive dependencies, but it runs tools inside an untrusted project and needs a working environment.
- **Write a small TOML parser.** Fewer dependencies, but TOML has many edge cases and bugs would surface as wrong detection.
- **Scan source code imports.** Catches undeclared dependencies, but is slower, language-specific and easily confused by dead code.

## Consequences

- Detection is fast, safe and works without installing the project.
- Dependencies declared only in `setup.py`, or only in subfolders, are not seen. The output says so.
- One third-party dependency is added.
