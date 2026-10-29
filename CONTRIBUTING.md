# Contributing to OTelStarter

Read [docs/ENGINEERING_GUIDELINES.md](docs/ENGINEERING_GUIDELINES.md) first. Every change follows it.

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

## Releasing the CLI

Pushing a tag like `v0.2.0` runs `.github/workflows/release-cli.yml`:
1. it tests the CLI;
2. it builds it for Linux, macOS and Windows (amd64 and arm64) with `.github/scripts/build-cli.sh`;
3. it publishes a GitHub release with the archives, `checksums.txt` and notes generated from the merged pull requests.

A tag with a suffix, such as `v0.2.0-rc.1`, becomes a pre-release.

Before tagging:
1. In `CHANGELOG.md`, rename `[Unreleased]` to the version and date, and start a new empty `[Unreleased]` above it. Merge that.
2. Then tag the merged commit:

```bash
git switch main && git pull
git log --oneline -1          # check it is the commit you mean
git tag v0.2.0 && git push origin v0.2.0
```

To try the build locally: `.github/scripts/build-cli.sh v0.0.0-test dist`.

The SDKs have their own release tags and workflows: `sdk/node/RELEASING.md` and `sdk/python/RELEASING.md`.

## Pull requests

Keep each PR to one purpose. Answer the questions in section 37 of the engineering guidelines in the PR description.
