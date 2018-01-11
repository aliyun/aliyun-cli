export VERSION=0.11
all: build
release: build build_mac build_linux build_windows

testdeps:
	go get -v github.com/onsi/ginkgo/ginkgo
	go get -v github.com/onsi/gomega
	go install github.com/onsi/ginkgo/ginkgo

build:
	go build -o out/aliyun main/main.go
	tar zcvf out/aliyun-cli-macosx-${VERSION}-amd64.tgz out/aliyun

build_mac:
	export GOOS=darwin
	export GOARCH=amd64
	go build -o out/aliyun main/main.go
	tar zcvf out/aliyun-cli-macosx-${VERSION}-amd64.tgz out/aliyun

build_linux:
	export GOOS=linux
	export GOARCH=amd64
	go build -o out/aliyun main/main.go
	tar zcvf out/aliyun-cli-linux-${VERSION}-amd64.tgz out/aliyun

build_windows:
	export GOOS=windows
	export GOARCH=amd64
	go build -o out/aliyun.exe main/main.go
	tar zcvf out/aliyun-cli-windows-${VERSION}-amd64.tgz out/aliyun.exe
