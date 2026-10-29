# Contributing to OTelStarter

Thank you for considering it. Every kind of help counts:
- a bug report with `otelstarter doctor` output;
- a fix to a confusing sentence in the docs;
- support for a library you use;
- a whole new language.

You do not need to understand the whole project to help. This page gets you from a fresh clone to a merged pull request.

## What kind of change?

| You want to… | Start here |
|---|---|
| Report a bug or ask for a feature | [Open an issue](https://github.com/IvyMurage/otelstarter/issues/new/choose). The forms ask for what helps most |
| Report a security problem | Privately, as in [SECURITY.md](SECURITY.md). Please not in a public issue |
| Fix a typo or improve the docs | Edit the file on GitHub and open the pull request from there. No setup needed |
| Support another library (Redis client, queue, ORM…) | [Adding a library](docs/adding-a-language.md#adding-a-library), usually two small edits and a test |
| Support another language | [Adding a language](docs/adding-a-language.md#adding-a-language). Please open an issue first so we can agree on the approach |
| Anything bigger | Open an issue first. A short discussion saves you time if the idea needs a different shape |

New to the codebase? The [project guide](docs/PROJECT_GUIDE.md) explains what every file does, and the [engineering guidelines](docs/ENGINEERING_GUIDELINES.md) explain how we build. Using an AI assistant is fine; [section 2](docs/ENGINEERING_GUIDELINES.md#2-using-ai-coding-tools) covers what that means here.

## Set up

The project has three parts. Install only what you will touch:

| Part | Folder | You need |
|---|---|---|
| The CLI | the repository root | [Go 1.25+](https://go.dev/dl/) |
| The Node.js SDK | `sdk/node/` | Node.js 22.14+ |
| The Python SDK | `sdk/python/` | Python 3.10+ |

```bash
git clone https://github.com/IvyMurage/otelstarter.git
cd otelstarter
go run ./cmd/otelstarter help        # the CLI, straight from the source
```

## Check your change

Run the checks for the part you changed. CI runs the same ones, so passing locally means passing there.

**The CLI (Go):**
```bash
gofmt -l .              # lists badly formatted files; must print nothing (gofmt -w . fixes them)
go vet ./...
go test -race ./...
```
Tests that start real Docker containers are skipped by default: `OTELSTARTER_DOCKER_TESTS=1 go test ./internal/localstack/`.

**The Node.js SDK:**
```bash
cd sdk/node
npm ci
npm test
```

**The Python SDK** (in a virtual environment):
```bash
cd sdk/python
python -m venv .venv && . .venv/bin/activate
pip install -e '.[dev]'
pytest && ruff check . && ruff format --check . && mypy
```

**The compatibility check.** Run this when you change how the SDKs start, export or stop, or what `init` writes. It runs real Express, NestJS, Flask and FastAPI apps with and without otelstarter, and checks they behave the same. It needs Go, Node.js, Python and a Postgres:
```bash
docker run -d --rm --name compat-pg -e POSTGRES_PASSWORD=pw -p 127.0.0.1:5432:5432 postgres:16-alpine
DATABASE_URL=postgres://postgres:pw@127.0.0.1:5432/postgres python3 compat/run.py
```
See [compat/README.md](compat/README.md). CI runs it on every pull request, so you can also leave it to CI.

## Open a pull request

1. Fork the repository, create a branch, and make **one** change. A fix and an unrelated refactor are two pull requests.
2. Add or update tests for what you changed, and the docs if users will notice it.
3. Add a line to `CHANGELOG.md` under `[Unreleased]` if users will notice the change.
4. Open the pull request. The template asks a few short questions; answer the ones that apply, one sentence each is fine.

**What happens next:**
- **Your first pull request:** GitHub waits for me to approve the CI run. This is a safety rule for public repositories, not a judgement on your change. I approve within a day or two.
- **CI** runs the Go tests, both SDKs on two versions each, the local-stack config check, and the compatibility check. If something fails, the log says which check and why. Ask in the pull request if it is unclear.
- **Review:** I aim to reply within a few days. Suggestions are explained, and a "why not X?" from you is always welcome.
- **When it is ready,** I merge it. Your change appears in the next release, and the release notes credit you.

Stuck at any point? Comment on the pull request or the issue. Asking is never a bother.

## Releasing (maintainers)

**The CLI:** pushing a tag like `v0.2.0` runs `.github/workflows/release-cli.yml`:
1. it tests the CLI;
2. it builds it for Linux, macOS and Windows (amd64 and arm64) with `.github/scripts/build-cli.sh`;
3. it publishes a GitHub release with the archives, `checksums.txt` and notes generated from the merged pull requests.

A tag with a suffix, such as `v0.2.0-rc.1`, becomes a pre-release. Before tagging, rename `[Unreleased]` in `CHANGELOG.md` to the version and date, start a new empty `[Unreleased]` above it, and merge that. Then:

```bash
git switch main && git pull
git log --oneline -1          # check it is the commit you mean
git tag v0.2.0 && git push origin v0.2.0
```

Never move a `v*` tag after pushing it: Go's module proxy keeps every version forever. If a release is wrong, release the next version.

To try the build locally: `.github/scripts/build-cli.sh v0.0.0-test dist`.

**The SDKs** have their own tags and workflows: [sdk/node/RELEASING.md](sdk/node/RELEASING.md) and [sdk/python/RELEASING.md](sdk/python/RELEASING.md).

## Code of conduct

Everyone taking part is expected to follow the [code of conduct](CODE_OF_CONDUCT.md).
