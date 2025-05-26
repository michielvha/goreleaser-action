# Go Build and Release Action

This GitHub Action builds and releases a Go application using GitVersion for versioning and GoReleaser for building and publishing releases.

## Features

- Go code linting with golangci-lint
- Binary signing with PGP
- Multi-platform binary building and releasing with GoReleaser

## Usage

> [!IMPORTANT]
> You must tag the repository with an initial tag (0.0.1) when using our gitversion file, otherwise the pipeline will fail.

Add this to your workflow file:

```yaml
jobs:
  build-and-release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Fetch all history to use GitVersion
          persist-credentials: true
      - name: Tag with GitVersion
        id: gitversion
        uses: michielvha/gitversion-tag-action@v4
        with:
          configFilePath: gitversion.yml
      - name: Build and Release
        uses: ./.github/actions/build-release-action
        with:
          github-token: ${{ secrets.EDGECTL_GITHUB_TOKEN }}
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
| `golangci-lint-config`   | Path to golangci-lint config file     | No       | `.golangci.yml`   |

## Requirements

### Config files

- Your project must have a valid Golang configuration file
- Your project must have a valid GoReleaser configuration file (.goreleaser.yml or .goreleaser.yaml)
- PGP key for signing releases must be set as a repository secret
- Optional: A `.golangci.yml` file for custom linting configuration (if not provided, golangci-lint will use its default settings)
  - If you specify a custom path for the golangci-lint config using the `golangci-lint-config` parameter (different from the default `.golangci.yml`) and the file doesn't exist, the action will fail. If you don't specify it, default behaviour is applied as fallback.

### 🧩 Secrets

| Name                      | Purpose                             |
| ------------------------- | ----------------------------------- |
| `PGP_PRIVATE_KEY`         | Used to sign the checksums          |
| `$(PROJECT)_GITHUB_TOKEN` | Custom token with extra permissions |

> [!IMPORTANT]
> You need to manually create a token  with `repo` and `workflow` permissions and generate a PGP Private Key, afterwards add these values to repo secrets


 <!-- |                     | `GPG_FINGERPRINT`                   | Exported at runtime for signing | -->

## Generate PGP Private Key

You can run the following on any unix host to quickly create a key

1. Create config file
```sh
cat > gen-key.conf <<EOF
%no-protection
Key-Type: RSA
Key-Length: 2048
Name-Real: Test User
Name-Email: test@example.com
Expire-Date: 0
%commit
EOF
```
2. Generate the key
```sh
gpg --batch --gen-key gen-key.conf
```
3. List the key fingerprint (optional)
```sh
gpg --list-secret-keys --keyid-format=long
```
4. Export the key and manually copy the value to repo secret.
```sh
gpg --armor --export-secret-keys test@example.com
```
5. Clean up
```sh
rm gen-key.conf
```

## 🧪 Action Overview

This custom GitHub Action(`binary-release.yaml`) automates the build and release process of a Go Binary. It is triggered by:

### 👣 Workflow Steps

1. **Setup Go**
   - Installs Go with version of choice, defaults to `1.24.3` using `actions/setup-go@v5`.

2. **Linting**
   - Runs `golangci-lint` (`v1.64`) to check for common Go issues and allows to set a custom config file path.

3. **Cache GoReleaser**
   - Caches GoReleaser binary to speed up subsequent runs.

4. **PGP Key Import & Fingerprint Extraction**
   - Imports the GPG private key from a GitHub secret.
   - Extracts the fingerprint and sets it as an env variable for signing.

5. **Release with GoReleaser**
   - Runs `goreleaser/goreleaser-action@v6` to:
     - Build binaries across multiple OS/architecture targets.
     - Package, checksum, sign, and publish the release.

## 📌 Future Improvements

- ✅ **Add Unit Tests** using `gotestsum` and coverage report.
- ✅ **Add CodeQL** for automated security analysis.