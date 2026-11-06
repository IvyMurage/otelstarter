# Reviewing and merging pull requests

**For:** maintainers and reviewers. If you are contributing, you want [CONTRIBUTING.md](../../CONTRIBUTING.md) instead.

Contributors give us their time. The way we review decides whether they come back. Every pull request gets a reply, every suggestion comes with a reason, and nobody waits in silence.

## Targets

These are the promises in [CONTRIBUTING.md](../../CONTRIBUTING.md#7-what-happens-next-and-how-long-it-takes). Keep them, or change them there.

| Step | Target |
|---|---|
| Approve the CI run of a first-time contributor | within 2 working days |
| First review | within 5 working days |
| Follow-up review, after the contributor pushes | 1–2 working days |

If you cannot review properly in time, leave a short comment saying when you will: "Thanks! I'll look at this properly on Friday." Silence is the one thing that always loses a contributor.

The [open pull requests](https://github.com/IvyMurage/otelstarter/pulls) page, sorted by oldest, shows who has waited longest.

## 1. A first-time contributor: approve the CI run

GitHub does not run CI for someone's first pull request until a maintainer approves it. The pull request shows **"Workflows awaiting approval"** and an **Approve and run workflows** button.

Our CI runs on `pull_request`, so a fork's code runs with a read-only token and **no secrets**: it cannot publish to npm or PyPI, or push to the repository. The worst a hostile pull request can do is waste CI minutes. Still, look at the **Files changed** tab before you click, and stop if you see:

- changes under `.github/` (workflows, scripts, settings files);
- new or changed dependencies (`go.mod`, `package.json`, `pyproject.toml`) or install scripts (`"postinstall"`);
- tests or `compat/` apps that download, upload or run anything unexpected;
- anything unrelated to what the pull request says it does.

If any of these appear, ask about them first, and leave changes to `.github/` to the owner.

Only people with **Write** access can approve a CI run.

## 2. Review the change

Read the description first, then the diff. Ask, in this order:

**Is it wanted?**
- Does it solve a real problem, ideally one in an issue?
- Is it one change? A fix plus an unrelated refactor should be two pull requests ([guidelines §31](../ENGINEERING_GUIDELINES.md#31-focused-changes)).
- If it is a large change nobody discussed, say so kindly, before reviewing line by line.

**Is it right?**
- Does the code do what the description says? Think of an input that would break it.
- Are there tests, and would they fail without the change?
- Is CI green? If a check fails, help read the log; contributors often do not know where to look.

**Does it fit?** The [engineering guidelines](../ENGINEERING_GUIDELINES.md) in short:
- the simplest version that works, with no abstraction "for later" (§4, §5);
- no new dependency without a strong reason; the CLI has a single one (§13);
- names that say what a thing does, code where its job belongs, no `utils` (§8, §30);
- errors that say what happened, why, and how to fix it (§15).

**Is it safe for users?** Look hardest here:
- **Privacy.** Anything that changes what the SDKs send: redaction, query cleaning, log fields, new attributes. Redaction changes need a case in [`sdk/conventions.json`](../../sdk/conventions.json), so both SDKs test the same rule.
- **What `init` writes.** `otel.env`, install commands and version pins are a public interface; real projects depend on them ([§27](../ENGINEERING_GUIDELINES.md#27-backwards-compatibility)).
- **Startup and shutdown.** Changes to how the SDKs load, export or stop should pass the compatibility check in `compat/`. CI runs it.
- **Version pins.** A new Node.js instrumentation must be pinned to the version built on the SDK's OpenTelemetry core ([how to check](../adding-a-language.md#nodejs-an-example-with-socketio)).

**Is it documented?**
- Docs updated if users will notice.
- A line in `CHANGELOG.md` under `[Unreleased]` if users will notice. CI, templates and contributor docs do not need one.

### Changes the owner reviews

Some areas can hurt every user or the project itself. A reviewer may comment on them, but the owner approves them:

- `.github/` (workflows, release scripts, CODEOWNERS, templates);
- redaction and query cleaning: `sdk/*/redaction.*`, `sdk/*/sanitize.*`, `sdk/conventions.json`;
- the SDK version pins and anything in `internal/setup/`, which writes to users' projects;
- new dependencies;
- security reports, which never go in public pull requests ([SECURITY.md](../../SECURITY.md)).

### Pull requests written with AI tools

Judge the change, not how it was written. The contributor is responsible for it either way ([guidelines §2](../ENGINEERING_GUIDELINES.md#2-using-ai-coding-tools)). Watch for the usual signs: a description claiming tests passed that did not run, unrelated "tidying", or code that looks right but was never executed. If the contributor says they could not run the checks, CI is the test; approve the run and read the results.

## 3. Write the review

- **Explain why.** "This should use `os.Root`, because it stops a path like `../../etc` escaping the project folder" teaches; "use os.Root" does not.
- **Say how much it matters.** Prefix small points with `nit:` and say they are optional. Make it clear which comments block the merge.
- **Suggest, don't rewrite.** GitHub's **suggestion** button (the ± icon in a comment) lets the contributor accept a fix with one click.
- **Ask, don't accuse.** "What happens if the file is empty?" works better than "This breaks on empty files."
- **Say what is good.** One line is enough: "Nice test, it covers exactly the case from the issue."
- **Thank first-time contributors** by name.

Then choose:

| Choice | When |
|---|---|
| **Approve** | Ready to merge, or only optional `nit:` points left |
| **Request changes** | Something must change before it can merge. Say exactly what |
| **Comment** | Questions, or you reviewed only part of it |

## 4. Saying no

Sometimes a pull request should not be merged: the feature does not fit, or it duplicates work. Close it the same week, with a reason and a thank-you. For example:

> Thank you for this, and for the clear description. I'm going to close it: OTelStarter leaves the choice of backend to the user, and this would make Datadog the default ([guidelines §24](../ENGINEERING_GUIDELINES.md#24-no-lock-in-to-a-backend)). If you'd like to keep going, #42 needs exactly the kind of parsing you did here.

A pull request with no reply from its author for 30 days after a review can be closed, with a note that it can be reopened any time.

## 5. Merge

Before you click merge:

1. All required checks are green. `main` does not allow merging otherwise.
2. The review comments are resolved.
3. **The title reads well.** It appears as-is in the release notes ("What's Changed"), so edit it if needed: `feat(python): support pymongo instrumentation`, not `update python.go`.
4. The `CHANGELOG.md` line is there, if users will notice the change.

Then:
- **Squash and merge** when the pull request has many small commits ("fix typo", "address review"). The squashed commit takes the pull request's title.
- **Create a merge commit** when the commits are already clean and tell a story.

GitHub then deletes the branch, and closes the issue if the description says `Closes #31`.

After a contributor's merge, leave a short thank-you, and say when it will ship: "This goes out in the next release." See [releasing.md](releasing.md).
