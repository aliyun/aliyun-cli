#!/usr/bin/env bash
# Write NAME.cdx.json and NAME.provenance.json for one finished archive.
# The SBOM comes from the binary's own Go build info. The provenance binds
# the archive digest to the tag, commit, and aliyun-openapi-meta gitlink.
set -euo pipefail

ASSET=${1:-}
BINARY=${2:-}
VERSION=${3:-}
if [[ -z "${ASSET}" || -z "${BINARY}" || -z "${VERSION}" ]]; then
  echo "trace_asset: usage: trace_asset.sh ARCHIVE BINARY VERSION" >&2
  exit 1
fi
if [[ -z "${GITHUB_REF_NAME:-}" || -z "${GITHUB_SHA:-}" || -z "${GITHUB_RUN_ID:-}" || -z "${GITHUB_REPOSITORY:-}" ]]; then
  echo "trace_asset: GITHUB_REF_NAME, GITHUB_SHA, GITHUB_RUN_ID, and GITHUB_REPOSITORY are required" >&2
  exit 1
fi

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)
cd "${REPO_ROOT}"

gitlink=$(git rev-parse "HEAD:aliyun-openapi-meta")
server=${GITHUB_SERVER_URL:-https://github.com}
run_url="${server}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}"

go run ./tools/buildtrace/cmd/buildtrace \
  --binary "${BINARY}" \
  --asset "${ASSET}" \
  --version "${VERSION}" \
  --tag "${GITHUB_REF_NAME}" \
  --commit "${GITHUB_SHA}" \
  --gitlink "${gitlink}" \
  --run-url "${run_url}"
