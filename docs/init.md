# otelstarter init

## What it does

`otelstarter init` adds OpenTelemetry to a Python project:

1. It creates **`otel.env`**, a file of standard OpenTelemetry settings: service name, environment, Collector endpoint and log correlation.
2. It adds the **instrumentation packages** that [`otelstarter analyze`](analyze.md) recommends, pinned to versions otelstarter has tested together.
3. It prints the **start command**, `otelstarter run -- <your command>`, which starts a local Collector and Grafana and then your app with instrumentation on, with no code changes.

It shows every change first and asks before writing anything.

## Why

Setting up OpenTelemetry by hand means choosing between dozens of packages, matching their versions, and finding the right environment variables. `init` does that from what it detects in the project.

## Usage

```bash
otelstarter init              # current folder
otelstarter init ./my-app     # another folder
otelstarter init --dry-run    # show the plan, change nothing
otelstarter init --yes        # apply without asking (for scripts and CI)
```

Without a terminal (in CI, or with input piped in), `init` refuses to write unless `--yes` is given, so a script never changes files by accident.

## Example

```text
$ otelstarter init
Project: /home/dev/shop (Python, FastAPI, pip)

Files to change:
  create  otel.env           OpenTelemetry settings for this service
  update  requirements.txt   add 7 packages in a block marked "otelstarter"

otel.env (new file):
  | ...
  | OTEL_SERVICE_NAME=shop
  | ...

requirements.txt (lines added):
  | opentelemetry-distro==0.66b0
  | opentelemetry-exporter-otlp-proto-http==1.45.0
  | ...

Then run:
  pip install -r requirements.txt
  otelstarter run -- uvicorn main:app
  (replace the command after -- with your own; run starts a local Collector and Grafana first)

Apply these changes? [y/N]
```

## What changes, by package manager

| Package manager | Dependencies | You run |
|-----------------|--------------|---------|
| pip | Added to `requirements.txt` inside a marked block | `pip install -r requirements.txt` |
| uv | Not edited | `uv add …` (printed in full) |
| Poetry | Not edited | `poetry add …` |
| PDM | Not edited | `pdm add …` |
| Pipenv | Not edited | `pipenv install …` |

`pyproject.toml` and `Pipfile` are never rewritten, because doing so from outside Python would lose your comments and formatting. The printed command lets your own package manager make the change.

The block in `requirements.txt` looks like this. Delete it to undo the change:

```text
# >>> otelstarter: OpenTelemetry packages
opentelemetry-distro==0.66b0
...
# <<< otelstarter
```

## Node.js projects

`init` writes `otel.env` and prints the install command; it never edits `package.json`:

| Package manager | You run |
|-----------------|---------|
| npm | `npm install --save-exact @opentelemetry/api@1.9.1 @opentelemetry/auto-instrumentations-node@0.80.0` |
| yarn / pnpm / bun | the same packages with `yarn add --exact`, `pnpm add --save-exact` or `bun add --exact` |

`otel.env` gets two Node.js settings on top of the shared ones:
- **`OTEL_NODE_ENABLED_INSTRUMENTATIONS`** loads only the instrumentations for libraries the app uses, for example `http,undici,runtime-node,express,pg,pino`.
- **`OTEL_NODE_RESOURCE_DETECTORS=env,host,os,process`** stops startup from probing cloud metadata servers, which prints a warning outside those clouds.

The start command is `otelstarter run -- npm start`, or your own command. Logs reach the backend only through pino, winston or bunyan: OpenTelemetry for Node does not export `console.log`.

## otel.env

`otel.env` contains only standard [OpenTelemetry environment variables](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/), so it also works with Docker's `env_file` and Kubernetes, without otelstarter installed.

| Variable | Default | Meaning |
|----------|---------|---------|
| `OTEL_SERVICE_NAME` | `[project].name` from `pyproject.toml`, else the folder name | Name shown in every trace, metric and log |
| `OTEL_RESOURCE_ATTRIBUTES` | `deployment.environment.name=local` | Where the service runs |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` | The Collector that receives the telemetry |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` | Matches the HTTP exporter that `init` installs |
| `OTEL_TRACES_EXPORTER`, `OTEL_LOGS_EXPORTER` | `otelstarter` | Python: send through the otelstarter SDK's exporters (0.3.0+), which replace SQL values with `?` and redact secrets first. Console logs become structured JSON with trace IDs |

Edit it freely. `init` never overwrites an existing `otel.env`. To regenerate it, delete it and run `init` again.

## Running it again

`init` is safe to run repeatedly:

- packages already in your manifest are not added again
- an existing `otel.env` is left alone
- when everything is in place it prints `Already set up. Nothing to change.`

If you add a library later (for example Redis), running `init` again adds only its instrumentation package.

## Versions

OpenTelemetry's Python packages require each other at exact versions, so otelstarter pins a set it has tested together: `0.66b0` for the distro and instrumentation packages, and `1.45.0` for the exporter. If the project already lists OpenTelemetry packages at other versions, `init` warns you to align them.

## Production

`otel.env` is set up for local development: it records every trace and sends to `localhost`. For other environments:
- set `OTEL_EXPORTER_OTLP_ENDPOINT` to your Collector
- set `OTEL_RESOURCE_ATTRIBUTES=deployment.environment.name=<env>`
- consider sampling traces to reduce cost, for example `OTEL_TRACES_SAMPLER=parentbased_traceidratio` with `OTEL_TRACES_SAMPLER_ARG=0.1`

See [performance](performance.md) for what instrumentation costs.

### Node.js in production: SIGTERM

OpenTelemetry's loader (`@opentelemetry/auto-instrumentations-node/register`) handles **SIGTERM** by flushing telemetry, but it **does not exit the process**. Docker, Kubernetes and most process managers stop apps with SIGTERM, so an instrumented app keeps running until it is force-killed (usually after 10–30 seconds).

Until otelstarter ships its own loader, give your app a SIGTERM handler that shuts the server down and exits:

```js
process.on('SIGTERM', () => server.close(() => process.exit(0)));
```

`otelstarter run` is not affected: Ctrl+C sends SIGINT.

## Limitations

- Python and Node.js for now (CommonJS apps; ES modules are next). Go projects get a clear error. A folder with both Python and Node.js is set up as Python.
- **Start the app with `otel.env` loaded.** [`otelstarter run -- <command>`](run.md) does this for you. Without Docker, load it yourself: `set -a && . ./otel.env && set +a && opentelemetry-instrument <command>`. Without `otel.env`, the SDK looks for the gRPC exporter, which is not installed. The app keeps working, but OpenTelemetry prints `Configuration of configurator failed` and sends nothing.
- `init` does not install packages or run your package manager. You run the printed command.
- `init` adds missing packages but **does not change versions already pinned**. `otelstarter validate` warns when the `otelstarter` SDK pin is older than the current one.
- A project whose dependencies live only in `setup.py` gets a `pip install …` command. Add the packages to your dependency list yourself.

## Security and privacy

- Changes are shown in full and need confirmation (or `--yes`).
- Files are written only inside the project folder, through `os.Root`. Symbolic links are never replaced.
- Each file is written to a temporary file first and then renamed into place, so a crash cannot leave half a file. If a file changes between the preview and your confirmation, nothing is written.
- `init` runs no commands.
- The service name is limited to letters, digits, `.`, `_` and `-`, so a crafted project name cannot inject extra settings into `otel.env`.
- `otel.env` is meant to be committed. Never put secrets such as API keys in it.
