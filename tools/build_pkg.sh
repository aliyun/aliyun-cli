#!/usr/bin/env bash

VERSION=$1

mkdir -p out/dist/usr/local/bin/
mkdir -p out/productbuild
cp out/aliyun out/dist/usr/local/bin/

mkdir -p out/pkgs
pkgbuild --version "$VERSION" \
    --identifier com.aliyun.cli.pkg \
    --root out/dist out/pkgs/aliyun-cli-"${VERSION}".pkg

cat pkg/productbuild/distribution.xml.tmpl | \
    sed -E "s/\\{cli_version\\}/$VERSION/g" > out/productbuild/distribution.xml

for dirname in pkg/productbuild/Resources/*/; do
    lang=$(basename "$dirname")
    mkdir -p out/productbuild/Resources/"$lang"
    printf "Found localization directory %s\n" "$dirname"
    cat "$dirname"/welcome.html.tmpl | \
        sed -E "s/\\{cli_version\\}/$VERSION/g" > out/productbuild/Resources/"$lang"/welcome.html
    cat "$dirname"/conclusion.html.tmpl | \
        sed -E "s/\\{cli_version\\}/$VERSION/g" > out/productbuild/Resources/"$lang"/conclusion.html
done

cp pkg/osx_installer_logo.png out/productbuild/Resources
cp LICENSE out/productbuild/Resources/license.txt

productbuild --distribution out/productbuild/distribution.xml \
    --resources out/productbuild/Resources \
    --package-path out/pkgs \
    out/aliyun-cli-"${VERSION}".pkg
