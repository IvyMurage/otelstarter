# Releasing

**For:** the owner. Releases are not delegated to reviewers: a tag publishes to GitHub, npm or PyPI, and a published version can never be taken back.

OTelStarter has three things to release, each with its own tag:

| What | Tag | Where the version lives | Publishes to |
|---|---|---|---|
| The CLI | `v0.2.0` | nowhere: the tag is the version | GitHub Releases, and Go's module proxy |
| The Node.js SDK | `sdk-node-v0.1.5` | `sdk/node/package.json` | npm, `otelstarter` |
| The Python SDK | `sdk-python-v0.3.2` | `sdk/python/src/otelstarter/__init__.py` | PyPI, `otelstarter` |

Each one has a workflow in `.github/workflows/` that runs when its tag is pushed.

## Rules that apply to every release

- **Never move or delete a pushed tag, and never reuse a version.** npm and PyPI refuse a version twice, and Go's module proxy keeps every CLI tag forever. If a release is wrong, release the next version.
- **Tag the commit you mean.** Pull `main` first, and check `git log --oneline -1` before tagging. A tag on an old local `main` is the most common mistake.
- **Version numbers.** While we are on `0.x`: the last number for fixes (`0.2.0` → `0.2.1`), the middle one for new features or anything that changes behaviour (`0.2.1` → `0.3.0`).

## When to release

- **A security fix:** straight away, and SDK before CLI (see below).
- **Otherwise:** when `[Unreleased]` in `CHANGELOG.md` holds something users would want. There is no fixed schedule.

## Releasing an SDK

An SDK is released first when the CLI should install the new version, because `init` pins an exact SDK version.

1. On a branch, set the new version:
   - **Node.js:** `"version"` in `sdk/node/package.json`, and `SDKVersion` in `internal/adapters/node/setup.go`.
   - **Python:** `__version__` in `sdk/python/src/otelstarter/__init__.py`, and `otelstarterVersion` in `internal/adapters/python/setup.go`.

   A Go test fails if the SDK's version and the pin differ, so you cannot forget one.
2. Add a line to `CHANGELOG.md` under `[Unreleased]`, for example "Node.js SDK 0.1.6: …".
3. Open the pull request, wait for green, and merge.
4. Tag and push:
   ```bash
   git switch main && git pull
   git log --oneline -1                        # the merge you just made?
   git tag sdk-node-v0.1.6 && git push origin sdk-node-v0.1.6
   ```
5. Watch the run on the [Actions](https://github.com/IvyMurage/otelstarter/actions) tab. It checks the tag matches the version, runs the tests, and publishes.
6. Check it is live: `npm view otelstarter version`, or https://pypi.org/project/otelstarter/.

The one-time setup (trusted publishing, the `npm` and `pypi` environments) is in [sdk/node/RELEASING.md](../../sdk/node/RELEASING.md) and [sdk/python/RELEASING.md](../../sdk/python/RELEASING.md).

## Releasing the CLI

1. On a branch `release/cli-v0.3.0`, edit `CHANGELOG.md`:
   - rename `## [Unreleased]` to `## [0.3.0] - 2026-12-01`;
   - add a new, empty `## [Unreleased]` above it;
   - under the new heading, write one or two sentences saying what the release is about. They become the first lines of the release page.

   Also update the docs that name the current CLI version (`grep -rn "v0\.2\.0" docs README.md`).
2. Open the pull request, wait for green, and merge.
3. Tag and push:
   ```bash
   git switch main && git pull
   git log --oneline -1                        # the release merge?
   git tag v0.3.0 && git push origin v0.3.0
   ```
4. The **Release CLI** workflow (about 3 minutes):
   1. copies the `## [0.3.0]` section of `CHANGELOG.md` into the release notes, and stops before anything is built if there is no such section;
   2. runs the tests;
   3. builds the CLI for Linux, macOS and Windows, on amd64 and arm64;
   4. publishes the release with the six archives and `checksums.txt`. GitHub adds the list of merged pull requests and new contributors under your notes.

A tag with a suffix, such as `v0.3.0-rc.1`, becomes a pre-release and does not need a changelog section.

To try the build without releasing: `.github/scripts/build-cli.sh v0.0.0-test dist`.

## Check the release

Do this every time, as a new user would:

```bash
cd "$(mktemp -d)"
curl -fsSLO https://github.com/IvyMurage/otelstarter/releases/latest/download/otelstarter_linux_amd64.tar.gz
curl -fsSLO https://github.com/IvyMurage/otelstarter/releases/latest/download/checksums.txt
sha256sum --ignore-missing -c checksums.txt     # otelstarter_linux_amd64.tar.gz: OK
tar -xzf otelstarter_linux_amd64.tar.gz && ./otelstarter version   # the new version
```

Then:
- the [release page](https://github.com/IvyMurage/otelstarter/releases/latest) is marked **Latest**, and its notes read well;
- `./otelstarter init --dry-run` on a small Node.js or Python app pins the newest SDK versions.

## If something goes wrong

| Problem | What to do |
|---|---|
| The workflow says the tag does not match the version | The tag was pushed on the wrong commit, or the version was not bumped. Nothing was published. Fix the version in a pull request, merge, and tag the **next** version |
| The CLI workflow says there is no changelog section | Nothing was published. Rename `[Unreleased]` to the **next** version, merge, and tag that |
| A published release has a bug | Fix it and release the next version. For a broken CLI release, you may also mark the GitHub release as a pre-release so `latest` points to the previous one. Never delete the tag |
| A published SDK has a security problem | Release the fix first. Then deprecate the bad version: `npm deprecate otelstarter@0.1.4 "Security fix in 0.1.5"`, or **yank** it on PyPI. Say in the changelog how users update |

## After a security release

`init` does not change a version pin that is already in a project. So the changelog must tell users who set up with an older version how to update by hand, as v0.2.0 did:

> - Node.js: `npm install --save-exact otelstarter@0.1.5`
> - Python: change the pin to `otelstarter==0.3.2` and reinstall.
