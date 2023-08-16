#!/usr/bin/env bash

TAGNAME=$1
RELEASE_NAME=$1

DATA='{"tag_name":"'$TAGNAME'","name":"'$RELEASE_NAME'","body":"TBD","draft":true,"prerelease":true,"generate_release_notes":false}'

curl -L \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/repos/aliyun/aliyun-cli/releases \
  -d "$DATA"
