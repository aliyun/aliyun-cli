#!/usr/bin/env bash
# Print the go -ldflags for a release build.
set -euo pipefail

VERSION=${1:?usage: release_ldflags.sh <version>}
MODULE=github.com/aliyun/aliyun-cli/v3

printf -- "-X '%s/cli.Version=%s'\n" "${MODULE}" "${VERSION}"
