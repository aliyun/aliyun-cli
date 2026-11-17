#!/usr/bin/env bash
# Sign one release asset with the publisher key. The secret never touches the repo.
# CLI_RELEASE_SIGNING_KEY is the GitHub Actions secret (header + base64 private key).
set -euo pipefail

ASSET=${1:-}
if [[ -z "${ASSET}" || ! -f "${ASSET}" ]]; then
  echo "sign_asset: asset file is required" >&2
  exit 1
fi
if [[ -z "${CLI_RELEASE_SIGNING_KEY:-}" ]]; then
  echo "sign_asset: CLI_RELEASE_SIGNING_KEY is required; refusing to publish an unsigned asset" >&2
  exit 1
fi

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)
tmp=$(mktemp)
chmod 600 "${tmp}"
trap 'rm -f "${tmp}"' EXIT
printf '%s\n' "${CLI_RELEASE_SIGNING_KEY}" > "${tmp}"

(
  cd "${REPO_ROOT}"
  go run ./tools/publishsign/cmd/publishsign sign --secret-key-file "${tmp}" --asset "${ASSET}"
)
