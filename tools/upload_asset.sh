#!/usr/bin/env bash

TAG=v$1
ASSET=$2

if [[ $ASSET == *.tgz ]]
then
    TYPE=application/x-compressed-tar
elif [[ $ASSET == *.txt ]]
then
    TYPE=text/plain
else
    TYPE=application/zip
fi


RELEASE_ID=$(curl -fsSL \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/repos/aliyun/aliyun-cli/releases/tags/$TAG | jq '.["id"]')

curl -fsSL \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -H "Content-Type: $TYPE" \
  "https://uploads.github.com/repos/aliyun/aliyun-cli/releases/$RELEASE_ID/assets?name=$(basename $ASSET)" \
  --data-binary "@$ASSET"
