#!/usr/bin/env bash
# Trace one finished archive, then upload the archive, its SBOM, and its
# provenance. upload_asset.sh signs each of those files.
set -euo pipefail

VERSION=${1:-}
ASSET=${2:-}
BINARY=${3:-}
if [[ -z "${VERSION}" || -z "${ASSET}" || -z "${BINARY}" ]]; then
  echo "publish_archive: usage: publish_archive.sh VERSION ARCHIVE BINARY" >&2
  exit 1
fi

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
bash "${SCRIPT_DIR}/trace_asset.sh" "${ASSET}" "${BINARY}" "${VERSION}"
bash "${SCRIPT_DIR}/upload_asset.sh" "${VERSION}" "${ASSET}"
bash "${SCRIPT_DIR}/upload_asset.sh" "${VERSION}" "${ASSET}.cdx.json"
bash "${SCRIPT_DIR}/upload_asset.sh" "${VERSION}" "${ASSET}.provenance.json"
