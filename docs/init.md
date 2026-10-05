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
  update  requirements.txt   add 6 OpenTelemetry packages in a block marked "otelstarter"

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

## otel.env

`otel.env` contains only standard [OpenTelemetry environment variables](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/), so it also works with Docker's `env_file` and Kubernetes, without otelstarter installed.

| Variable | Default | Meaning |
|----------|---------|---------|
| `OTEL_SERVICE_NAME` | `[project].name` from `pyproject.toml`, else the folder name | Name shown in every trace, metric and log |
| `OTEL_RESOURCE_ATTRIBUTES` | `deployment.environment.name=local` | Where the service runs |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` | The Collector that receives the telemetry |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` | Matches the HTTP exporter that `init` installs |
| `OTEL_PYTHON_LOG_CORRELATION` | `true` | Adds `trace_id` and `span_id` to log messages |

Edit it freely. `init` never overwrites an existing `otel.env`. To regenerate it, delete it and run `init` again.

## Running it again

`init` is safe to run repeatedly:

- packages already in your manifest are not added again
- an existing `otel.env` is left alone
- when everything is in place it prints `Already set up. Nothing to change.`

If you add a library later (for example Redis), running `init` again adds only its instrumentation package.

## Versions

OpenTelemetry's Python packages require each other at exact versions, so otelstarter pins a set it has tested together: `0.66b0` for the distro and instrumentation packages, and `1.45.0` for the exporter. If the project already lists OpenTelemetry packages at other versions, `init` warns you to align them.

## Limitations

- Python only for now. Node.js and Go projects get a clear error.
- **Start the app with `otel.env` loaded.** [`otelstarter run -- <command>`](run.md) does this for you. Without Docker, load it yourself: `set -a && . ./otel.env && set +a && opentelemetry-instrument <command>`. Without `otel.env`, the SDK looks for the gRPC exporter, which is not installed. The app keeps working, but OpenTelemetry prints `Configuration of configurator failed` and sends nothing.
- `init` does not install packages or run your package manager. You run the printed command.
- A project whose dependencies live only in `setup.py` gets a `pip install …` command. Add the packages to your dependency list yourself.

## Security and privacy

- Changes are shown in full and need confirmation (or `--yes`).
- Files are written only inside the project folder, through `os.Root`. Symbolic links are never replaced.
- Each file is written to a temporary file first and then renamed into place, so a crash cannot leave half a file. If a file changes between the preview and your confirmation, nothing is written.
- `init` runs no commands.
- The service name is limited to letters, digits, `.`, `_` and `-`, so a crafted project name cannot inject extra settings into `otel.env`.
- `otel.env` is meant to be committed. Never put secrets such as API keys in it.
