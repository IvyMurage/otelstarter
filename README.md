# OTelStarter

OTelStarter is a command-line tool that adds production-quality observability to an existing application: structured logs, metrics, traces, correlation between them, and business-aware domain events.

It is a thin layer around [OpenTelemetry](https://opentelemetry.io/), not an observability backend. Your application emits standard OpenTelemetry data and you choose where it goes.

```bash
otelstarter init                      # add OpenTelemetry to your app
otelstarter run -- uvicorn main:app   # start a local Collector + Grafana, then your app
```

**New here? Follow the [quickstart](docs/quickstart.md):** a Python app with traces, logs, metrics and business events in about 10 minutes.

## Install

**Linux and macOS:** download the latest release and put it on your `PATH`:

```bash
os=$(uname -s | tr '[:upper:]' '[:lower:]'); arch=$(uname -m)
case "$arch" in x86_64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; esac
curl -fsSL "https://github.com/IvyMurage/otelstarter/releases/latest/download/otelstarter_${os}_${arch}.tar.gz" | tar -xz otelstarter
sudo mv otelstarter /usr/local/bin/
otelstarter version
```

**Windows:** download `otelstarter_windows_amd64.zip` (or `_arm64`) from the [latest release](https://github.com/IvyMurage/otelstarter/releases/latest), unzip it, and put `otelstarter.exe` in a folder on your `PATH`.

Each release lists SHA-256 checksums in `checksums.txt`. The binaries are not code-signed yet:
- **macOS:** a file downloaded with a browser (not `curl`) is blocked the first time. Allow it with `xattr -d com.apple.quarantine otelstarter`.
- **Windows:** SmartScreen may ask you to confirm.

**With Go 1.25 or newer:**

```bash
go install github.com/IvyMurage/otelstarter/cmd/otelstarter@latest
```

Go puts the program in `~/go/bin`. If `otelstarter` is then "command not found", add that folder to your `PATH`:

```bash
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.zshrc   # or ~/.bashrc
```

`otelstarter run` also needs Docker, and Python apps need Python 3.10 or newer.

## Status

Early development. Working commands:

- `otelstarter analyze`: detect a project's language, framework and dependencies, and recommend OpenTelemetry instrumentation ([docs](docs/analyze.md)). Python and Node.js are analysed in full; Go is recognised but not analysed yet. `init`, `run` and `validate` support Python and Node.js (CommonJS and ES modules).
- `otelstarter init`: add OpenTelemetry to a Python project ([docs](docs/init.md)).
- `otelstarter run`: start a local Collector and Grafana, and optionally your app ([docs](docs/run.md)).
- `otelstarter validate` and `otelstarter doctor`: check the setup and explain how to fix problems ([docs](docs/validate.md)).
- `otelstarter version` and `otelstarter help`.

Domain events for Python apps: `observability.event("payment.completed", domain="payments", outcome="success")`. See the [Python SDK](sdk/python/README.md).

## Build from source

From a clone of this repository:

```bash
go install ./cmd/otelstarter    # or: go build -o otelstarter ./cmd/otelstarter
```

## Documentation

- [Quickstart](docs/quickstart.md): from an app to traces, logs, metrics and events
- [Performance](docs/performance.md): what otelstarter and instrumentation cost
- [Project guide](docs/PROJECT_GUIDE.md): the whole project in plain English, from how a request flows through it to what every file does
- [Architecture](docs/ARCHITECTURE.md): system design and boundaries
- [Requirements](docs/REQUIREMENTS.md): what the product must do
- [Engineering guidelines](docs/ENGINEERING_GUIDELINES.md): the rules every change follows
- [Domain events](docs/DOMAIN_EVENTS.md): the business event model
- [Architecture decision records](docs/adr/)
- [Contributing](CONTRIBUTING.md): from a fresh clone to a merged pull request
- [Adding a library or a language](docs/adding-a-language.md)

## License

[Apache License 2.0](LICENSE)
