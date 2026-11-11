# Releasing

A release is started by pushing a `vX.Y.Z` tag. This triggers `cd` workflow that builds the binaries, creates the GH release, updates brew formula at `snowplow/homebrew-taps` and pushes the npm packages

## Steps

1. Tag `main` and push the tag:

   ```bash
   git checkout main && git pull
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

2. Watch the `cd` run, then check the [GitHub release](https://github.com/snowplow/snowplow-cli/releases), `brew info snowplow/taps/snowplow-cli` and `npm view @snowplow/snowplow-cli version`.

The GitHub release notes are generated from the commits since the previous tag, without the ones starting with `docs:` or `test:`.

## Potential issues

**`preflight` fails.** Nothing is released yet. Fix the token, then re-run the failed job.

- `HOMEBREW_TAPS_GITHUB_TOKEN`: a PAT with write access to `snowplow/homebrew-taps`.
- `NPM_PUBLISHING_AUTOMATION_TOKEN`: an npm token that can publish to `@snowplow`. If `npm whoami` fails, create a new one and update the secret.

**npm publish fails halfway.** npm never allows publishing the same version twice, so a re-run fails on the packages that are already published. Release the next patch version instead.

**The release failed because of a bug in `cd.yaml` or `.goreleaser.yaml`.** GitHub runs the workflow as it was in the tagged commit. So after the fix is merged to `main`, re-running the failed run still uses the old, broken files. The tag has to point to a commit that has the fix. Either release the next patch version, or, if nothing was published to npm yet, delete the GitHub release if it was created, then delete the tag and create it again on the current `main`:

```bash
git checkout main && git pull
git push origin --delete vX.Y.Z
git tag -d vX.Y.Z
git tag vX.Y.Z
git push origin vX.Y.Z
```

Moving a tag can cause checksum errors for anyone who already fetched it with `go install`, so prefer a new version when the tag has been public for a while.
