#!/usr/bin/env bash
# Print the go -ldflags for a release build.
#
# Release binaries must embed the public plugin trust anchors; without them
# official plugin installs fail with "artifact trust bootstrap root is not
# configured". The values are public and come from the GitHub repository
# variables of the same name.
set -euo pipefail

VERSION=${1:?usage: release_ldflags.sh <version>}
MODULE=github.com/aliyun/aliyun-cli/v3

for name in CLI_TRUST_ROOT_ENVELOPE_BASE64 CLI_TRUST_RECOVERY_KEYS_BASE64 CLI_TRUST_RECOVERY_ROLE_BASE64; do
  value=${!name:-}
  if [[ -z "${value}" ]]; then
    echo "refuse release build: ${name} is not set" >&2
    exit 1
  fi
  if [[ ! "${value}" =~ ^[A-Za-z0-9+/]+=*$ ]]; then
    echo "refuse release build: ${name} is not base64" >&2
    exit 1
  fi
done

printf '%s' "-X '${MODULE}/cli.Version=${VERSION}'"
printf ' %s' "-X '${MODULE}/cli/trust.BuiltinRootEnvelopeBase64=${CLI_TRUST_ROOT_ENVELOPE_BASE64}'"
printf ' %s' "-X '${MODULE}/cli/trust.BuiltinRecoveryKeysBase64=${CLI_TRUST_RECOVERY_KEYS_BASE64}'"
printf ' %s\n' "-X '${MODULE}/cli/trust.BuiltinRecoveryRoleBase64=${CLI_TRUST_RECOVERY_ROLE_BASE64}'"
