#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf '::error::%s\n' "$*" >&2
  exit 1
}

case "${1:-}" in
  import)
    [[ -n "${PGP_PRIVATE_KEY:-}" ]] || fail 'A private signing key is required.'
    GNUPGHOME=$(mktemp -d "$RUNNER_TEMP/goreleaser-gpg.XXXXXX")
    export GNUPGHOME
    printf 'gnupghome=%s\n' "$GNUPGHOME" >> "$GITHUB_OUTPUT"
    printf '%s\n' "$PGP_PRIVATE_KEY" | gpg --batch --pinentry-mode loopback --import
    FINGERPRINT=$(gpg --batch --with-colons --list-secret-keys |
      awk -F: '$1 == "sec" { primary = 1; next } primary && $1 == "fpr" { print $10; primary = 0 }')
    [[ "$FINGERPRINT" =~ ^([0-9A-F]{40}|[0-9A-F]{64})$ ]] ||
      fail 'Supply exactly one private primary key; signing subkeys are supported.'
    printf 'fingerprint=%s\n' "$FINGERPRINT" >> "$GITHUB_OUTPUT"
    ;;
  cleanup)
    [[ "${GNUPGHOME:-}" == "$RUNNER_TEMP"/goreleaser-gpg.* ]] ||
      fail 'Refusing to remove a keyring outside this action.'
    NAME=${GNUPGHOME#"$RUNNER_TEMP"/}
    [[ "$NAME" =~ ^goreleaser-gpg\.[[:alnum:]]+$ ]] ||
      fail 'Refusing to remove an unexpected keyring path.'
    trap 'rm -r -- "$GNUPGHOME"' EXIT
    SOCKET=$(gpgconf --list-dirs agent-socket)
    if [[ -S "$SOCKET" ]]; then
      AGENT_PID=$(gpg-connect-agent --no-autostart 'GETINFO pid' /bye |
        awk '$1 == "D" { print $2 }')
      [[ "$AGENT_PID" =~ ^[0-9]+$ && "$AGENT_PID" -gt 1 ]] ||
        fail 'Could not identify this keyring agent for cleanup.'
      kill "$AGENT_PID"
    fi
    ;;
  *)
    fail 'Expected import or cleanup.'
    ;;
esac
