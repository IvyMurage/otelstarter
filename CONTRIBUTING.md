# Contributing to OTelStarter

Read [docs/ENGINEERING_GUIDELINES.md](docs/ENGINEERING_GUIDELINES.md) first. It is mandatory for humans and AI agents.

## Development setup

Install Go 1.25 or newer from <https://go.dev/dl/>.

```bash
go build ./...                       # compile everything
go run ./cmd/otelstarter help        # run the CLI without installing it
```

## Before opening a pull request

Run all three checks. CI runs the same checks.

```bash
gofmt -l .                # lists badly formatted files; must print nothing
go vet ./...              # catches common mistakes
go test -race ./...       # runs the tests and checks for data races
```

`gofmt -w .` fixes formatting for you.

Tests that start real Docker containers are skipped by default. Run them with:

```bash
OTELSTARTER_DOCKER_TESTS=1 go test ./internal/localstack/
```

## Layout

| Path | What lives there |
|------|------------------|
| `cmd/otelstarter/` | The `main` package. One line that calls `internal/cli`. |
| `internal/cli/` | Argument parsing, command dispatch and output formatting. No business logic. |
| `internal/project/` | Language-neutral description of an analysed project. |
| `internal/detect/` | Finds which languages a project uses and calls their adapter. |
| `internal/adapters/python/` | Reads Python manifests, recommends instrumentation, and knows how Python projects add packages. |
| `internal/check/` | The checks behind `validate` and `doctor`. Read-only. |
| `internal/localstack/` | The local Collector + Grafana stack for `run`. The only package that runs Docker. Its config files are in `files/` and embedded in the binary. |
| `internal/setup/` | Builds and applies the file changes for `init`. The only package that writes to a user's project. |
| `sdk/node/` | The Node.js SDK (TypeScript): loader, structured logger, redaction, query sanitising, domain events. `npm test` inside it; `RELEASING.md` covers publishing to npm. |
| `sdk/python/` | The Python domain-event SDK that runs inside users' apps. Its own README covers development; `RELEASING.md` covers publishing to PyPI. |
| `docs/` | Architecture, requirements, guidelines, ADRs. |

Go only allows code inside `internal/` to be imported by this module, so nothing there is a public API.

New packages are added when a feature needs them, not ahead of time. The planned order is in [docs/adr/0001-write-the-cli-in-go.md](docs/adr/0001-write-the-cli-in-go.md).

## Tests

Tests live next to the code they test, in `*_test.go` files. That is the Go convention. Test behaviour (exit codes, output) rather than internals.

## Pull requests

Keep each PR to one purpose. Answer the questions in section 37 of the engineering guidelines in the PR description.
