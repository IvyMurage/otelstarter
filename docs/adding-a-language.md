# Adding a library or a language

Most contributions here are **libraries**: "my app uses X, and otelstarter does not set it up". That is usually two small edits and a test. A whole **language** is a bigger piece of work, and the second half of this page walks through it.

## Adding a library

otelstarter recognises a library from the app's manifest (`package.json`, `requirements.txt`…) and recommends OpenTelemetry's instrumentation package for it. Adding a library means teaching it that pair.

### First, check an instrumentation exists

OTelStarter builds on OpenTelemetry's own instrumentations; it does not write its own. Look the library up:
- **Node.js:** [opentelemetry-js-contrib](https://github.com/open-telemetry/opentelemetry-js-contrib/tree/main/packages). The package is named `@opentelemetry/instrumentation-<name>`.
- **Python:** [opentelemetry-python-contrib](https://github.com/open-telemetry/opentelemetry-python-contrib/tree/main/instrumentation). The package is named `opentelemetry-instrumentation-<name>`.

If none exists, the library can still be recognised, and `analyze` can explain the gap in a note, as it does for Fastify and Prisma.

### Node.js: an example with `socket.io`

1. **Recognise it.** In `internal/adapters/node/node.go`, add an entry to `knownPackages`. The key is the npm name in `package.json`; `instrumentation` is the part after `@opentelemetry/instrumentation-`:
   ```go
   "socket.io": {category: "WebSockets", instrumentation: "socket.io"},
   ```
2. **Pin a version that matches the SDK.** All instrumentations share one core package, `@opentelemetry/instrumentation`. Pick the version built on the same core as the SDK, so npm installs the core once. The newest release is not always the right one: when OpenTelemetry moves to a new core, the instrumentations move first.

   Find the SDK's core version:
   ```bash
   grep '"@opentelemetry/instrumentation"' sdk/node/package.json     # "~0.223.0"
   ```
   Then list which core each instrumentation version needs, and take the newest that matches:
   ```bash
   for v in $(npm view @opentelemetry/instrumentation-socket.io versions --json | tr -d '[]",' | tail -5); do
     echo "$v needs $(npm view @opentelemetry/instrumentation-socket.io@$v dependencies.@opentelemetry/instrumentation)"
   done
   # 0.70.0 needs ^0.223.0   <- this one
   # 0.71.0 needs ^0.300.0
   ```
   Add it to `pins` in `internal/adapters/node/setup.go`:
   ```go
   instrumentationPackage("socket.io"): "0.70.0",
   ```
   Forget the pin and `TestEveryRecommendedPackageIsPinned` fails, so it cannot slip through.
3. **If the instrumentation needs a companion package** to do its whole job, as winston needs `@opentelemetry/winston-transport` to send logs, add it to `companions` and pin it too.
4. **Test it.** In `internal/adapters/node/node_test.go`, add a case showing `analyze` recommends the package for a `package.json` that uses the library. `TestAnalyzeExpressApp` is a good model.

Nothing else changes: the SDK loads `@opentelemetry/instrumentation-<name>` for every name in `OTEL_NODE_ENABLED_INSTRUMENTATIONS`, which `init` writes.

### Python: an example with `pymongo`

1. **Recognise it.** In `internal/adapters/python/python.go`, add an entry to `knownPackages`. Use the name as it appears in `requirements.txt`, lower-case, with `-` rather than `_`:
   ```go
   "pymongo": {category: "MongoDB", instrumentation: "opentelemetry-instrumentation-pymongo"},
   ```
2. **Describe what it adds.** If the category is new, add a line to `categoryPurpose` in `internal/adapters/python/instrumentation.go`, e.g. `"MongoDB": "Database queries"`. `analyze` shows it next to the package.
3. **No pin to add.** Python instrumentation packages are released together, and `init` pins them all to the same tested version (`contribVersion` in `setup.go`).
4. **Test it.** Add a case to `internal/adapters/python/python_test.go`.

### For both

- Update the list of supported libraries in `docs/analyze.md`.
- Add a line to `CHANGELOG.md` under `[Unreleased]`.
- Try it on a real app if you can:
  ```bash
  go run ./cmd/otelstarter analyze path/to/app
  go run ./cmd/otelstarter init --dry-run path/to/app
  ```

## Adding a language

Please open an issue before starting, so we can agree on how instrumentation works for that language. It differs a lot between languages: Python and Node.js can instrument an app from the outside, while Go cannot, for example.

A language touches these places. The Node.js adapter is the best model, because it is the most recent.

| Step | Where | What |
|---|---|---|
| 1. Detect it | `internal/detect/detect.go` | Add the language and its manifest files to the `languages` list, with its `Analyze` function. Go is listed already, with no adapter yet |
| 2. Analyse it | `internal/adapters/<language>/<language>.go` | `Manifests`, and `Analyze(fs.FS) (project.Language, error)`. Read the manifest files only, never run anything. Fill in frameworks, dependencies, recommended packages, the package manager, a start command, and notes for gaps |
| 3. Set it up | `internal/adapters/<language>/setup.go` | What `init` writes to `otel.env` (`EnvVars`), the install command per package manager, tested version pins, and how `run` starts an app with instrumentation |
| 4. Wire `init` | `internal/setup/plan.go` | Choose the language's env vars and install command (see `planPackages` and `languageEnvVars`) |
| 5. Wire `run` | `internal/cli/run.go`, `prepareApp` | Start the app with instrumentation switched on, as `NODE_OPTIONS` does for Node.js and `opentelemetry-instrument` for Python |
| 6. Check it | `internal/check/<language>.go`, `check.go` | Checks for `validate`: runtime version, packages installed, anything that would silently send nothing. Each problem has a Why and a Fix |
| 7. Test it | `*_test.go` next to each file | Use `fstest.MapFS` for fake projects, as the existing adapter tests do |
| 8. Prove it | `compat/apps/<app>` and `compat/run.py` | A small real app with the same routes as the others, so CI shows it behaves the same with and without otelstarter |
| 9. Document it | `docs/analyze.md`, `init.md`, `run.md`, `validate.md`, the README, `CHANGELOG.md`, and an ADR for the main decisions | |

If the language needs its own SDK, as Node.js and Python have, it lives in `sdk/<language>/`. It must pass the shared test cases in `sdk/conventions.json`, so redaction and query cleaning behave the same in every language. See [TELEMETRY_CONVENTIONS.md](TELEMETRY_CONVENTIONS.md).
