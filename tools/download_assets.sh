#!/usr/bin/env bash

VERSION=$1

LIST=(
    "aliyun-cli-macosx-$VERSION-amd64.tgz"
    "aliyun-cli-macosx-$VERSION-arm64.tgz"
    "aliyun-cli-$VERSION.pkg"
    "aliyun-cli-macosx-$VERSION-universal.tgz"
    "aliyun-cli-linux-$VERSION-amd64.tgz"
    "aliyun-cli-linux-$VERSION-arm64.tgz"
    "aliyun-cli-windows-$VERSION-amd64.zip"
)

for filename in "${LIST[@]}"
do
    curl -fsSL -O \
        -H "Authorization: Bearer $GITHUB_TOKEN" \
        https://github.com/aliyun/aliyun-cli/releases/download/v"$VERSION"/"$filename"
    shasum -a 256 "$filename" >> SHASUMS256.txt
done

cat ./SHASUMS256.txt
