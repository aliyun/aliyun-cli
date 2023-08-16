#!/usr/bin/env bash

TAG=v$1
ASSET=$2

if [[ $ASSET == ".tgz" ]]
then
    TYPE=application/x-compressed-tar
else
    TYPE=application/zip
fi

RELEASE_ID=$(curl -L \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/repos/aliyun/aliyun-cli/releases/tags/$TAG | jq '.["id"]')

curl -L \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -H "Content-Type: $TYPE" \
  "https://uploads.github.com/repos/aliyun/aliyun-cli/releases/$RELEASE_ID/assets?name=$(basename $ASSET)" \
  --data-binary "@$ASSET"
