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
  https://api.github.com/repos/aliyun/aliyun-cli/releases/tags/"$TAG" | jq '.["id"]')

if [ -z "$RELEASE_ID" ]; then
  echo "Failed to get release ID for tag $TAG"
  exit 1
fi

# 获取现有资产列表
ASSET_ID=$(curl -fsSL \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/repos/aliyun/aliyun-cli/releases/$RELEASE_ID/assets | jq -r ".[] | select(.name == \"$(basename "$ASSET")\") | .id")

# 如果资产已存在，删除它
if [ -n "$ASSET_ID" ]; then
  echo "Asset already exists. Deleting asset ID $ASSET_ID"
  curl -fsSL \
    -X DELETE \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $GITHUB_TOKEN" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    https://api.github.com/repos/aliyun/aliyun-cli/releases/assets/$ASSET_ID
fi

printf "Uploading %s to release %s\n" "$ASSET" "$RELEASE_ID"

curl -fsSL \
  -X PUT \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -H "Content-Type: $TYPE" \
  "https://uploads.github.com/repos/aliyun/aliyun-cli/releases/$RELEASE_ID/assets?name=$(basename "$ASSET")" \
  --data-binary "@$ASSET"
