#!/usr/bin/env bash
# Print the go -ldflags for a release build.
#
# Source already contains the public plugin trust anchors, so Homebrew and
# other source builds work without GitHub Actions variables. When all three
# variables are set, they override the source anchors. A partial set is refused.
set -euo pipefail

VERSION=${1:?usage: release_ldflags.sh <version>}
MODULE=github.com/aliyun/aliyun-cli/v3

names=(CLI_TRUST_ROOT_ENVELOPE_BASE64 CLI_TRUST_RECOVERY_KEYS_BASE64 CLI_TRUST_RECOVERY_ROLE_BASE64)
present=0
for name in "${names[@]}"; do
  value=${!name:-}
  if [[ -n "${value}" ]]; then
    present=$((present + 1))
    if [[ ! "${value}" =~ ^[A-Za-z0-9+/]+=*$ ]]; then
      echo "refuse release build: ${name} is not base64" >&2
      exit 1
    fi
  fi
done
if [[ "${present}" -ne 0 && "${present}" -ne 3 ]]; then
  echo "refuse release build: set all three CLI_TRUST_* anchors or none" >&2
  exit 1
fi

printf '%s' "-X '${MODULE}/cli.Version=${VERSION}'"
if [[ "${present}" -eq 3 ]]; then
  printf ' %s' "-X '${MODULE}/cli/trust.BuiltinRootEnvelopeBase64=${CLI_TRUST_ROOT_ENVELOPE_BASE64}'"
  printf ' %s' "-X '${MODULE}/cli/trust.BuiltinRecoveryKeysBase64=${CLI_TRUST_RECOVERY_KEYS_BASE64}'"
  printf ' %s' "-X '${MODULE}/cli/trust.BuiltinRecoveryRoleBase64=${CLI_TRUST_RECOVERY_ROLE_BASE64}'"
fi
printf '\n'
