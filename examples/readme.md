# Example configuration files

in this directory we provide examples for all the dependent configuration files

## 🧷 Config Files

### 1. `.goreleaser.yml`

Used by **GoReleaser** to define build and release behavior.

#### Highlights:

- **Multi-platform Builds:** Targets `linux`, `darwin`, and `windows` for `amd64` and `arm64`.
- **No CGO:** Uses `CGO_ENABLED=0` for portability in CI/CD environments.
- **LDFLAGS Injection:** Injects version and commit at build time:
  ```go
  -X main.version={{.Version}} -X main.commit={{.Commit}}
  ```
- **Checksum & Signing:**
  - Generates SHA256 checksum files.
  - Signs checksum files using GPG.
- **Changelog Generation:**
  - Extracted from Git commits.
  - Uses regex-based grouping:
    - 🚀 Features (`feat:`)
    - 🐛 Bug Fixes (`fix:`)
    - 🛠 Maintenance (`chore:`)

> See full config in [.goreleaser.yml](.goreleaser.yml).

### 2. `.golangci.yml`

**optional** - Configuration for `golangci-lint`. If you don't specify this file `golanci-lint` will run with default settings.

#### Enabled Linters:

- `govet`, `errcheck`, `staticcheck`, `unused`
- `gofmt`, `goimports`, `gosimple`, `gocritic`

Configure to your preferences

> See full config in [.golangci.yml](.golangci.yml).

### 3. `gitversion.yml`

**Optional** - Used to define GitVersion's behavior for semantic versioning. Used in the example but not required to run the action.

> See full config in [gitversion.yml](gitversion.yml).