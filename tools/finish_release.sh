#!/usr/bin/env bash

set -e

VERSION=$1

ALIYUN="./out/aliyun"

go build -ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'" -o $ALIYUN main/main.go

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

if [[ "$VERSION" == *"-beta" ]]; then
  echo "beta. skip."
else
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
  # local version

  echo "${VERSION}" > out/version
  ${ALIYUN} oss cp out/version $FLAGS

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
