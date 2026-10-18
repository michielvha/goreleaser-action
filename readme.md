# Go Release action

Build, sign, and publish Go artifacts with GoReleaser. This branch prepares the
release-only next major. Existing `main` and released consumer refs are not
changed by this branch.

The action provisions Go, imports the signing key through a dedicated GPG
action, and runs the upstream GoReleaser action. It does not lint, test, decide
versions, or create source tags. Use `michielvha/go-ci-action` for CI and call
`michielvha/gitversion-tag-action` directly for versioning.

```yaml
steps:
  - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
    with:
      fetch-depth: 0
      persist-credentials: false
  - uses: michielvha/goreleaser-action@FULL_COMMIT_SHA
    with:
      go-version: '1.27.2'
      github-token: ${{ github.token }}
      pgp-private-key: ${{ secrets.PGP_PRIVATE_KEY }}
```

Replace the placeholder with the verified next-major commit. The job requires
`contents: write`. The caller checks out the exact tag's source and runs its CI
checks before this action.

## Inputs

| Input | Required | Default |
| --- | --- | --- |
| `go-version` | No | `1.25.11` |
| `github-token` | Yes | None |
| `pgp-private-key` | Yes | None |
| `goreleaser-version` | No | `~> v2` |
| `goreleaser-args` | No | `release --clean` |

The imported key's fingerprint is supplied as `GPG_FINGERPRINT`. Configure
checksum signing in the caller's GoReleaser configuration.

## Migration from v1

This is an opt-in major change, not an update to the existing v1 contract.

- Move formatting, module checks, vet, tests, build checks, and lint to
  `michielvha/go-ci-action`.
- Remove the `golangci-lint-config` input from the release action.
- Remove the unused `gitversion-config-path` input. Configure the separate
  GitVersion action instead.
- The ineffective private GoReleaser cache and advisory lint step are removed.
- GPG import uses the dedicated import action and its fingerprint output.

GoReleaser owns draft creation, artifact upload, and final publication. Configure
prereleases and replacement policy in its configuration rather than adding
another release engine in the workflow. Keep existing-asset replacement disabled
and inspect partial releases before retrying. Never move the source tag.

Run `bun test` for the action contract checks. Signing and publication need a
separately authorized integration run; the tests do not publish releases.
