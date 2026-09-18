#!/usr/bin/env bash

set -euo pipefail

VERSION=${1:-}
if [[ -z "${VERSION}" ]]; then
  echo "refuse release: version is required" >&2
  exit 1
fi

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)
cd "${REPO_ROOT}"

# One semver policy for every tag: legal prerelease stays off latest;
# illegal versions fail before any publish side effect.
if ! PLAN=$(go run ./tools/releasever/cmd/releasever plan -- "${VERSION}"); then
  echo "refuse release: invalid version: ${VERSION}" >&2
  exit 1
fi

release_kind=""
update_latest=""
write_stable_version=""
mark_official=""
while IFS='=' read -r key value; do
  case "${key}" in
    kind) release_kind=${value} ;;
    update_latest) update_latest=${value} ;;
    write_stable_version) write_stable_version=${value} ;;
    mark_official) mark_official=${value} ;;
    "") ;;
    *)
      echo "refuse release: unexpected policy field: ${key}" >&2
      exit 1
      ;;
  esac
done <<< "${PLAN}"

case "${release_kind}" in
  stable|prerelease) ;;
  *)
    echo "refuse release: unexpected kind: ${release_kind}" >&2
    exit 1
    ;;
esac
for flag in "${update_latest}" "${write_stable_version}" "${mark_official}"; do
  case "${flag}" in
    0|1) ;;
    *)
      echo "refuse release: unexpected policy flag: ${flag}" >&2
      exit 1
      ;;
  esac
done

if [[ "${FINISH_RELEASE_DRY_RUN:-}" == "1" ]]; then
  printf '%s\n' "${PLAN}"
  exit 0
fi

ALIYUN="./out/aliyun"

go build -tags aliyun_cli_packed_meta -ldflags "-X 'github.com/aliyun/aliyun-cli/v3/cli.Version=${VERSION}'" -o $ALIYUN ./main

FLAGS="oss://aliyun-cli --force --access-key-id ${ACCESS_KEY_ID} --access-key-secret ${ACCESS_KEY_SECRET} --region cn-hangzhou"

# mac amd64
${ALIYUN} oss cp ./aliyun-cli-macosx-"${VERSION}"-amd64.tgz $FLAGS
# mac arm64
${ALIYUN} oss cp ./aliyun-cli-macosx-"${VERSION}"-arm64.tgz $FLAGS
# mac universal
${ALIYUN} oss cp ./aliyun-cli-macosx-"${VERSION}"-universal.tgz $FLAGS
# mac pkg
${ALIYUN} oss cp ./aliyun-cli-"${VERSION}".pkg $FLAGS
  # linux amd64
${ALIYUN} oss cp ./aliyun-cli-linux-"${VERSION}"-amd64.tgz $FLAGS
# linux arm64
${ALIYUN} oss cp ./aliyun-cli-linux-"${VERSION}"-arm64.tgz $FLAGS
# windows
${ALIYUN} oss cp ./aliyun-cli-windows-"${VERSION}"-amd64.zip $FLAGS

if [[ "${update_latest}" != "1" && "${write_stable_version}" != "1" && "${mark_official}" != "1" ]]; then
  echo "prerelease. skip."
else
  if [[ "${update_latest}" == "1" ]]; then
    cp ./aliyun-cli-macosx-"${VERSION}"-amd64.tgz ./aliyun-cli-macosx-latest-amd64.tgz
    ${ALIYUN} oss cp ./aliyun-cli-macosx-latest-amd64.tgz $FLAGS

    cp ./aliyun-cli-macosx-"${VERSION}"-arm64.tgz ./aliyun-cli-macosx-latest-arm64.tgz
    ${ALIYUN} oss cp ./aliyun-cli-macosx-latest-arm64.tgz $FLAGS

    cp ./aliyun-cli-macosx-"${VERSION}"-universal.tgz ./aliyun-cli-macosx-latest-universal.tgz
    ${ALIYUN} oss cp ./aliyun-cli-macosx-latest-universal.tgz $FLAGS

    cp ./aliyun-cli-"${VERSION}".pkg ./aliyun-cli-latest.pkg
    ${ALIYUN} oss cp ./aliyun-cli-latest.pkg $FLAGS

    cp ./aliyun-cli-linux-"${VERSION}"-amd64.tgz ./aliyun-cli-linux-latest-amd64.tgz
    ${ALIYUN} oss cp ./aliyun-cli-linux-latest-amd64.tgz $FLAGS

    cp ./aliyun-cli-linux-"${VERSION}"-arm64.tgz ./aliyun-cli-linux-latest-arm64.tgz
    ${ALIYUN} oss cp ./aliyun-cli-linux-latest-arm64.tgz $FLAGS

    cp ./aliyun-cli-windows-"${VERSION}"-amd64.zip ./aliyun-cli-windows-latest-amd64.zip
    ${ALIYUN} oss cp ./aliyun-cli-windows-latest-amd64.zip $FLAGS
  fi

  if [[ "${write_stable_version}" == "1" ]]; then
    echo "${VERSION}" > out/version
    ${ALIYUN} oss cp out/version $FLAGS
  fi

  if [[ "${mark_official}" == "1" ]]; then
    RELEASE_ID=$(curl -fsSL \
      -H "Accept: application/vnd.github+json" \
      -H "Authorization: Bearer $GITHUB_TOKEN" \
      -H "X-GitHub-Api-Version: 2022-11-28" \
      https://api.github.com/repos/aliyun/aliyun-cli/releases/tags/v"$VERSION" | jq '.["id"]')

    DATA='{"draft":false,"prerelease":false,"make_latest":true}'

    curl -fsSL \
      -X PATCH \
      -H "Accept: application/vnd.github+json" \
      -H "Authorization: Bearer $GITHUB_TOKEN" \
      -H "X-GitHub-Api-Version: 2022-11-28" \
      https://api.github.com/repos/aliyun/aliyun-cli/releases/"$RELEASE_ID" \
      -d "$DATA"
  fi
fi
