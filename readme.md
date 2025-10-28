<!-- TODO: Properly test statements like which config files are required and which will default, I think golanci is needed and gorelease isn't 
test the whole setup better so we can give better explenations in this readme when it comes to the token required, optionally pushing to brew tap etc etc.

Create a second version that also uses goreleaser for deploying the container to ghcr, get inspiration here :

project_name: crossplane-docs

builds:
  - id: crossplane-docs
    binary: crossplane-docs
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64

dockers:
  - image_templates: ["ghcr.io/kavinraja-g/{{ .ProjectName }}:{{ .Version }}-amd64"]
    dockerfile: Dockerfile
    use: buildx
    build_flag_templates:
      - --platform=linux/amd64
      - --label=org.opencontainers.image.title={{ .ProjectName }}
      - --label=org.opencontainers.image.description={{ .ProjectName }}
      - --label=org.opencontainers.image.url=https://github.com/kavinraja-g/{{ .ProjectName }}
      - --label=org.opencontainers.image.source=https://github.com/kavinraja-g/{{ .ProjectName }}
      - --label=org.opencontainers.image.version={{ .Version }}
      - --label=org.opencontainers.image.revision={{ .FullCommit }}
      - --label=org.opencontainers.image.licenses=Apache
  - image_templates: ["ghcr.io/kavinraja-g/{{ .ProjectName }}:{{ .Version }}-arm64v8"]
    goarch: arm64
    dockerfile: Dockerfile
    use: buildx
    build_flag_templates:
      - --platform=linux/arm64/v8
      - --label=org.opencontainers.image.title={{ .ProjectName }}
      - --label=org.opencontainers.image.description={{ .ProjectName }}
      - --label=org.opencontainers.image.url=https://github.com/kavinraja-g/{{ .ProjectName }}
      - --label=org.opencontainers.image.source=https://github.com/kavinraja-g/{{ .ProjectName }}
      - --label=org.opencontainers.image.version={{ .Version }}
      - --label=org.opencontainers.image.revision={{ .FullCommit }}
      - --label=org.opencontainers.image.licenses=Apache

docker_manifests:
  - name_template: ghcr.io/kavinraja-g/{{ .ProjectName }}:{{ .Version }}
    image_templates:
      - ghcr.io/kavinraja-g/{{ .ProjectName }}:{{ .Version }}-amd64
      - ghcr.io/kavinraja-g/{{ .ProjectName }}:{{ .Version }}-arm64v8
  - name_template: ghcr.io/kavinraja-g/{{ .ProjectName }}:latest
    image_templates:
      - ghcr.io/kavinraja-g/{{ .ProjectName }}:{{ .Version }}-amd64
      - ghcr.io/kavinraja-g/{{ .ProjectName }}:{{ .Version }}-arm64v8

archives:
  - builds:
      - crossplane-docs
    name_template: "{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"
    wrap_in_directory: false
    format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    files:
      - LICENSE
      - README.md

brews:
  - name: crossplane-docs
    homepage: "https://github.com/Kavinraja-G/crossplane-docs/"
    description: "XDocs generator for Crossplane"
    repository:
      owner: Kavinraja-G
      name: homebrew-tap
      token: "{{ .Env.TAP_GITHUB_TOKEN }}"

-->

# Go Build and Release Action

This GitHub Action builds and releases a Go application using GitVersion for versioning and GoReleaser for building and publishing releases.

## Features

- Go code linting with golangci-lint
- Binary signing with PGP
- Multi-platform binary building and releasing with GoReleaser

## Usage

> [!IMPORTANT]
> You must tag the repository with an initial tag (v0.0.0) when using our gitversion file, otherwise the pipeline will fail.

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
          github-token: ${{ github.token }}
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
| `golangci-lint-config`   | Path to golangci-lint config file (passed as --config argument)     | No       | `.golangci.yml`   |

## Requirements

### Config files

- Your project must have a valid Golang configuration file
- PGP key for signing releases must be set as a repository secret
- Optional: a valid GoReleaser configuration file (.goreleaser.yml or .goreleaser.yaml) if omitted, gorelease will run using it's default configuration.
- Optional: A `.golangci.yml` file for custom linting configuration (if not provided, golangci-lint will use its default settings)
### 🧩 Secrets

| Name                      | Purpose                             |
| ------------------------- | ----------------------------------- |
| `PGP_PRIVATE_KEY`         | Used to sign the checksums          |
| `${{ github.token }}` or `$(PROJECT)_GITHUB_TOKEN` | Default for normal release with `contents: write` or Custom token with extra permissions provided via secrets if needed|

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
