#!/usr/bin/env bash

set -euo pipefail

TAGNAME=${1:-}
if [[ -z "${TAGNAME}" ]]; then
  echo "refuse release: tag is required" >&2
  exit 1
fi

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)

# Reject illegal versions before creating a GitHub release. Legal prerelease
# tags are still created as prerelease; finish_release decides latest promotion.
if ! (cd "${REPO_ROOT}" && go run ./tools/releasever/cmd/releasever classify -- "${TAGNAME}") >/dev/null; then
  echo "refuse release: invalid version: ${TAGNAME}" >&2
  exit 1
fi

DATA='{"tag_name":"'$TAGNAME'","name":"'$TAGNAME'","draft":false,"prerelease":true,"generate_release_notes":true}'

curl -fsSL \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/repos/aliyun/aliyun-cli/releases \
  -d "$DATA"
