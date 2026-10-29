# How we build OTelStarter

**Status:** These guidelines apply to every change, whoever or whatever writes it.
**For:** contributors, reviewers, and me, the maintainer.

## 1. Why this page exists

I started OTelStarter because adding observability to an app should take minutes, not a week of reading OpenTelemetry docs. The tool only keeps that promise if its own code stays easy to understand and safe to change. This page is how we keep it that way.

What I want for the codebase:
- Someone new to Go, Node.js or Python can read it and follow what it does.
- An experienced engineer can extend it with confidence.
- You can make a useful contribution without understanding the whole system.
- It stays simple as it grows.

The users of OTelStarter are developers who trust it with their apps. Contributors deserve the same care: a clear path, quick answers, and no surprises.

## 2. Using AI coding tools

You are welcome to use AI assistants such as Copilot or Claude. Everything on this page applies to the code they produce, exactly as if you had typed it, and you are responsible for it. In practice:
- Read every line before you open the pull request. Remove what is not needed.
- Run the checks yourself (section 33). Never write "tests pass" for tests that did not run.
- Keep the change focused (section 31). Assistants like to tidy unrelated code; leave it out.
- If an assistant suggests something complicated, ask whether a simpler version would do (section 4).

## 3. Before you change code

1. Read the [README](../README.md), the [project guide](PROJECT_GUIDE.md) and, for anything structural, the [architecture](ARCHITECTURE.md) and [requirements](REQUIREMENTS.md).
2. Look at the code you are about to change, and its tests.
3. Search for existing functions that already do part of the job.
4. Follow the patterns already in use, even where you would have chosen differently.
5. Find the smallest change that solves the problem.

Please do not start by creating new abstractions.

## 4. Keep it simple

Prefer the simplest correct solution. Before adding a framework, a dependency, an abstraction, a service, a configuration layer or an extension point, ask:

> Can this be simpler?

## 5. Build what is needed now

Do not build something because it might be useful one day. No unused settings, no future-proof abstractions, no extension points nobody uses yet.

## 6. Avoid duplication, but not at any price

Remove duplication when it is real. But three simple copies are often better than one clever abstraction that is hard to follow.

## 7. One job per function

Each function should have one clear job. A function that detects the project, installs packages, writes configuration and starts Docker is four functions.

Our own code shows the shape:
```text
detect.Detect()          finds the languages
setup.BuildPlan()        works out the changes, writing nothing
setup.Apply()            writes them
check.Run()              runs the checks
localstack.Stack.Start() starts Docker
```

## 8. Names

A name should say what the thing does: `BuildPlan`, `sanitizeQuery`, `redact_url`, `checkPrivacyExporters`.

Avoid names that say nothing, such as `process`, `handle`, `manager`, `helper`, `data` or `utils`, unless the context makes them clear.

Follow each language's conventions: `MixedCaps` in Go, `camelCase` in TypeScript, `snake_case` in Python.

## 9. Functions

Functions should:
- do one thing;
- take clear inputs and return clear outputs;
- avoid side effects nobody would expect;
- be easy to test.

## 10. Classes and types

Do not create a class when a function or a plain data structure is enough. A class or type should stand for something real: a concept in the domain, a component with state, or a clear boundary. Never a bag of unrelated helpers.

## 11. Reuse before you write

Before writing something new, search the code and the tests for something that already does it. Do not duplicate behaviour that exists. But do not force unrelated things together just to share code either.

## 12. When to add an abstraction

Add one when:
- several implementations really share behaviour;
- it makes the code clearer;
- it stands for a real concept;
- it is likely to stay stable.

Never only because it might be needed later.

## 13. Dependencies

Every dependency is code we did not write but must trust, update and ship. Before adding one, ask:
- Can the standard library do this?
- Can a dependency we already have do it?
- Is it maintained?
- Does it add security risk?
- Does it remove more complexity than it adds?

The CLI has a single dependency today. Please keep it that way unless there is a strong reason.

## 14. Configuration

Configuration should be explicit and predictable. Do not hard-code values that differ between environments, and do not make everything configurable either.

## 15. Error messages

Every error a user sees answers three questions:
1. What happened?
2. Why?
3. How do I fix it?

`otelstarter validate` is the model: each problem has a **Why** and a **Fix**.

## 16. OTelStarter's own logs

Our own output must be useful and must never contain passwords, tokens, secrets, API keys, private keys or credentials.

## 17. Security

Security is part of the design, not a later step. Think about:
- secrets and input validation;
- path traversal, command injection and unsafe file changes;
- dependency vulnerabilities;
- sensitive data in telemetry.

Anything that runs a command or changes a file needs a deliberate security review. That is why the CLI never runs a project's code, calls Docker without a shell, and writes files only through `os.Root`.

Report security problems privately, as described in [SECURITY.md](../SECURITY.md).

## 18. Privacy

Collect the minimum. Record `payment.completed`, not the payment. Business context means business meaning, never business payloads.

## 19. Telemetry cardinality

Never turn values with many possible settings into metric labels automatically: `user_id`, `request_id`, `transaction_id`, email, phone. Each distinct value becomes a separate series, and the backend's cost explodes.

## 20. Domain event names

Use `<domain>.<action>`, describing what happened in the business, not which function ran:

```text
payment.completed   payment.failed   order.created   order.cancelled
loan.submitted      loan.approved    user.authenticated
```

## 21. Business context

Add business metadata sparingly, for example `domain = payments`, `operation = payment`, `outcome = success`. The aim is the most diagnostic value from the least data.

## 22. Technical context

Developers should never have to add these by hand. Attach them automatically wherever possible:
- `service.name`, `service.version` and the environment;
- `trace_id` and `span_id`;
- the HTTP route and status;
- the runtime and the deployment.

## 23. OpenTelemetry first

When OpenTelemetry already does something (logs, metrics, traces, context propagation, resources, export), use it. OTelStarter is an opinionated experience around OpenTelemetry, not a replacement for it.

## 24. No lock-in to a backend

The SDKs must work with any OTLP backend. Grafana, Tempo, Loki, Prometheus and OpenObserve are choices the user makes, not ones we make for them. The local Grafana stack is a development convenience only.

## 25. Tests

Every meaningful change comes with tests. Depending on the change, that means:
- unit tests;
- tests of the CLI's output and exit codes;
- configuration tests;
- safety tests;
- adapter tests for each language;
- the compatibility check in `compat/`.

Test behaviour, not internals.

## 26. Mostly fast tests

Most tests should be fast unit tests, fewer should be integration tests, and only a few should be end-to-end:

```text
        end-to-end (compat/)
       /                    \
     integration (real processes, Docker)
    /                                    \
   unit tests
```

## 27. Backwards compatibility

What OTelStarter generates (`otel.env`, the install commands) and the commands themselves are a public interface: real projects depend on them. Before changing them, think about existing projects and users.

Document a breaking change in `CHANGELOG.md`, and give users a way through. When the Node.js setup moved away from the 80 MB bundle, `run` kept using the old loader and `validate` warned instead of failing.

## 28. Documentation

Every public feature explains:
- what it does and why;
- how to use it, with an example;
- its limitations.

If something cannot be explained simply, reconsider the design.

## 29. Comments

Comments explain **why**, not **what**. A comment that repeats the code is noise.

## 30. Where code lives

Put new code where its job belongs. Do not create a `utils`, `helpers` or `common` file:

```text
cmd/otelstarter/        the program's entry point, one line
internal/cli/           commands: parse flags, call packages, print results
internal/detect/        which language a project is
internal/adapters/      one folder per language: python/, node/
internal/setup/         the only code that writes to a user's project
internal/check/         the checks behind validate and doctor (read-only)
internal/localstack/    the local Collector and Grafana
sdk/node/, sdk/python/  the SDKs that run inside users' apps
compat/                 real apps, run with and without otelstarter
```

The [project guide](PROJECT_GUIDE.md) describes every file.

## 31. Focused changes

A pull request should have one clear purpose. Do not mix a feature with unrelated refactoring, formatting or dependency upgrades. Send those separately; they are welcome.

## 32. Review your own diff

Before asking for a review, read your own diff: the files changed, added and deleted, the dependencies, the tests, the docs, and anything that touches security or telemetry. Remove what is not needed.

## 33. Run the checks

Before saying a change is done:
- run the tests;
- run the linters and type checks;
- check any configuration you generate;
- run the relevant integration tests, or the compatibility check in `compat/`.

[CONTRIBUTING.md](../CONTRIBUTING.md) lists the commands for each part. If you could not run something, say what and why. Never claim a check passed if it did not run.

## 34. Push back on bad designs

If an issue, a reviewer, or I ask for something needlessly complicated, say so and suggest a simpler way. I want good engineering, not agreement.

## 35. What a contributor should find easily

Without asking anyone, a contributor should be able to find out:
- how to set up and run the project ([CONTRIBUTING.md](../CONTRIBUTING.md));
- how it fits together ([project guide](PROJECT_GUIDE.md), [architecture](ARCHITECTURE.md));
- where the language adapters and the tests live (section 30);
- how to add support for a library or a language ([adding a library or a language](adding-a-language.md));
- how to open a pull request, and what happens next ([CONTRIBUTING.md](../CONTRIBUTING.md)).

If any of this is hard to find, that is a bug. Please report it.

## 36. Adding a language

A language adapter is self-contained:

```text
internal/adapters/<language>/
├── <language>.go     reads the manifests, detects frameworks and libraries
├── instrumentation   recommends packages (a file or a section)
├── setup.go          what init writes, pinned versions, how run starts the app
└── *_test.go
```

Adding one should not mean changing unrelated parts of the system. [Adding a library or a language](adding-a-language.md) walks through it.

## 37. Pull requests

The pull request template asks these questions. Answer the ones that apply; a sentence each is enough:
- What problem does this solve?
- Why this way? Which alternatives did you consider?
- What tests did you add or run?
- Does it add a dependency?
- Does it affect performance, security, or how much telemetry users send?
- Does it break anything for existing users?

## 38. Decisions

Write significant decisions down as an ADR in [`docs/adr/`](adr/): context, decision, alternatives, consequences. The ten existing ones show the format.

## 39. When a change is done

A change is done when:
- it works, and has tests;
- its documentation is updated;
- security, privacy and performance have been considered;
- its errors explain themselves;
- existing behaviour still works;
- the code is easy to read, and the diff is focused.

## 40. The short version

1. Understand before you change.
2. Prefer simple solutions.
3. Reuse before you create.
4. Do not abstract early.
5. Keep functions focused.
6. Name things clearly.
7. Security is a requirement.
8. Privacy is a requirement.
9. Collect useful telemetry, not all of it.
10. Business context is not business payload.
11. Build on OpenTelemetry; do not reinvent it.
12. Do not tie the SDKs to a backend.
13. Test behaviour.
14. Document what users see.
15. Keep pull requests focused.
16. Push back on unnecessary complexity.
17. Never claim a check you did not run.

## 41. In one sentence

> **Simple enough for a beginner to understand, structured enough for a senior engineer to extend, documented enough for anyone to contribute, and explicit enough that any change, by a person or with an AI assistant, can be made safely.**

And for the product:

> **Collect technical context automatically, capture business context deliberately, connect logs, metrics, traces and domain events, and send only the telemetry that is worth having.**
