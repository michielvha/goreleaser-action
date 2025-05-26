# Go Build and Release Action

This GitHub Action builds and releases a Go application using GitVersion for versioning and GoReleaser for building and publishing releases.

## Features

- Automatic versioning with GitVersion
- Go code linting with golangci-lint
- Binary signing with PGP
- Multi-platform binary building and releasing with GoReleaser

## Usage

Add this to your workflow file:

```yaml
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Fetch all history to use GitVersion
          persist-credentials: true
      
      - name: Build and Release
        uses: ./.github/actions/build-release-action
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          pgp-private-key: ${{ secrets.PGP_PRIVATE_KEY }}
          # Optional overrides:
          # go-version: '1.24.3'
          # gitversion-config-path: 'gitversion.yml'
          # goreleaser-version: '~> v2'
          # goreleaser-args: 'release --clean'
```

## Inputs

| Name                     | Description                           | Required | Default           |
| ------------------------ | ------------------------------------- | -------- | ----------------- |
| `github-token`           | GitHub token for release              | Yes      | N/A               |
| `pgp-private-key`        | PGP private key for signing           | Yes      | N/A               |
| `go-version`             | The Go version to use                 | No       | `1.23.3`          |
| `gitversion-config-path` | Path to GitVersion configuration file | No       | `gitversion.yml`  |
| `goreleaser-version`     | GoReleaser version to use             | No       | `~> v2`           |
| `goreleaser-args`        | Arguments to pass to GoReleaser       | No       | `release --clean` |

## Requirements

- Your project must have a valid GitVersion configuration file
- Your project must have a valid GoReleaser configuration file (.goreleaser.yml or .goreleaser.yaml)
- PGP key for signing releases must be set as a repository secret
