#!/usr/bin/env bash

VERSION=$1

RELEASE_ID=$(curl -fsSL \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/repos/aliyun/aliyun-cli/releases/tags/v$VERSION | jq '.["id"]')

LIST=(
    "aliyun-cli-$VERSION.pkg"
    "aliyun-cli-macosx-$VERSION-universal.tgz"
    "aliyun-cli-linux-$VERSION-amd64.tgz"
    "aliyun-cli-linux-$VERSION-arm64.tgz"
    "aliyun-cli-windows-$VERSION-amd64.zip"
)

for filename in ${LIST[@]}
do
    curl -fsSL -O \
        -H "Authorization: Bearer $GITHUB_TOKEN" \
        https://github.com/aliyun/aliyun-cli/releases/download/v$VERSION/$filename
    shasum -a 256 $filename >> SHASUMS256.txt
done

cat ./SHASUMS256.txt
