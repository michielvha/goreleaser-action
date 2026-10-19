#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf '::error::%s\n' "$*" >&2
  exit 1
}

case "${1:-}" in
  prepare)
    if [[ "${REQUESTED_VERSION:?}" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      printf 'cacheable=true\n' >> "$GITHUB_OUTPUT"
    else
      printf 'cacheable=false\n' >> "$GITHUB_OUTPUT"
      printf '::notice::Version ranges and prerelease channels are installed fresh; use an exact stable version for binary caching.\n'
    fi
    ;;
  stage)
    if [[ "${CACHE_HIT:-}" == true ]]; then
      [[ -x "$CACHE_DIRECTORY/goreleaser" ]] ||
        fail 'The restored GoReleaser cache has no executable.'
      printf 'cache-hit=true\n' >> "$GITHUB_OUTPUT"
    else
      BINARY=$(command -v goreleaser) || fail 'The GoReleaser installer supplied no executable.'
      mkdir -p "$CACHE_DIRECTORY"
      install -m 0755 "$BINARY" "$CACHE_DIRECTORY/goreleaser"
      printf 'cache-hit=false\n' >> "$GITHUB_OUTPUT"
    fi
    ;;
  run)
    [[ -x "$CACHE_DIRECTORY/goreleaser" ]] || fail 'GoReleaser is not installed.'
    [[ "${GORELEASER_ARGS:-}" =~ [^[:space:]] ]] || fail 'GoReleaser arguments are required.'
    ARGUMENT_FILE=$(mktemp "$RUNNER_TEMP/goreleaser-args.XXXXXX")
    trap 'rm -- "$ARGUMENT_FILE"' EXIT
    printf '%s\n' "$GORELEASER_ARGS" | xargs printf '%s\0' > "$ARGUMENT_FILE"
    ARGS=()
    while IFS= read -r -d '' ARGUMENT; do
      ARGS+=("$ARGUMENT")
    done < "$ARGUMENT_FILE"
    [[ "${#ARGS[@]}" -gt 0 && -n "${ARGS[0]}" ]] || fail 'A GoReleaser command is required.'
    "$CACHE_DIRECTORY/goreleaser" "${ARGS[@]}"
    ;;
  *)
    fail 'Expected prepare, stage, or run.'
    ;;
esac
