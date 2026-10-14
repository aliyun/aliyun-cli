#!/usr/bin/env bash
set -euo pipefail

TAG=v$1
ASSET=$2
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

upload_one() {
  local file=$1
  local base
  base=$(basename "${file}")
  local type
  case "${base}" in
    *.tgz) type=application/x-compressed-tar ;;
    *.txt|*.minisig) type=text/plain ;;
    *) type=application/zip ;;
  esac

  local release_id
  release_id=$(curl -fsSL \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $GITHUB_TOKEN" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    https://api.github.com/repos/aliyun/aliyun-cli/releases/tags/"$TAG" | jq '.["id"]')

  if [ -z "${release_id}" ] || [ "${release_id}" = "null" ]; then
    echo "Failed to get release ID for tag $TAG"
    exit 1
  fi

  local asset_id
  asset_id=$(curl -fsSL \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $GITHUB_TOKEN" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    https://api.github.com/repos/aliyun/aliyun-cli/releases/"${release_id}"/assets | jq -r ".[] | select(.name == \"${base}\") | .id")

  if [ -n "${asset_id}" ]; then
    echo "Asset already exists. Deleting asset ID ${asset_id}"
    curl -fsSL \
      -X DELETE \
      -H "Accept: application/vnd.github+json" \
      -H "Authorization: Bearer $GITHUB_TOKEN" \
      -H "X-GitHub-Api-Version: 2022-11-28" \
      https://api.github.com/repos/aliyun/aliyun-cli/releases/assets/"${asset_id}"
  fi

  printf "Uploading %s to release %s\n" "${file}" "${release_id}"
  curl -fsSL \
    -X PUT \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $GITHUB_TOKEN" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    -H "Content-Type: ${type}" \
    "https://uploads.github.com/repos/aliyun/aliyun-cli/releases/${release_id}/assets?name=${base}" \
    --data-binary "@${file}"
}

case "$(basename "${ASSET}")" in
  *.tgz|*.zip|*.pkg)
    bash "${SCRIPT_DIR}/sign_asset.sh" "${ASSET}"
    upload_one "${ASSET}"
    upload_one "${ASSET}.minisig"
    ;;
  *)
    upload_one "${ASSET}"
    ;;
esac

