# otelstarter analyze

## What it does

`otelstarter analyze` reads a project's manifest files and reports its language, package manager, web framework, the dependencies that matter for observability, and any OpenTelemetry packages already installed.

For Python it also recommends the OpenTelemetry packages to install and shows how to start the app with them.

It changes nothing. Later commands (`init`, `doctor`) build on the same detection.

## Why

Choosing OpenTelemetry instrumentation starts with knowing what the application uses. A FastAPI service talking to PostgreSQL through asyncpg needs different instrumentation packages from a Flask app using requests.

## Usage

```bash
otelstarter analyze            # analyse the current folder
otelstarter analyze ./my-app   # analyse another folder
```

## Example

```text
$ otelstarter analyze ./shop
Project: /home/dev/shop

  Language:         Python
  Found:            pyproject.toml
  Package manager:  uv
  Framework:        FastAPI
  Dependencies:
    HTTP client      httpx
    ORM              sqlalchemy
    PostgreSQL       asyncpg
    Redis            redis
  OpenTelemetry:    not installed

  Recommended instrumentation (✓ installed, + to add):
    + opentelemetry-distro                          SDK and the opentelemetry-instrument launcher
    + opentelemetry-exporter-otlp-proto-http        Sends telemetry to an OpenTelemetry Collector over HTTP
    + opentelemetry-instrumentation-logging         Adds trace and span IDs to log records
    + opentelemetry-instrumentation-system-metrics  Runtime metrics: CPU, memory, garbage collection
    + otelstarter                                   Domain events: observability.event("order.created", ...)
    + opentelemetry-instrumentation-asyncpg         Database queries (asyncpg)
    + opentelemetry-instrumentation-fastapi         Incoming HTTP requests: traces and request metrics (fastapi)
    + opentelemetry-instrumentation-httpx           Outgoing HTTP requests (httpx)
    + opentelemetry-instrumentation-redis           Redis commands (redis)
    + opentelemetry-instrumentation-sqlalchemy      ORM queries (sqlalchemy)

  Start with:       opentelemetry-instrument uvicorn main:app
                    (replace the example with your own start command)

  Note: OpenTelemetry's Python packages need Python 3.10 or newer.
```

Exit codes: `0` when a project was analysed, `1` when no supported project was found or a manifest could not be read, `2` for wrong usage.

## What is detected

| Language | Identified by | Dependency detection |
|----------|---------------|----------------------|
| Python | `pyproject.toml`, `requirements.txt`, `Pipfile`, `setup.py` | Yes |
| Node.js | `package.json` | Yes |
| Go | `go.mod` | Not yet |

Python details:

- **Manifests read:** `[project].dependencies` and `[tool.poetry.dependencies]` in `pyproject.toml`, `requirements.txt`, and `[packages]` in `Pipfile`.
- **Package manager:** taken from the lock file (`uv.lock`, `poetry.lock`, `pdm.lock`, `Pipfile.lock`), else from the manifest.
- **Frameworks:** FastAPI, Flask, Django.
- **Dependencies:** PostgreSQL, MySQL, Redis, Kafka, RabbitMQ, HTTP clients, ORMs, Celery. The full list is in `internal/adapters/python/python.go`.

Node.js details:

- **Manifest read:** `dependencies` in `package.json`. `devDependencies` are ignored.
- **Package manager:** taken from the lock file (`pnpm-lock.yaml`, `yarn.lock`, `bun.lock`, `package-lock.json`); npm when there is none.
- **Frameworks:** Express, NestJS, Koa, hapi, Fastify.
- **Dependencies:** PostgreSQL, MySQL, Redis, MongoDB, Kafka, RabbitMQ, knex, Prisma, GraphQL, axios, and the logging libraries pino, winston and bunyan.

## Recommended instrumentation (Node.js)

Node.js apps get the `otelstarter` SDK, plus one `@opentelemetry/instrumentation-<name>` package per detected library. For example, an Express + pg app gets `otelstarter`, `@opentelemetry/instrumentation-express` and `@opentelemetry/instrumentation-pg`. The SDK loads with zero code changes through `--require otelstarter/register`, and already includes http, `fetch` (undici), runtime metrics and pino. winston also gets `@opentelemetry/winston-transport`, which sends its logs.

**Only what the app uses is switched on.** `analyze` lists the instrumentations to enable: always `http`, `undici` (for `fetch`) and `runtime-node`, plus one per detected library, for example `http, undici, runtime-node, express, pg, pino`. `init` sets this list as `OTEL_NODE_ENABLED_INSTRUMENTATIONS`, so the other instrumentations are never loaded.

Notes explain the gaps:
- **No logging library.** OpenTelemetry for Node does not export `console.log`; logs reach the backend only through pino, winston or bunyan.
- **ES-module apps** (`"type": "module"`) are not supported yet.
- **Fastify and Prisma** need their own instrumentation packages (`@fastify/otel`, `@prisma/instrumentation`).
- **`@opentelemetry/auto-instrumentations-node`,** the 80 MB bundle earlier otelstarter versions installed. `analyze` suggests moving to the SDK.

## Recommended instrumentation (Python)

Every Python service gets five base packages:

| Package | Why |
|---------|-----|
| `opentelemetry-distro` | The SDK, plus the `opentelemetry-instrument` command that turns instrumentation on without code changes |
| `opentelemetry-exporter-otlp-proto-http` | Sends logs, metrics and traces to an OpenTelemetry Collector over HTTP. The gRPC exporter is left out because it pulls in `grpcio`, a large compiled package; `otelstarter init` sets `OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf` to match |
| `opentelemetry-instrumentation-logging` | Adds `trace_id` and `span_id` to Python log records, so logs can be matched to traces |
| `opentelemetry-instrumentation-system-metrics` | CPU, memory and garbage-collection metrics |
| `otelstarter` | The domain-event SDK, for recording business events with `observability.event(...)`. See the [Python SDK](../sdk/python/README.md) |

Each detected framework or library adds its matching instrumentation package. Only packages published by the OpenTelemetry project are recommended. Libraries without one (currently `aiomysql` and `peewee`) are reported in a note, because their calls will be missing from traces.

Packages already installed are marked `✓`. `otelstarter init` adds the missing ones with pinned versions; see [init](init.md).

### Verified

For a FastAPI app using httpx, installing exactly the recommended packages and starting with `opentelemetry-instrument uvicorn main:app` produced:

- one trace spanning the incoming request, the outgoing httpx call and the downstream service
- log records carrying the same `trace_id`
- HTTP server and client duration metrics, plus CPU, memory and garbage-collection metrics

## Limitations

- Only the top level of the folder is read. In a monorepo, pass the path of each service.
- `setup.py` is never read, because it is Python code. Move dependencies to `pyproject.toml` or `requirements.txt`.
- Optional and development dependencies are ignored, since they rarely run in production.
- A `requirements.txt` produced by `pip freeze` lists indirect dependencies too, so a library may be reported even though your code does not use it directly.
- Running more than one worker process (for example `--workers 4`) can break metrics under `opentelemetry-instrument`. See the [OpenTelemetry troubleshooting guide](https://opentelemetry.io/docs/zero-code/python/troubleshooting/).
- Only packages on the known list are reported. An ORM such as SQLAlchemy is reported as an ORM, not as a specific database.

## Security and privacy

- Project code is never executed, and no package manager is run.
- Files are opened through Go's `os.Root`, which refuses any path or symlink leading outside the project folder.
- Manifests larger than 1 MiB are refused.
- Output contains only package names from the known list, never other file contents.
