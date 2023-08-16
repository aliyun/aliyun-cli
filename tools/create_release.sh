#!/usr/bin/env bash

TAGNAME=$1

DATA='{"tag_name":"'$TAGNAME'","name":"'$TAGNAME'","body":"TBD","draft":false,"prerelease":true,"generate_release_notes":false}'

curl -fsSL \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/repos/aliyun/aliyun-cli/releases \
  -d "$DATA"
