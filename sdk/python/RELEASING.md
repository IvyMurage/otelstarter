# Releasing the Python SDK

The `Release Python SDK` workflow (`.github/workflows/release-python-sdk.yml`) publishes `sdk/python` to PyPI when a tag like `sdk-python-v0.1.0` is pushed. It uses PyPI **Trusted Publishing**: PyPI trusts the workflow's GitHub identity, so no API token is stored anywhere.

## One-time setup

**On PyPI**, signed in as the maintainer:

1. Open your account menu, then **Publishing**.
2. Add a new **pending publisher** for **GitHub Actions** with:

   | Field | Value |
   |-------|-------|
   | PyPI project name | `otelstarter` |
   | Repository | `IvyMurage/otelstarter` |
   | Workflow file name | `release-python-sdk.yml` |
   | Environment name | `pypi` |

A pending publisher does not reserve the name. If someone else registers `otelstarter` before the first release, the pending publisher stops working, so publish soon after setting it up.

**On GitHub**, in the repository's **Settings → Environments**:

1. Create an environment named `pypi`.
2. Optionally add yourself as a **required reviewer**. Every release then waits for you to click *Approve* before anything is uploaded.

## Each release

1. Set the new version in `src/otelstarter/__init__.py`. It is the only place the version lives.
2. Add the release to `CHANGELOG.md`.
3. Merge to `main`, then tag that commit and push the tag:

   ```bash
   git tag sdk-python-v0.1.0
   git push origin sdk-python-v0.1.0
   ```

The workflow then:

1. checks that the tag matches the version, and stops if it doesn't
2. runs the tests
3. builds and checks the package
4. publishes it

A version cannot be uploaded to PyPI twice, so a mistake needs a new version number.
