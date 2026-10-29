# OTelStarter: the owner's guide

This guide explains the whole project in plain English: what it is, how people use it, what happens inside, and what every file in the repository does. Read it top to bottom once. After that, use the file tour (section 6) as a map.

---

## 1. What OTelStarter is, in one minute

**The problem.** A developer has a working web app, say a FastAPI shop or an Express API, and wants to see what it is doing:
- **Traces:** which requests are slow, and why. "This checkout took 2 s, and 1.8 s of that was one SQL query."
- **Logs:** what the app wrote, each line linked to the request that wrote it.
- **Metrics:** numbers over time, such as requests per second, memory and errors.

OpenTelemetry is the industry standard for collecting all three. Setting it up by hand is slow and full of traps: which of 40 packages to install, which versions fit together, which settings to use, and how to run something that can show the results.

**What OTelStarter does.** It does that setup for you, safely, in a few commands:

```bash
otelstarter analyze      # "This is a FastAPI app using PostgreSQL and Redis"
otelstarter init         # writes otel.env and adds the right packages, after asking
otelstarter run -- uvicorn main:app    # starts Grafana locally and runs the app with telemetry on
```

Open http://localhost:3000 and the app's traces, logs and metrics are there.

**Its promises:**
- **Lightweight:** only the instrumentation the app needs. An Express + pg app gets 3 packages, not 40.
- **Private by default:** passwords, tokens and API keys become `[REDACTED]`, and SQL values become `?`, before anything leaves the app.
- **Never breaks the app:** the same answers, the same errors, the same way of stopping. `compat/run.py` proves this on real apps in CI.
- **Standard:** everything is plain OpenTelemetry. The settings file works with Docker, Kubernetes, or any OpenTelemetry backend, even without OTelStarter installed.

---

## 2. The three pieces

The repository holds three products that work together:

| Piece | Written in | What it is | How users get it |
|---|---|---|---|
| **The CLI** (`otelstarter` command) | Go | The tool the developer types. It reads the project, writes settings, starts Grafana, checks the setup | A ready-made download from GitHub Releases (Linux, macOS, Windows), or `go install github.com/IvyMurage/otelstarter/cmd/otelstarter@latest` |
| **The Node.js SDK** | TypeScript | A small library that runs **inside** a Node app: starts OpenTelemetry, cleans the data, writes JSON logs | npm: `otelstarter` |
| **The Python SDK** | Python | The same idea for Python apps | PyPI: `otelstarter` |

**How they connect.** The CLI never runs the user's code. It only reads files such as `package.json` and `requirements.txt`, and writes `otel.env`. It also tells the user which SDK packages to install, at exact versions. When the app starts, the SDK inside it reads `otel.env` and does the actual work.

```
  Developer's machine
  ┌──────────────────────────────────────────────────────────────────────┐
  │  otelstarter CLI (Go)                                                │
  │   reads package.json / requirements.txt                              │
  │   writes otel.env, prints/adds install command                       │
  │   starts Docker: Collector + Grafana                                 │
  │                                                                      │
  │   ┌──────────────────────────┐    OTLP     ┌────────────┐  ┌─────────┐ │
  │   │ the user's app           │ ──────────► │ Collector  │─►│ Grafana │ │
  │   │  + otelstarter SDK inside│  :4318      │ (filters)  │  │ :3000   │ │
  │   └──────────────────────────┘             └────────────┘  └─────────┘ │
  └──────────────────────────────────────────────────────────────────────┘
```

---

## 3. How a user uses it

### 3.1 Python app (FastAPI, Flask or Django)

```bash
cd ~/my-shop
otelstarter analyze
```
This prints what it found, for example: FastAPI, PostgreSQL (`psycopg2`), Redis, package manager pip. It also lists the packages it recommends, marking those already installed.

```bash
otelstarter init
```
This shows a **plan** first, then asks "Apply these changes?":
- **creates `otel.env`** with the service name, where to send data, the protocol, and the privacy exporters;
- **for pip,** adds a marked block of pinned packages to `requirements.txt`;
- **for uv, poetry, pdm or pipenv,** prints the exact command to run instead. It never edits those files itself.

`--dry-run` shows the plan without writing. `--yes` skips the question. Running it again is safe: it only adds what is missing.

```bash
pip install -r requirements.txt
otelstarter validate        # checks everything; exit code 1 if something is wrong
otelstarter run -- uvicorn main:app
```
`run`:
1. starts the local Collector and Grafana in Docker; the first time downloads about 1 GB;
2. loads `otel.env`;
3. starts the app through `opentelemetry-instrument`, which switches instrumentation on with no code changes.

**Ctrl+C** stops everything.

**Business events (optional).** In Python code:
```python
from otelstarter import observability
observability.event("order.created", domain="orders", outcome="success", entity_type="order", entity_id=order.id)
```

### 3.2 Node.js app (Express, NestJS, Koa or hapi)

```bash
otelstarter init
# prints, for an Express + pg app:
#   npm install --save-exact otelstarter@0.1.4 @opentelemetry/instrumentation-express@0.71.0 @opentelemetry/instrumentation-pg@0.75.0
npm install --save-exact ...          # run what it printed
otelstarter run -- npm start
```
`run` adds `--require otelstarter/register` to `NODE_OPTIONS`, which loads the SDK before the app. Any start command works: `node app.js`, `npm start`, `nest start`.

**Optional, in Node code:**
```js
const { logger, event } = require('otelstarter');
logger.info('order placed', { orderId: 'ord_1' });   // JSON line on stdout, also sent to Grafana
event('order.created', { domain: 'orders', outcome: 'success', entityType: 'order', entityId: 'ord_1' });
```

### 3.3 In production

The same `otel.env` works with Docker's `env_file`, Kubernetes and others. The user changes:
- `OTEL_EXPORTER_OTLP_ENDPOINT` to their real Collector;
- `deployment.environment.name` to `production`.

Production start commands, without the CLI:
- **Python:** `opentelemetry-instrument gunicorn app:app`
- **Node:** `NODE_OPTIONS="--require otelstarter/register" node server.js`

### 3.4 When something is wrong

- `otelstarter validate` lists every problem, each with a **Why** and a **Fix**. Example:
  ```
  ✗ OTEL_EXPORTER_OTLP_PROTOCOL is not set
      Why: The Python SDK then uses gRPC, but only the HTTP exporter is installed, so no telemetry is sent.
      Fix: Add OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf to otel.env.
  ```
- `--offline` skips the "is the Collector reachable?" check, for CI.
- `otelstarter doctor` runs the same checks, then prints an environment report that is safe to paste into a bug report. Secrets show as `*** (hidden)`.

---

## 4. What happens inside, following one request

An Express shop app runs under `otelstarter run`. A customer calls `GET /orders/42`:

1. **Startup.** Node reads `NODE_OPTIONS`, loads `otelstarter/register` (`sdk/node/src/register.ts`), and calls `start()` in `sdk.ts`. `start()`:
   - reads `OTEL_NODE_ENABLED_INSTRUMENTATIONS=http,undici,runtime-node,express,pg,pino` from `otel.env`;
   - loads only those instrumentation packages;
   - sets up three exporters (traces, metrics, logs) pointed at `localhost:4318`.

   For an ES-module app, `esm.ts` also switches on the `import` hook.
2. **The request.** The instrumentations record spans:
   - `GET /orders/:id` (http)
   - `request handler - /orders/:id` (express)
   - `pg.query:SELECT` (pg), with the SQL `SELECT 42 AS id, 'ann@shop.rw' AS email`
3. **Cleaning.** When the pg span ends, `PrivacySpanProcessor` (`processor.ts`) rewrites the SQL to `SELECT ? AS id, ? AS email`. Log records pass through `PrivacyLogRecordProcessor`, so `password: "hunter2"` becomes `[REDACTED]`. The rules live in `redaction.ts` and `sanitize.ts`.
4. **Sending.** Every few seconds the batch processors send the cleaned data over OTLP/HTTP to the Collector.
5. **Collector** (`internal/localstack/files/collector.yaml`). It drops noise spans such as Express's `query` middleware, then forwards to Grafana.
6. **Grafana** (`grafana/otel-lgtm`) stores the data:

   | Data | Stored in | Where in Grafana |
   |---|---|---|
   | Traces | Tempo | Explore → Tempo |
   | Logs | Loki | Explore → Loki |
   | Metrics | Prometheus | Explore → Prometheus |

7. **Stopping.** On SIGTERM, `register.ts`:
   - sends what is still waiting, for at most 3 seconds;
   - then raises SIGTERM again, so the app stops exactly as it would without OTelStarter.

**For Python the steps are the same idea, with different parts:**
- `opentelemetry-instrument` does the startup.
- The cleaning happens in the `otelstarter` exporters (`exporters.py`), because Python freezes a span once it ends. They clean a copy.
- `shutdown.py` handles stopping.

---

## 5. Words you will see

| Word | Meaning |
|---|---|
| **Span** | One timed step, such as "SQL query, 12 ms". A **trace** is all the spans of one request, as a tree |
| **Instrumentation** | Code that wraps a library (express, pg, flask…) so its work becomes spans, without changing the app's code |
| **OTLP** | OpenTelemetry's wire format. `4318` is OTLP over HTTP and `4317` over gRPC. OTelStarter uses HTTP to avoid the large `grpcio` package |
| **Collector** | A small server that receives OTLP, can filter it, and forwards it to a backend |
| **Exporter** | The part inside the app that sends data to the Collector |
| **Batch processor** | Gathers spans or logs and sends them in groups every few seconds; this is why shutdown must flush |
| **Resource** | Who sent the data: service name, environment, host |
| **Domain event** | A business fact, such as `order.created` or `payment.failed`, recorded as one log record linked to the trace |
| **Manifest** | The file that lists a project's dependencies: `package.json`, `requirements.txt`, `pyproject.toml`, `Pipfile` |
| **Adapter** | The per-language code in the CLI that understands one ecosystem (Python or Node.js) |
| **ADR** | Architecture Decision Record: a short page in `docs/adr/` explaining one decision and why |
| **Trusted publishing** | npm and PyPI accept uploads from our GitHub workflow without any stored password |
| **ESM / CommonJS** | Node's two module systems: `import` versus `require` |

---

## 6. Every file, folder by folder

### 6.1 The root

| File | What it is |
|---|---|
| `README.md` | The front page on GitHub: what it is, install, commands, links |
| `CHANGELOG.md` | Every user-visible change, newest first. New changes go under `[Unreleased]`; a CLI release renames that to the version, e.g. `[0.1.0]` |
| `CONTRIBUTING.md` | The contributor's path: which kind of change starts where, setup and checks for each part, what happens after opening a pull request, and releasing (for maintainers) |
| `LICENSE` | Apache-2.0. The SDK folders each have a copy, and CI checks the copies are identical |
| `SECURITY.md` | How to report a vulnerability privately (GitHub's Security tab), and what otelstarter guarantees |
| `CODE_OF_CONDUCT.md` | How people treat each other in the project, and how to report a problem |
| `go.mod` / `go.sum` | The Go module name (`github.com/IvyMurage/otelstarter`), minimum Go 1.25, and the CLI's only dependency, `go-toml`, which reads `pyproject.toml`. `go.sum` holds its checksums. Everything else is Go's standard library |
| `.gitignore` | Keeps build output, virtual environments, `node_modules` and `compat/.work/` out of git |

### 6.2 `cmd/otelstarter/`: where the program starts

| File | What it is |
|---|---|
| `main.go` | One line of real work: `os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))`. Everything else lives in `internal/`, so the logic can be tested without running a real program |

### 6.3 `internal/cli/`: the commands

Thin handlers. They parse flags, call the real packages, and print results. Go's `internal/` folder means no other project can import this code.

| File | What it is |
|---|---|
| `cli.go` | The list of commands (`init`, `analyze`, `run`, `validate`, `doctor`, `version`), help text and exit codes: 0 ok, 1 failed, 2 typed wrong. `Version` is stamped in at build time |
| `analyze.go` | `otelstarter analyze [path]`: calls `detect`, prints the language, framework, dependencies, recommended packages and notes |
| `init.go` | `otelstarter init [--dry-run] [--yes] [path]`: builds the plan (`setup.BuildPlan`), shows it, asks, applies it (`setup.Apply`), then prints the commands to run next |
| `run.go` | `otelstarter run [-- command]`, the longest command:<br>• checks Docker and ports, starts the stack (or reuses one already running), waits until healthy<br>• with a command: loads `otel.env`, adds `opentelemetry-instrument` (Python) or `--require otelstarter/register` (Node), and starts the app<br>• Ctrl+C stops everything. If the app crashes, the stack stays up so the crash's telemetry can still be explored |
| `validate.go` | `otelstarter validate [--offline]` and `doctor`: run `check.Run` and print ✓ / ! / ✗ with Why and Fix. `doctor` adds the environment report |
| `*_test.go` | Tests for each command. They run `cli.Run` with in-memory input and output and check exit codes and printed text |

### 6.4 `internal/project/`: the shared vocabulary

| File | What it is |
|---|---|
| `project.go` | The data types everything else passes around:<br>• `Project`: a folder and its languages<br>• `Language`: name, frameworks, dependencies, recommended packages, start command, notes, `ESModule`<br>• `Dependency`, `Instrumentation`, `EnvVar`, `Pin`<br><br>It is in its own package so the adapters and the detector can both use it without importing each other |

### 6.5 `internal/detect/`: which language is this?

| File | What it is |
|---|---|
| `detect.go` | Looks for manifest files (`package.json`, `requirements.txt`, `pyproject.toml`, `Pipfile`, `go.mod`), asks the matching adapter for details, and picks the primary language: Python, else Node.js. Go projects are recognised but not supported yet. Reads through `os.Root`, so it can never leave the project folder |
| `detect_test.go` | Tests with fake folders |

### 6.6 `internal/adapters/`: one adapter per language

Each adapter only **reads** files. It never runs pip, npm or the app, so analysing an untrusted project cannot run its code.

**`adapters/python/`**

| File | What it is |
|---|---|
| `python.go` | Reads `requirements.txt`, `pyproject.toml` (PEP 621 and Poetry) and `Pipfile`. Its table of known packages maps each library to its instrumentation, e.g. `flask` → `opentelemetry-instrumentation-flask`, `psycopg2-binary` → `…-psycopg2`, `redis`, `celery`, `httpx`… It works out the package manager from lock files and suggests a start command |
| `instrumentation.go` | The recommendation: 5 base packages for every app (distro, HTTP exporter, logging, system metrics, the `otelstarter` SDK), plus one per detected library, marking those already installed |
| `setup.go` | What `init` writes for Python:<br>• `EnvVars`: the Python lines of `otel.env`, including `OTEL_TRACES_EXPORTER=otelstarter`<br>• the pinned versions (`0.66b0` for instrumentation, `1.45.0` for the exporter, `otelstarterVersion = "0.3.1"`)<br>• the install command per package manager<br>• how `run` prefixes `opentelemetry-instrument` |
| `*_test.go` | Parsing tests for every manifest format. One test fails if `otelstarterVersion` differs from the Python SDK's `__version__` |

**`adapters/node/`**

| File | What it is |
|---|---|
| `node.go` | Reads `package.json`: frameworks (Express, NestJS, Koa, hapi, Fastify), libraries (pg, mysql, redis, ioredis, mongodb, kafkajs, amqplib, knex, graphql…), logging library, package manager from the lock file, `"type": "module"`. Produces `OTEL_NODE_ENABLED_INSTRUMENTATIONS` and the package list. Adds notes for gaps: Fastify, Prisma, `console.log`, and the old 80 MB bundle |
| `setup.go` | The tested versions (`SDKVersion = "0.1.4"` plus one pin per instrumentation package), the install command for npm, yarn, pnpm or bun, and how `run` finds the loader. It prefers `otelstarter/register` and falls back to the old bundle's loader for older projects |
| `*_test.go` | Tests. Two of them guard the data: every recognised library has a pinned version, and `SDKVersion` equals `sdk/node/package.json` |

### 6.7 `internal/setup/`: the only code that writes to a user's project

| File | What it is |
|---|---|
| `plan.go` | `BuildPlan(dir)`: works out the changes **without writing anything** (create `otel.env`, add a block to `requirements.txt`, commands to run). This is why `--dry-run` is safe |
| `apply.go` | `Apply(plan)`: writes the changes. Each file is written to a temporary file, then renamed into place, so a crash never leaves a half-written file. Uses `os.Root` so it cannot write outside the project |
| `envfile.go` | Writes `otel.env` with a comment above every setting, and reads it back for `run`. Real shell variables win over the file |
| `*_test.go` | Tests: plans for each package manager, running twice changes nothing, existing files untouched |

### 6.8 `internal/check/`: the checks behind validate and doctor

Read-only. The only outside contact is an **empty** OTLP request to the Collector, which stores nothing.

| File | What it is |
|---|---|
| `check.go` | `Run()`: the list of checks in order, and the `Result` type (pass, warn or fail, plus Why and Fix). Python projects get the Python checks, Node projects the Node checks |
| `config.go` | Checks on `otel.env`:<br>• the file parses, and the service name is set<br>• the protocol is `http/protobuf`, and the endpoint is a plain URL<br>• deprecated settings<br>• secret-looking values. `IsSecret` is also what hides values in `doctor` |
| `runtime.go` | Checks on the machine:<br>• the Collector answers on `/v1/traces`, `/v1/metrics` and `/v1/logs`<br>• the required packages are declared<br>• `opentelemetry-instrument` is on PATH<br>• the Python version is 3.10+ |
| `versions.go` | Python version checks:<br>• OpenTelemetry pins that do not belong together (all `0.NNbN` packages must match, and all `1.NN.N`)<br>• an `otelstarter` pin older than the one `init` uses<br>• `otel.env` not using the privacy exporters, or selecting them with an SDK too old to have them |
| `node.go` | Node checks:<br>• the Node.js version: 18.19+ / 20.6+, and 22.14+ for an ES-module app<br>• packages actually in `node_modules`<br>• the advice notes (no logger, Fastify, Prisma, old bundle), turned into warnings |
| `report.go` | `doctor`'s environment report: OS, Go and Docker versions, ports, and `otel.env` with secrets hidden |
| `*_test.go` | Tests. Fake `python3` and `node` scripts on PATH stand in for real versions, and a test HTTP server stands in for a Collector |

### 6.9 `internal/localstack/`: the local Collector and Grafana

| File | What it is |
|---|---|
| `stack.go` | Starts and stops the stack with `docker compose`. Arguments are always a fixed list, never a shell string, so nothing a user types can run as a command. The config files are copied to `~/.cache/otelstarter`, and the project name is fixed, so a second `run` reuses the first one's stack instead of breaking it. Also checks that ports are free |
| `files/compose.yaml` | The two containers:<br>• `otel/opentelemetry-collector:0.162.0`, on 4317 (gRPC), 4318 (HTTP) and 13133 (health)<br>• `grafana/otel-lgtm:0.35.0`, Grafana on 3000<br><br>Everything listens on `127.0.0.1` only, never on the network |
| `files/collector.yaml` | The Collector's pipeline: receive OTLP, drop noisy spans (FastAPI's per-request send/receive, Express's `query`/`expressInit`), batch, forward to Grafana's image |
| `stack_test.go` | Tests. The ones that start real containers run only with `OTELSTARTER_DOCKER_TESTS=1` |

These files are built into the binary with `go:embed`, so users never download them separately.

### 6.10 `sdk/node/`: the Node.js SDK (npm `otelstarter`)

| File | What it is |
|---|---|
| `package.json` | Name, **version** (bump this to release), dependencies (only the OpenTelemetry parts needed, plus pino and `import-in-the-middle`), and `exports`: `otelstarter` and `otelstarter/register` |
| `tsconfig.json` | TypeScript settings. Compiles `src/` and `test/` to `dist/` as CommonJS |
| `src/register.ts` | **The entry point** loaded with `--require`:<br>• calls `start()` and warns about missing instrumentation packages<br>• handles SIGTERM: flush for at most 3 s, then stop like Node would<br>• does nothing inside Node's internal loader thread |
| `src/sdk.ts` | `start()`: builds the resource (service name, host…), the three providers and OTLP exporters wrapped in the privacy processors, and loads the instrumentation packages named in `OTEL_NODE_ENABLED_INSTRUMENTATIONS`. A missing package is reported, never fatal |
| `src/esm.ts` | ES-module support. Switches on the `import-in-the-middle` hook, limited to the instrumented libraries, only when the app's entry file is an ES module. `OTELSTARTER_ESM=true/false` overrides |
| `src/processor.ts` | `PrivacySpanProcessor` (cleans span attributes when the span ends) and `PrivacyLogRecordProcessor` (redacts every log record, including logs from the app's own pino or winston) |
| `src/redaction.ts` | The list of sensitive names (password, token, api_key, authorization, cookie, cvv…), `isSensitiveKey`, and `redactUrl`, which hides `?token=…` in URLs |
| `src/sanitize.ts` | `sanitizeQuery`: SQL literals become `?`. `querySummary`: a short label such as `SELECT orders` |
| `src/logger.ts` | `logger.info/warn/error/debug`: JSON lines on stdout with timestamp, severity, service, environment, trace_id and span_id. Takes `(message, fields)` and pino's `(fields, message)` |
| `src/events.ts` | `event(name, fields)`: domain events, one log record each. It enforces `<domain>.<action>` names and the allowed fields, and never throws |
| `src/config.ts` | Reads the service name, environment and lists from the standard `OTEL_*` variables |
| `src/index.ts` | What `require('otelstarter')` returns: `logger`, `event`, the redaction and sanitising helpers |
| `test/*.test.ts` | Tests, run with `npm test`:<br>• `privacy.test.ts`: the shared cases plus both processors<br>• `process.test.ts`: real child processes for JSON logs, SIGTERM and the Collector-down timing<br>• `esm.test.ts`: the `import` hook<br>• `events.test.ts` |
| `README.md` | The page shown on npm |
| `RELEASING.md` | How to publish: bump the version, merge, tag `sdk-node-vX.Y.Z`, plus the one-time npm setup |
| `LICENSE` | A copy of the root LICENSE |

### 6.11 `sdk/python/`: the Python SDK (PyPI `otelstarter`)

| File | What it is |
|---|---|
| `pyproject.toml` | Package metadata. The only runtime dependency is `opentelemetry-api`. Its **entry points** are how OpenTelemetry finds our code without imports: the `otelstarter` trace and log exporters, and the instrumentor |
| `src/otelstarter/__init__.py` | Holds `__version__`, **the one place to bump** for a release |
| `src/otelstarter/exporters.py` | `PrivacySpanExporter` and `PrivacyLogExporter`: they clean a **copy** of each span and log record (SQL to `?`, secrets to `[REDACTED]`), then hand it to the normal OTLP HTTP exporter. Selected by `OTEL_TRACES_EXPORTER=otelstarter` |
| `src/otelstarter/instrumentor.py` | Runs automatically under `opentelemetry-instrument`. Turns on JSON logs (`OTELSTARTER_JSON_LOGS=false` turns them off) and installs the shutdown behaviour |
| `src/otelstarter/structured_logs.py` | `JsonFormatter`: makes Python's normal `logging` print the same JSON fields as the Node SDK |
| `src/otelstarter/shutdown.py` | At exit, flushes traces, logs and metrics at the same time, for at most 3 s. On SIGTERM, if the app has no handler of its own, sends the last batch, then stops exactly as before |
| `src/otelstarter/observability.py` | `observability.event(...)`: domain events, with the same rules as the Node SDK. With `OTELSTARTER_CONSOLE_EVENTS=true` (set by `run`) it also prints a line to the terminal |
| `src/otelstarter/redaction.py`, `sanitize.py` | The same rules as the Node files of the same names |
| `src/otelstarter/py.typed` | An empty marker telling type checkers the package has type hints |
| `tests/` | pytest:<br>• `test_conventions.py`: the shared cases<br>• `test_exporters.py`, `test_structured_logs.py`, `test_observability.py`<br>• `test_shutdown.py`: real processes, SIGTERM, Collector down, last span delivered<br>• `conftest.py`: test setup |
| `README.md`, `RELEASING.md`, `LICENSE` | The PyPI page, release steps (tag `sdk-python-vX.Y.Z`), license copy |

### 6.12 `sdk/conventions.json`: the shared contract

Test cases that **both** SDKs run, for example "`SELECT * FROM users WHERE email = 'a@b.c'` becomes `... = ?`" and "`apiKey` is sensitive". If someone changes a rule in one language only, the other language's tests fail. The human-readable version is `docs/TELEMETRY_CONVENTIONS.md`.

### 6.13 `compat/`: proof that apps do not break

| File | What it is |
|---|---|
| `run.py` | The compatibility check, run in CI:<br>1. sets up each app like a user would: install, `otelstarter init`, install what init asks for, with the SDKs from this checkout<br>2. runs it **without** OTelStarter, **with** it, and with it while the Collector is **down**<br>3. compares every answer, times SIGTERM, and checks that a small built-in Collector got traces, logs and metrics with no email or password<br>4. prints the cost: startup, memory, request time |
| `apps/express`, `apps/express-esm` | Express 5 + pg + pino, as CommonJS and as ES modules |
| `apps/nestjs11`, `apps/nestjs12` | NestJS 11 and 12 (TypeScript compiled to CommonJS) |
| `apps/flask` | Flask 3 + psycopg2. Runs on the dev server and under gunicorn with 2 workers |
| `apps/fastapi` | FastAPI + psycopg2. Runs under uvicorn, alone and with 2 workers |
| `README.md` | How to run it locally and how to add an app |

Every test app has the same routes: `/health`, `/orders/42` (SQL with an email in it), `/boom` (crashes), an unknown page, and `POST /echo` (logs a password).

### 6.14 `.github/`: automation on GitHub

| File | What it is |
|---|---|
| `pull_request_template.md` | The short questions every pull request answers: what and why, how, checks, impact |
| `ISSUE_TEMPLATE/` | The forms for new issues. A bug report asks for `otelstarter doctor` output; a feature request asks what problem it solves; security reports are pointed to the private channel |
| `ci.yml` | Runs on every PR and every push to `main`:<br>• `test`: gofmt, go vet, go test -race, build<br>• `local-stack-config`: real Collector and Compose validate their configs<br>• `python-sdk`: Python 3.10 and 3.13: lint, type check, tests<br>• `node-sdk`: Node 22 and 24<br>• `compat`: the 8 real apps, with a Postgres service |
| `release-python-sdk.yml` | Runs on a tag `sdk-python-v*`: checks the tag matches the version, tests, builds, publishes to PyPI. No password: PyPI trusts this workflow (environment `pypi`) |
| `release-node-sdk.yml` | Runs on a tag `sdk-node-v*`: the same checks, then `npm publish` through npm's trusted publishing (environment `npm`) |
| `release-cli.yml` | Runs on a tag `v*` (e.g. `v0.1.0`): checks the tag is a version, tests, builds the CLI for 6 platforms, and publishes a GitHub Release with the archives and `checksums.txt`. A tag like `v0.2.0-rc.1` becomes a pre-release |
| `../scripts/build-cli.sh` | The build the release runs, also usable locally: `.github/scripts/build-cli.sh v0.0.0-test dist`. Static binaries (`CGO_ENABLED=0`) for Linux, macOS and Windows on amd64 and arm64, each packed with LICENSE and README. Archive names have no version, so `releases/latest/download/otelstarter_linux_amd64.tar.gz` always gets the newest |

### 6.15 `docs/`: the documentation

**For users**

| File | What it covers |
|---|---|
| `quickstart.md` | From nothing to traces in Grafana in about 10 minutes, with a small Python app |
| `analyze.md`, `init.md`, `run.md`, `validate.md` | One page per command: what it does, every option, examples, limitations |
| `performance.md` | Measured cost of the instrumentation, and how to sample traces in production |
| `TELEMETRY_CONVENTIONS.md` | The log fields, the redaction list and the SQL rules both SDKs follow |

**For the project**

| File | What it covers |
|---|---|
| `README_DOCS.md` | The reading order for contributors |
| `ARCHITECTURE.md` | The intended system design and its boundaries |
| `REQUIREMENTS.md` | What the product must do (the FR-xxx numbers referenced in code) |
| `ENGINEERING_GUIDELINES.md` | How we build OTelStarter: the rules every change follows, including changes made with AI assistants |
| `adding-a-language.md` | Step by step: adding support for a library (Node.js or Python), and adding a whole language |
| `DOMAIN_EVENTS.md` | The business-event model: names, fields, what is allowed |
| `PROJECT_GUIDE.md` | This page |

**`docs/adr/`: why things are the way they are**

| ADR | Decision |
|---|---|
| 0001 | The CLI is written in Go: one small binary, no runtime to install |
| 0002 | Detect from manifest files only; never run the project's code |
| 0003 | `init` writes standard `OTEL_*` settings and never runs package managers itself |
| 0004 | The local stack: a Collector in front of Grafana's all-in-one image |
| 0005 | `validate` and `doctor` share one set of checks |
| 0006 | Domain events are log-based events, not span events |
| 0007 | Our own lean Node SDK with JSON logs, redaction and query cleaning, instead of the 80 MB bundle |
| 0008 | Python privacy through exporters, because Python freezes finished spans |
| 0009 | ES modules through the same loader, only switched on when needed |
| 0010 | Shutdown sends what it can within 3 s, then stops exactly as before |

---

## 7. Releases: how a new version reaches users

| Product | Version lives in | Released by | Lands on |
|---|---|---|---|
| Node SDK | `sdk/node/package.json` | tag `sdk-node-vX.Y.Z` | npm |
| Python SDK | `sdk/python/src/otelstarter/__init__.py` | tag `sdk-python-vX.Y.Z` | PyPI |
| CLI | the tag itself, stamped in with `-ldflags -X …cli.Version=`; CHANGELOG section | tag `vX.Y.Z` | GitHub Releases, and `go install` |

**The pins.** The CLI pins the SDK versions it was tested with: `SDKVersion` in `internal/adapters/node/setup.go`, and `otelstarterVersion` in `internal/adapters/python/setup.go`. Tests fail if a pin and its SDK version differ.

**The recipe for an SDK release:**
1. Bump the version and the CLI's pin in the same PR. Add a CHANGELOG line.
2. Merge.
3. Tag right away, so `init` on `main` never asks for a version that is not published yet:
   ```bash
   git switch main && git pull
   git log --oneline -1                 # check it shows the merged PR
   git tag sdk-node-v0.1.5 && git push origin sdk-node-v0.1.5
   ```
4. The workflow refuses a tag that does not match the version, so a mistake publishes nothing. Delete the tag and tag again: `git push origin :refs/tags/<tag>`.

**The recipe for a CLI release:**
1. In `CHANGELOG.md`, rename `[Unreleased]` to the version and date, and add a new empty `[Unreleased]` above it.
2. Merge.
3. Tag the merged commit: `git tag v0.2.0 && git push origin v0.2.0`.

The Release CLI workflow does the rest. Go's module proxy picks up the tag too, so `go install …@latest` gets the new version. Go's proxy keeps every tag forever: never move a `v*` tag that has been pushed. If a release is wrong, release the next version.

---

## 8. Things worth knowing

**Settings a user can change** (in `otel.env` or the shell):

| Setting | Effect |
|---|---|
| `OTEL_SERVICE_NAME` | The name shown in Grafana |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Where to send data |
| `OTEL_SDK_DISABLED=true` | Turns everything off, with no code change |
| `OTEL_NODE_ENABLED_INSTRUMENTATIONS` | Node: which libraries to instrument |
| `OTELSTARTER_JSON_LOGS=false` | Python: keep the app's own log format |
| `OTELSTARTER_ESM=true/false` | Node: force the ES-module hook on or off |
| `OTELSTARTER_CONSOLE_EVENTS=true` | Print domain events in the terminal (`run` sets it) |
| `OTELSTARTER_LOG_LEVEL` | Node SDK logger level |

**Limits today:**
- **Go apps** are recognised but not instrumented yet.
- **Node.js:**
  - ES-module apps need Node 22.14+.
  - Fastify and Prisma need their own plugins.
  - NestJS 12 works, but OpenTelemetry's NestJS instrumentation only covers up to 11, so traces show the route through Express, without Nest's own steps.
  - `console.log` is not sent; use pino, winston or bunyan.
- **Python:** Flask's debug reloader drops instrumentation; use `--no-reload` while testing.
- **Another monitoring agent** in the same app (Datadog, New Relic) is untested.

**Development gotchas:**
- Volta's `node` is a launcher. Signals sent to it do not reach the app.
- If `gh` is logged in to more than one GitHub account, it can end up on one without access to this repo, which then returns 404. Fix: `gh auth switch --user IvyMurage`.

---

## 9. Working on the project: everyday commands

```bash
# Go (the CLI)
go build ./...                      # compile
go run ./cmd/otelstarter help       # run without installing
gofmt -l . ; go vet ./... ; go test -race ./...   # what CI runs
go install ./cmd/otelstarter        # install your build as ~/go/bin/otelstarter

# Node SDK
cd sdk/node && npm ci && npm test

# Python SDK (in its own virtualenv)
cd sdk/python && pip install -e '.[dev]' && pytest && ruff check . && mypy

# Compatibility check (needs Postgres)
DATABASE_URL=postgres://postgres:pw@127.0.0.1:5432/postgres python3 compat/run.py [app]
```

**Where to start reading the code:**
1. `internal/cli/cli.go`, the menu of commands.
2. One command end to end: `init.go` → `setup/plan.go` → `adapters/python/setup.go`.
3. The Node SDK starts in `sdk/node/src/register.ts`.
4. The Python SDK starts in `sdk/python/pyproject.toml`'s entry points.
