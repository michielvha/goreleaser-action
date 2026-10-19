# Go Release action

Build, sign, and publish Go artifacts with GoReleaser. This branch prepares the
release-only next major. Existing `main` and released consumer refs are not
changed by this branch.

The action provisions Go, caches the GoReleaser binary, imports the signing key
with native GnuPG, and runs GoReleaser. It does not lint, test, decide
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

## Binary caching

Exact stable versions use a cache key containing the operating system,
architecture, and requested version. The existing upstream GoReleaser action
installs and verifies the binary only on a cache miss. This action then runs that
binary directly. Version ranges and prerelease channels remain supported, but
install fresh instead of caching a moving version.

The binary is saved before key import or release execution, so a later failure
does not discard the installation. The `cache-hit` output is `true` when the
binary was restored. Go's module and build cache remains managed by setup-go.
Renovate tracks both tool-version defaults.

## Migration from v1

This is an opt-in major change, not an update to the existing v1 contract.

- Move formatting, module checks, vet, tests, build checks, and lint to
  `michielvha/go-ci-action`.
- Remove the `golangci-lint-config` input from the release action.
- Remove the unused `gitversion-config-path` input. Configure the separate
  GitVersion action instead.
- Advisory lint moves to the CI action.
- Binary caching uses the actual installed executable instead of the old unused
  `~/.goreleaser` directory.
- GPG import remains in our action, uses an isolated keyring, and reports exactly
  one primary fingerprint. Empty, public-only, and ambiguous multi-key inputs fail.

GoReleaser owns draft creation, artifact upload, and final publication. Configure
prereleases and replacement policy in its configuration rather than adding
another release engine in the workflow. Keep existing-asset replacement disabled
and inspect partial releases before retrying. Never move the source tag.

## Tests

Run `bun test` and `shellcheck scripts/*.sh`.

- Signing tests generate disposable keys, import signing subkeys, sign and verify
  checksums, detect tampering, reject invalid inputs, and check keyring isolation
  and cleanup.
- Execution tests cover cold/warm binary staging, damaged caches, quoted
  arguments, command failure, and invoking the CLI once without evaluating shell
  syntax. They do not freeze the action's step order or input list.
- GitHub CI installs the real pinned GoReleaser, checks failure on an invalid
  signing key, then restores the cache and builds a signed snapshot. It verifies
  the checksum signature with the public key and checks private-key cleanup.

Snapshot publication is disabled. These tests do not publish GitHub Releases or
prove that production publication credentials are configured.
