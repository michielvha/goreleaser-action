<!-- TODO: Verify config defaults/token requirements and revisit Homebrew/GHCR publication: https://github.com/michielvha/goreleaser-action/issues/12 -->

# Go Release action

Build, sign, and publish Go artifacts with GoReleaser. This branch prepares the
release-only next major. Existing `main` and released consumer refs are not
changed by this branch.

The action provisions Go, imports the signing key with native GnuPG, and calls
the upstream GoReleaser action to install and run GoReleaser. It does not lint, test, decide
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
| `go-version` | No | See [action.yml](action.yml) |
| `github-token` | Yes | None |
| `pgp-private-key` | Yes | None |
| `goreleaser-version` | No | Pinned stable version in [action.yml](action.yml) |
| `goreleaser-args` | No | `release --clean` |

The imported key's primary fingerprint is supplied as `GPG_FINGERPRINT`; signing
subkeys are supported. Import uses a private temporary keyring, not the caller's
keyring. The action removes that keyring even when GoReleaser fails. Configure
checksum signing in the caller's GoReleaser configuration. No separate GPG action
is required.

Tag-triggered runs pass the event's exact tag to GoReleaser, even when another
tag points at the same commit. Non-tag runs preserve an explicitly supplied
`GORELEASER_CURRENT_TAG`, or leave tag discovery to GoReleaser.

## Go caching

Setup-go manages Go's module and build caches. It does not run CI checks or cache
the GoReleaser executable. The upstream GoReleaser action downloads the requested
tool normally. There is no custom binary cache, staging, or CLI argument parser.
Renovate tracks both tool-version defaults.

## Migration from v1

This is an opt-in major change, not an update to the existing v1 contract.

- Move formatting, module checks, vet, tests, build checks, and lint to
  `michielvha/go-ci-action`.
- Remove the `golangci-lint-config` input from the release action.
- Remove the unused `gitversion-config-path` input. Configure the separate
  GitVersion action instead.
- Advisory lint moves to the CI action.
- The old unused `~/.goreleaser` binary cache is removed. Go module/build caching
  remains enabled through setup-go.
- GPG import remains in our action, uses an isolated keyring, and reports exactly
  one primary fingerprint. Empty, public-only, and ambiguous multi-key inputs fail.

GoReleaser owns draft creation, artifact upload, and final publication. Configure
prereleases and replacement policy in its configuration rather than adding
another release engine in the workflow. Keep existing-asset replacement disabled
and inspect partial releases before retrying. Never move the source tag.

## Tests

Run `go test -count=1 ./...` and `shellcheck scripts/*.sh`. Fresh test execution is
required because Go's result cache does not track changes to Bash subprocesses.
Go module/compiler caching stays enabled. The signing tests and ephemeral-key
generator use the standard library and GnuPG; targeted manifest contracts use
`go.yaml.in/yaml/v3`. Bun is not required.

- Signing tests generate disposable keys, import signing subkeys, sign and verify
  checksums, detect tampering, reject invalid inputs, and check keyring isolation
  and cleanup.
- Installation and argument handling belong to the upstream GoReleaser action,
  not a duplicate implementation in this action.
- Regression contracts protect the event-tag binding and fresh behavior-test
  execution, without freezing unrelated inputs or step order.
- GitHub CI checks failure on an invalid signing key, then calls the upstream
  action to build a signed snapshot with the real pinned GoReleaser. It verifies
  the selected tag when stable and prerelease tags share one commit, the checksum
  signature with the public key, and private-key cleanup.

Snapshot publication is disabled. These tests do not publish GitHub Releases or
prove that production publication credentials are configured.
