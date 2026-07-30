export VERSION ?= 3.0.0-beta
export RELEASE_PATH="releases/aliyun-cli-${VERSION}"

MODULE := github.com/aliyun/aliyun-cli/v3
META_DIR := $(CURDIR)/aliyun-openapi-meta
META_TAG := aliyun_cli_packed_meta
LDFLAGS := -X '$(MODULE)/cli.Version=$(VERSION)'

all: build
publish: build build_mac build_linux build_windows build_linux_arm64 gen_version

deps:
	git submodule update --init --recursive

clean:
	rm -rf out/*
	rm -rf bundledmeta/.generated

run: deps
	ALIYUN_CLI_META_DIR="$(META_DIR)" go run ./main/main.go

meta-pack: deps
	go generate ./bundledmeta

build: meta-pack
	mkdir -p out
	CGO_ENABLED=0 go build -tags "$(META_TAG)" -ldflags "$(LDFLAGS)" -o out/aliyun ./main/main.go

install: build
	cp out/aliyun /usr/local/bin

build_mac: meta-pack
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -tags "$(META_TAG)" -ldflags "$(LDFLAGS)" -o out/aliyun ./main/main.go
	tar zcvf out/aliyun-cli-macosx-${VERSION}-amd64.tgz -C out aliyun
	aliyun oss cp out/aliyun-cli-macosx-${VERSION}-amd64.tgz oss://aliyun-cli --force --profile oss

build_linux: meta-pack
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags "$(META_TAG)" -ldflags "$(LDFLAGS)" -o out/aliyun ./main/main.go
	tar zcvf out/aliyun-cli-linux-${VERSION}-amd64.tgz -C out aliyun
	aliyun oss cp out/aliyun-cli-linux-${VERSION}-amd64.tgz oss://aliyun-cli --force --profile oss

build_windows: meta-pack
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags "$(META_TAG)" -ldflags "$(LDFLAGS)" -o aliyun.exe ./main/main.go
	zip -r out/aliyun-cli-windows-${VERSION}-amd64.zip aliyun.exe
	aliyun oss cp out/aliyun-cli-windows-${VERSION}-amd64.zip oss://aliyun-cli --force --profile oss
	rm aliyun.exe

build_linux_arm64: meta-pack
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags "$(META_TAG)" -ldflags "$(LDFLAGS)" -o out/aliyun ./main/main.go
	tar zcvf out/aliyun-cli-linux-${VERSION}-arm64.tgz -C out aliyun
	aliyun oss cp out/aliyun-cli-linux-${VERSION}-arm64.tgz oss://aliyun-cli --force --profile oss

gen_version:
	-rm out/version
	echo ${VERSION} >> out/version
	aliyun oss cp out/version oss://aliyun-cli --force --profile oss

git_release: clean build make_release_dir release_mac release_linux release_linux_arm64 release_windows

make_release_dir:
	mkdir -p ${RELEASE_PATH}

release_mac: meta-pack
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -tags "$(META_TAG)" -ldflags "$(LDFLAGS)" -o out/aliyun ./main/main.go
	tar zcvf ${RELEASE_PATH}/aliyun-cli-darwin-amd64.tar.gz -C out aliyun

release_mac_arm64: meta-pack
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -tags "$(META_TAG)" -ldflags "$(LDFLAGS)" -o out/aliyun ./main/main.go
	tar zcvf ${RELEASE_PATH}/aliyun-cli-darwin-arm64.tar.gz -C out aliyun

release_linux: meta-pack
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags "$(META_TAG)" -ldflags "$(LDFLAGS)" -o out/aliyun ./main/main.go
	tar zcvf ${RELEASE_PATH}/aliyun-cli-linux-amd64.tar.gz -C out aliyun

release_linux_arm64: meta-pack
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags "$(META_TAG)" -ldflags "$(LDFLAGS)" -o out/aliyun ./main/main.go
	tar zcvf ${RELEASE_PATH}/aliyun-cli-linux-arm64.tar.gz -C out aliyun

release_windows: meta-pack
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags "$(META_TAG)" -ldflags "$(LDFLAGS)" -o aliyun.exe ./main/main.go
	zip -r ${RELEASE_PATH}/aliyun-cli-windows-amd64.exe.zip aliyun.exe
	rm aliyun.exe

fmt:
	go fmt ./bundledmeta/... ./util/... ./cli/... ./config/... ./i18n/... ./main/... ./openapi/... ./oss/... ./resource/... ./meta/... ./export/...

test: deps
	ALIYUN_CLI_META_DIR="$(META_DIR)" LANG="en_US.UTF-8" go test -race -coverprofile=coverage.txt -covermode=atomic ./bundledmeta ./util/... ./cli/... ./config/... ./i18n/... ./main/... ./openapi/... ./meta/... ./export/...
	go tool cover -html=coverage.txt -o coverage.html

test-release: meta-pack
	LANG="en_US.UTF-8" go test -tags "$(META_TAG)" ./bundledmeta ./meta ./export ./openapi/runtimehost

.PHONY: all publish install
.PHONY: deps clean run fmt test test-release
.PHONY: meta-pack build build_mac build_linux build_windows build_linux_arm64
.PHONY: gen_version git_release make_release_dir
.PHONY: release_mac release_mac_arm64 release_linux release_linux_arm64 release_windows
