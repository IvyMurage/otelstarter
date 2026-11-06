# Reviewers

**For:** anyone who would like to help review, and the owner, who adds them.

OTelStarter is maintained by one person today. Pull requests wait less, and the project survives a busy month, when more people can review. This page explains what reviewers do, how to become one, and how the owner adds one.

## Roles

| Role | GitHub permission | Can | Leaves to others |
|---|---|---|---|
| **Contributor** | none | open issues and pull requests, comment, review informally | everything else |
| **Triager** | Triage | label, assign and close issues; request reviews; mark duplicates; review pull requests (the approval shows, but does not count toward merging) | approving CI runs, merging |
| **Reviewer** | Write | everything above, plus approve first-time CI runs, approve and merge pull requests | releases, workflows, settings, security reports |
| **Owner** | Admin | everything: releases, repository settings, workflows and secrets, security advisories | |

Reviewers follow [reviewing.md](reviewing.md). Releases stay with the owner ([releasing.md](releasing.md)), because a tag publishes to npm, PyPI and GitHub, and a published version cannot be taken back.

## Becoming a reviewer

There is no exam. What matters is that people can trust your judgement and your tone. Usually that looks like:

- two or three merged pull requests, so you know the codebase and the checks;
- a few helpful comments on other people's pull requests or issues. You can review without any permission: anyone can comment on a pull request;
- following the [code of conduct](../../CODE_OF_CONDUCT.md), and the spirit of [reviewing.md](reviewing.md): explain why, and be kind.

To ask, post in [Discussions](https://github.com/IvyMurage/otelstarter/discussions/categories/general) or message the owner. The owner may also invite you. Most people start as a **triager** and become a **reviewer** after a few weeks.

You can focus on one area: the Node.js SDK, the Python SDK, the CLI, or the docs. Nobody is expected to review everything.

Being a reviewer is voluntary. If you have less time, say so; stepping back is always fine, and you are welcome back later.

## Adding a reviewer (owner)

### Before the first person gets Write access

Today, anyone with Write access could push a release tag, and the release workflows would publish it. Close that gap once, before adding the first reviewer:

1. **Protect the release tags.** In **Settings → Rules → Rulesets → New ruleset → New tag ruleset**:
   - name: `Release tags`; enforcement: **Active**;
   - target tags, include by pattern: `v*`, `sdk-node-v*` and `sdk-python-v*`;
   - rules: **Restrict creations**, **Restrict updates** and **Restrict deletions**;
   - bypass list: **Repository admin**.

   Only the owner can then create, move or delete a release tag.
2. **Optionally,** in **Settings → Environments**, add yourself as a **required reviewer** of `npm` and `pypi`. Every SDK publish then waits for your click as well.

### Each new person

1. **Settings → Collaborators and teams → Add people.** Choose **Triage** or **Write**, as above. They receive an invitation by email.
2. **Give them their areas** in [`.github/CODEOWNERS`](../../.github/CODEOWNERS), so GitHub asks them to review pull requests there automatically. The last matching line wins:
   ```text
   *               @IvyMurage
   /sdk/node/      @IvyMurage @their-login
   /docs/          @IvyMurage @their-login
   # Owner only, whatever else matches:
   /.github/       @IvyMurage
   ```
3. **Welcome them** in Discussions, so contributors know who else may review their work.
4. **Update [CONTRIBUTING.md](../../CONTRIBUTING.md#7-what-happens-next-and-how-long-it-takes),** which says the project is maintained by one person and names who reviews.

### Once there are two or more reviewers

Raise the bar on `main`: in **Settings → Rules → Rulesets → Main branch rule → Require a pull request before merging**, set **Required approvals** to **1**, and turn on **Require review from Code Owners**. Every change then has a second pair of eyes, including the owner's.

Leave it at **0** while you are the only one with Write access: GitHub does not let anyone approve their own pull request, so 1 would block every merge.

### Removing access

In **Settings → Collaborators and teams**, change their role or remove them, and take their name out of `CODEOWNERS`. Do this when someone asks to step back, and review the list every few months: people who have stopped contributing should not keep Write access.
