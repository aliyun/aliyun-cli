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

# Upload the publisher signature before the bytes it covers. Versioned
# signatures come from the build jobs. latest aliases are signed here
# because the asset name is part of the signed message.
publish_oss() {
  local file=$1
  if [[ ! -f "${file}.minisig" ]]; then
    bash "${SCRIPT_DIR}/sign_asset.sh" "${file}"
  fi
  ${ALIYUN} oss cp "${file}.minisig" $FLAGS
  ${ALIYUN} oss cp "${file}" $FLAGS
}

publish_traced() {
  local file=$1
  publish_oss "${file}"
  publish_oss "${file}.cdx.json"
  publish_oss "${file}.provenance.json"
}

retarget_latest() {
  local src=$1
  local dst=$2
  cp "${src}" "${dst}"
  cp "${src}.cdx.json" "${dst}.cdx.json"
  go run ./tools/buildtrace/cmd/buildtrace retarget \
    --from "${src}.provenance.json" \
    --asset "${dst}" \
    --sbom "${dst}.cdx.json"
  publish_traced "${dst}"
}

# mac amd64
publish_traced ./aliyun-cli-macosx-"${VERSION}"-amd64.tgz
# mac arm64
publish_traced ./aliyun-cli-macosx-"${VERSION}"-arm64.tgz
# mac universal
publish_traced ./aliyun-cli-macosx-"${VERSION}"-universal.tgz
# mac pkg
publish_traced ./aliyun-cli-"${VERSION}".pkg
# linux amd64
publish_traced ./aliyun-cli-linux-"${VERSION}"-amd64.tgz
# linux arm64
publish_traced ./aliyun-cli-linux-"${VERSION}"-arm64.tgz
# windows
publish_traced ./aliyun-cli-windows-"${VERSION}"-amd64.zip

if [[ "${update_latest}" != "1" && "${write_stable_version}" != "1" && "${mark_official}" != "1" ]]; then
  echo "prerelease. skip."
else
  if [[ "${update_latest}" == "1" ]]; then
    retarget_latest ./aliyun-cli-macosx-"${VERSION}"-amd64.tgz ./aliyun-cli-macosx-latest-amd64.tgz
    retarget_latest ./aliyun-cli-macosx-"${VERSION}"-arm64.tgz ./aliyun-cli-macosx-latest-arm64.tgz
    retarget_latest ./aliyun-cli-macosx-"${VERSION}"-universal.tgz ./aliyun-cli-macosx-latest-universal.tgz
    retarget_latest ./aliyun-cli-"${VERSION}".pkg ./aliyun-cli-latest.pkg
    retarget_latest ./aliyun-cli-linux-"${VERSION}"-amd64.tgz ./aliyun-cli-linux-latest-amd64.tgz
    retarget_latest ./aliyun-cli-linux-"${VERSION}"-arm64.tgz ./aliyun-cli-linux-latest-arm64.tgz
    retarget_latest ./aliyun-cli-windows-"${VERSION}"-amd64.zip ./aliyun-cli-windows-latest-amd64.zip
  fi

  if [[ "${write_stable_version}" == "1" ]]; then
    # /version stays one version plus a newline. History is a separate object
    # so already-released CLIs keep reading a single version token.
    versions_file="out/versions"
    versions_err=$(mktemp)
    rm -f "${versions_file}"
    set +e
    ${ALIYUN} oss cp oss://aliyun-cli/versions "${versions_file}" \
      --access-key-id "${ACCESS_KEY_ID}" \
      --access-key-secret "${ACCESS_KEY_SECRET}" \
      --region cn-hangzhou >"${versions_err}" 2>&1
    versions_status=$?
    set -e
    if [[ ${versions_status} -ne 0 ]]; then
      if grep -Eq 'NoSuchKey|No such object|does not exist|404' "${versions_err}"; then
        : > "${versions_file}"
      else
        echo "refuse release: cannot read oss://aliyun-cli/versions" >&2
        cat "${versions_err}" >&2
        rm -f "${versions_err}"
        exit 1
      fi
    fi
    rm -f "${versions_err}"
    # The list is the key generations this binary supports, not a range inferred
    # from neighboring versions. CLI_RELEASE_PUBLIC_KEYS overrides it, for
    # example "1,2" on a bridge release.
    public_keys="${CLI_RELEASE_PUBLIC_KEYS:-}"
    if [[ -z "${public_keys}" ]]; then
      public_keys=$(go run ./tools/releasever/cmd/releasever supported-public-keys)
    fi
    go run ./tools/releasever/cmd/releasever append-version -- "${VERSION}" "${versions_file}" "${public_keys}" > out/versions.next
    mv out/versions.next "${versions_file}"
    ${ALIYUN} oss cp "${versions_file}" oss://aliyun-cli --force \
      --access-key-id "${ACCESS_KEY_ID}" \
      --access-key-secret "${ACCESS_KEY_SECRET}" \
      --region cn-hangzhou
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
