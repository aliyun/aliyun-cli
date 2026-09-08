#!/usr/bin/env bash
# Download every release asset, then write a fresh SHASUMS256.txt.
# Checksum is committed only after the full set is present and hashed.
# A non-zero status must stop upload_asset.sh and finish_release.sh.
set -euo pipefail

VERSION=${1:-}
if [[ -z "${VERSION}" ]]; then
  echo "download_assets: version is required" >&2
  exit 1
fi

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)
cd "${REPO_ROOT}"

exec go run ./tools/downloadassets/cmd/downloadassets -- "${VERSION}"
