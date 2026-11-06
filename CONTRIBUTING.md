# Contributing to OTelStarter

Thank you for considering it. Every kind of help counts:
- a bug report with `otelstarter doctor` output;
- a fix to a confusing sentence in the docs;
- support for a library you use;
- a whole new language.

You do not need to understand the whole project to help. This page gets you from a fresh clone to a merged pull request:

1. [Fork and clone the repository](#1-fork-and-clone-the-repository)
2. [Install the dependencies](#2-install-the-dependencies)
3. [Run the tests](#3-run-the-tests)
4. [Run the CLI locally](#4-run-the-cli-locally)
5. [Follow the coding conventions](#5-coding-conventions)
6. [Submit a pull request](#6-submit-a-pull-request)
7. [What happens next, and how long it takes](#7-what-happens-next-and-how-long-it-takes)

## What kind of change?

| You want to… | Start here |
|---|---|
| Report a bug or ask for a feature | [Open an issue](https://github.com/IvyMurage/otelstarter/issues/new/choose). The forms ask for what helps most |
| Ask a question or talk through an idea | [Discussions](https://github.com/IvyMurage/otelstarter/discussions) |
| Report a security problem | Privately, as in [SECURITY.md](SECURITY.md). Please not in a public issue |
| Fix a typo or improve the docs | Edit the file on GitHub and open the pull request from there. No setup needed |
| Support another library (Redis client, queue, ORM…) | [Adding a library](docs/adding-a-language.md#adding-a-library), usually two small edits and a test |
| Support another language | [Adding a language](docs/adding-a-language.md#adding-a-language). Please open an issue first so we can agree on the approach |
| Anything bigger | Open an issue first. A short discussion saves you time if the idea needs a different shape |

Looking for something to start with? Issues labelled [good first issue](https://github.com/IvyMurage/otelstarter/labels/good%20first%20issue) are small and explain which files to touch and how to test. Comment on one to say you are taking it, so nobody duplicates your work.

New to the codebase? The [project guide](docs/PROJECT_GUIDE.md) explains what every file does.

## 1. Fork and clone the repository

You cannot push to this repository directly, so you work on your own copy (a *fork*) and send changes back as a pull request.

1. On [github.com/IvyMurage/otelstarter](https://github.com/IvyMurage/otelstarter), click **Fork**. GitHub creates `github.com/<you>/otelstarter`.
2. Clone your fork, and add this repository as `upstream` so you can pull in new changes:
   ```bash
   git clone https://github.com/<you>/otelstarter.git
   cd otelstarter
   git remote add upstream https://github.com/IvyMurage/otelstarter.git
   ```
   With the [GitHub CLI](https://cli.github.com/), `gh repo fork IvyMurage/otelstarter --clone` does both steps.
3. Create a branch for your change. Never work on `main`:
   ```bash
   git switch -c fix/clearer-permission-error
   ```
4. Before you start, and again before opening the pull request, bring your branch up to date:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

## 2. Install the dependencies

The project has three parts. Install only what you will touch:

| Part | Folder | You need |
|---|---|---|
| The CLI | the repository root | [Go 1.25+](https://go.dev/dl/) |
| The Node.js SDK | `sdk/node/` | [Node.js](https://nodejs.org/) 22 or 24 (what CI tests) |
| The Python SDK | `sdk/python/` | [Python](https://www.python.org/downloads/) 3.10+ |
| The compatibility check (optional) | `compat/` | all three, and [Docker](https://docs.docker.com/get-docker/) for a Postgres |

**The CLI** has no install step: Go downloads its one dependency the first time you build. To fetch it up front, run `go mod download`.

**The Node.js SDK:**
```bash
cd sdk/node
npm ci          # installs exactly what package-lock.json lists
```

**The Python SDK**, in a virtual environment so nothing touches your system Python:
```bash
cd sdk/python
python -m venv .venv
. .venv/bin/activate          # Windows: .venv\Scripts\activate
pip install -e '.[dev]'       # the SDK, editable, plus pytest, ruff and mypy
```

## 3. Run the tests

Run the checks for the part you changed. CI runs the same ones, so passing locally means passing there.

**The CLI (Go)**, from the repository root:
```bash
gofmt -l .              # lists badly formatted files; must print nothing (gofmt -w . fixes them)
go vet ./...
go test -race ./...
```
Tests that start real Docker containers are skipped by default. To run them: `OTELSTARTER_DOCKER_TESTS=1 go test ./internal/localstack/`.

To run one test while you work: `go test ./internal/adapters/node/ -run TestAnalyzeExpressApp -v`.

**The Node.js SDK:**
```bash
cd sdk/node
npm test                # builds the TypeScript, then runs the tests
```

**The Python SDK**, with the virtual environment active:
```bash
cd sdk/python
pytest && ruff check . && ruff format --check . && mypy
```

**Both SDKs** also run the shared cases in [`sdk/conventions.json`](sdk/conventions.json), so redaction and query cleaning behave the same in both. If you change that behaviour, add a case there and both test suites check it.

**The compatibility check.** Run this when you change how the SDKs start, export or stop, or what `init` writes. It runs real Express, NestJS, Flask and FastAPI apps with and without otelstarter, and checks they behave the same:
```bash
docker run -d --rm --name compat-pg -e POSTGRES_PASSWORD=pw -p 127.0.0.1:5432:5432 postgres:16-alpine
DATABASE_URL=postgres://postgres:pw@127.0.0.1:5432/postgres python3 compat/run.py
```
See [compat/README.md](compat/README.md). CI runs it on every pull request, so you can also leave it to CI.

## 4. Run the CLI locally

Run the CLI straight from your source code, with no install:
```bash
go run ./cmd/otelstarter help
```

The apps in `compat/apps/` make good practice projects. `analyze` and `init --dry-run` only read, so they are safe to run anywhere:
```bash
go run ./cmd/otelstarter analyze compat/apps/express
go run ./cmd/otelstarter init --dry-run compat/apps/flask
```

To try your build on one of your own projects, build a binary and run it from there:
```bash
go build -o ./otelstarter ./cmd/otelstarter
cd ~/code/my-app
/path/to/otelstarter/otelstarter analyze
```
Use `init --dry-run` first: it shows every file `init` would change, and changes nothing. Without `--dry-run`, `init` edits that project's files.

`go run ./cmd/otelstarter run` starts the local Collector and Grafana, so it needs Docker.

**Trying an SDK change in a real app.** Point the app at your local copy instead of the published package:
- Node.js: in the app, `npm install /path/to/otelstarter/sdk/node` (run `npm run build` in `sdk/node` after each change).
- Python: in the app's virtual environment, `pip install -e /path/to/otelstarter/sdk/python`.

## 5. Coding conventions

The [engineering guidelines](docs/ENGINEERING_GUIDELINES.md) explain how we build and why. The ones that come up most in review:

- **Keep it simple.** The smallest change that solves the problem. No new abstractions, settings or dependencies unless the change needs them (sections [4](docs/ENGINEERING_GUIDELINES.md#4-keep-it-simple), [5](docs/ENGINEERING_GUIDELINES.md#5-build-what-is-needed-now) and [13](docs/ENGINEERING_GUIDELINES.md#13-dependencies)).
- **Follow the code around you,** even where you would have chosen differently.
- **Each language's own style:** `gofmt` and `MixedCaps` in Go, `camelCase` in TypeScript, `snake_case` in Python, with `ruff format` and `mypy` passing.
- **Clear names** that say what a thing does: `BuildPlan`, `sanitizeQuery`. Not `handle`, `manager` or `utils`.
- **Code lives where its job belongs** ([section 30](docs/ENGINEERING_GUIDELINES.md#30-where-code-lives)). No `utils` or `helpers` files.
- **Comments explain why,** not what.
- **Errors a user sees** say what happened, why, and how to fix it.
- **Telemetry is private by default.** Never record request bodies, passwords, tokens or personal data. A change that affects what users send needs a test.
- **Tests come with the change.** Test behaviour, not internals.

Using an AI assistant is fine. You are responsible for what it writes, exactly as if you had typed it; [section 2](docs/ENGINEERING_GUIDELINES.md#2-using-ai-coding-tools) covers what that means.

## 6. Submit a pull request

1. Make **one** change per pull request. A fix and an unrelated refactor are two pull requests.
2. Add or update tests for what you changed, and the docs if users will notice it.
3. Add a line to [`CHANGELOG.md`](CHANGELOG.md) under `[Unreleased]` if users will notice the change.
4. Run the checks in [step 3](#3-run-the-tests) for the parts you changed.
5. Read your own diff and remove anything not needed.
6. Commit with a message that says what the change does, for example `fix(cli): explain how to fix a permission error when writing otel.env`. Then push your branch to your fork:
   ```bash
   git push -u origin fix/clearer-permission-error
   ```
7. GitHub shows a **Compare & pull request** button on your fork. Open the pull request against `main` of `IvyMurage/otelstarter`. If it closes an issue, write `Closes #40` in the description.
8. The template asks a few short questions. Answer the ones that apply; one sentence each is fine.

Not finished yet, but want early feedback? Open it as a **draft** pull request.

## 7. What happens next, and how long it takes

OTelStarter is maintained by one person, so these are honest targets, not guarantees:

| Step | What happens | How long |
|---|---|---|
| **CI approval** (first pull request only) | GitHub waits for me to approve the CI run. This is a safety rule for public repositories, not a judgement on your change | Within 2 working days |
| **CI** | Runs the Go tests, both SDKs on two versions each, the local-stack config check, and the compatibility check. If something fails, the log says which check and why | A few minutes |
| **First review** | I read the change and reply: approve it, or suggest changes with the reason | Within 5 working days |
| **Follow-up reviews** | After you push changes | Usually 1–2 working days |
| **Merge and release** | I merge it. Your change ships in the next release, and the release notes credit you | |

Heard nothing after a week? Please comment on the pull request. It has slipped past me, and a reminder is welcome, never a bother. Reviewers' suggestions are explained, and a "why not X?" from you is always welcome.

Stuck at any point? Comment on the pull request or the issue, or [ask in Discussions](https://github.com/IvyMurage/otelstarter/discussions/new?category=q-a).

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
