# otelstarter analyze

## What it does

`otelstarter analyze` reads a project's manifest files and reports its language, package manager, web framework, the dependencies that matter for observability, and any OpenTelemetry packages already installed.

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
```

Exit codes: `0` when a project was analysed, `1` when no supported project was found or a manifest could not be read, `2` for wrong usage.

## What is detected

| Language | Identified by | Dependency detection |
|----------|---------------|----------------------|
| Python | `pyproject.toml`, `requirements.txt`, `Pipfile`, `setup.py` | Yes |
| Node.js | `package.json` | Not yet |
| Go | `go.mod` | Not yet |

Python details:

- **Manifests read:** `[project].dependencies` and `[tool.poetry.dependencies]` in `pyproject.toml`, `requirements.txt`, and `[packages]` in `Pipfile`.
- **Package manager:** taken from the lock file (`uv.lock`, `poetry.lock`, `pdm.lock`, `Pipfile.lock`), else from the manifest.
- **Frameworks:** FastAPI, Flask, Django.
- **Dependencies:** PostgreSQL, MySQL, Redis, Kafka, RabbitMQ, HTTP clients, ORMs, Celery. The full list is in `internal/adapters/python/python.go`.

## Limitations

- Only the top level of the folder is read. In a monorepo, pass the path of each service.
- `setup.py` is never read, because it is Python code. Move dependencies to `pyproject.toml` or `requirements.txt`.
- Optional and development dependencies are ignored, since they rarely run in production.
- A `requirements.txt` produced by `pip freeze` lists indirect dependencies too, so a library may be reported even though your code does not use it directly.
- Only packages on the known list are reported. An ORM such as SQLAlchemy is reported as an ORM, not as a specific database.

## Security and privacy

- Project code is never executed, and no package manager is run.
- Files are opened through Go's `os.Root`, which refuses any path or symlink leading outside the project folder.
- Manifests larger than 1 MiB are refused.
- Output contains only package names from the known list, never other file contents.
