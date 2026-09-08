#!/usr/bin/env bash
# Offline simulation of finish_release policy. Does not call OSS or GitHub.
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "${ROOT}"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

expect_plan() {
  local version=$1
  local kind=$2
  local latest=$3
  local stable_version=$4
  local official=$5
  local out
  if ! out=$(FINISH_RELEASE_DRY_RUN=1 bash tools/finish_release.sh "${version}"); then
    fail "${version}: expected success, got failure"
  fi
  local want
  want=$(printf 'kind=%s\nupdate_latest=%s\nwrite_stable_version=%s\nmark_official=%s\n' \
    "${kind}" "${latest}" "${stable_version}" "${official}")
  if [[ "${out}" != "${want}" ]]; then
    echo "got:" >&2
    printf '%s\n' "${out}" >&2
    echo "want:" >&2
    printf '%s\n' "${want}" >&2
    fail "${version}: plan mismatch"
  fi
  echo "ok ${version} kind=${kind} update_latest=${latest} mark_official=${official}"
}

expect_refuse() {
  local version=$1
  local out
  local status=0
  out=$(FINISH_RELEASE_DRY_RUN=1 bash tools/finish_release.sh "${version}" 2>&1) || status=$?
  if [[ "${status}" -eq 0 ]]; then
    fail "${version}: expected refuse, got success: ${out}"
  fi
  if [[ "${out}" != *"invalid"* && "${out}" != *"required"* ]]; then
    fail "${version}: expected refuse error, got: ${out}"
  fi
  if [[ "${out}" == *"update_latest=1"* || "${out}" == *"mark_official=1"* ]]; then
    fail "${version}: refused version must not promote latest/official"
  fi
  echo "ok ${version} refused"
}

expect_plan "3.5.1" stable 1 1 1
expect_plan "3.5.1-beta" prerelease 0 0 0
expect_plan "3.5.1-beta.1" prerelease 0 0 0
expect_plan "3.5.1-rc.1" prerelease 0 0 0

expect_refuse "latest"
expect_refuse "3.5"
expect_refuse "3.5.1-"
expect_refuse ""

if (cd "${ROOT}" && bash tools/create_release.sh "not-a-version") >/tmp/create_release_invalid.out 2>&1; then
  fail "create_release.sh accepted invalid tag"
fi
if ! grep -q "invalid" /tmp/create_release_invalid.out; then
  fail "create_release.sh did not report invalid version"
fi
echo "ok create_release.sh refused invalid tag"
echo "offline release policy simulation passed"
