# Releasing the Node.js SDK

The `Release Node.js SDK` workflow (`.github/workflows/release-node-sdk.yml`) publishes `sdk/node` to npm when a tag like `sdk-node-v0.2.0` is pushed. It uses npm **Trusted Publishing**: npm trusts the workflow's GitHub identity, so no npm token is stored anywhere.

## One-time setup

npm only allows trusted publishing for a package that already exists, so the **first** version is published by hand. This differs from PyPI, which allowed setting it up first.

**1. Publish version 0.1.0 from your machine.** You need an npmjs.com account with two-factor authentication.

```bash
cd sdk/node
npm login                    # opens the browser
npm ci && npm test           # builds dist/ and runs the tests
npm publish --access public  # publishes otelstarter 0.1.0
```

If npm answers `E403 ... Two-factor authentication ... is required to publish packages`, turn on two-factor authentication in your npmjs.com account settings, then `npm logout && npm login` and publish again. Do not use a token that bypasses two-factor authentication instead.

npm may first show a placeholder version `0.0.0-stage` and say the package "is being processed". The real version appears a few minutes later. Check with `npm view otelstarter versions`, or at https://www.npmjs.com/package/otelstarter.

**2. Link npm to the repository.** On npmjs.com, open the package, then **Settings → Trusted publishing → GitHub Actions**, and enter:

| Field | Value |
|-------|-------|
| Organization or user | `IvyMurage` |
| Repository | `otelstarter` |
| Workflow filename | `release-node-sdk.yml` |
| Environment name | `npm` |

npm then shows the status **Pending validation**. The link counts as validated after the workflow's first publish, which must happen within 2 days of saving it, or the setting has to be added again. A new version is needed for that, since npm never accepts the same version twice.

**3. On GitHub,** in **Settings → Environments**, create an environment named `npm`. Optionally add yourself as a required reviewer, so each release waits for your approval.

**4. Optionally, on npmjs.com,** set **Publishing access** to "Require two-factor authentication and disallow tokens", so only the workflow can publish.

## Each release

1. Set the new version in `package.json`, the package's only version declaration.
2. Add the release to `CHANGELOG.md`.
3. Merge to `main`, then tag that commit and push the tag:

   ```bash
   git tag sdk-node-v0.2.0
   git push origin sdk-node-v0.2.0
   ```

The workflow then:

1. checks that the tag matches the version, and stops if it doesn't
2. checks the npm version
3. runs the tests and the license check
4. publishes

npm adds a provenance statement automatically when the repository is public. A version cannot be published twice; a mistake needs a new version number.
