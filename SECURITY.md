# Security

## Reporting a vulnerability

Please report security problems privately, not in a public issue. On GitHub, open the repository's **Security** tab and choose **Report a vulnerability**. Include what you found, how to reproduce it, and which version you used (`otelstarter version`, or the SDK version).

You will get an answer within a week. Once a fix is released, the report is credited unless you prefer otherwise.

## What otelstarter is designed to guarantee

- **The CLI never runs a project's code.** It only reads manifest files, inside the project folder.
- **The only command that changes files is `otelstarter init`.** It shows its plan and asks first.
- **The local stack listens on `127.0.0.1` only.**
- **The SDKs redact secrets and replace SQL values with `?` before telemetry leaves the app.** The rules are in [docs/TELEMETRY_CONVENTIONS.md](docs/TELEMETRY_CONVENTIONS.md).

A way around any of these is a security problem worth reporting.
