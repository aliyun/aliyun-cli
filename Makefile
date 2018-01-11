export VERSION=0.1
all: build

testdeps:
	go get -v github.com/onsi/ginkgo/ginkgo
	go get -v github.com/onsi/gomega
	go install github.com/onsi/ginkgo/ginkgo

build:
	go build -o aliyun main/main.go
	tar zcvf aliyun-cli-macosx-${VERSION}-amd64.tgz aliyun

build_mac:
	export GOOS=darwin
	export GOARCH=amd64
	go build -o aliyun main/main.go
	tar zcvf aliyun-cli-macosx-${VERSION}-amd64.tgz aliyun

build_linux:
	export GOOS=linux
	export GOARCH=amd64
	go build -o aliyun main/main.go
	tar zcvf aliyun-cli-linux-${VERSION}-amd64.tgz aliyun

build_windows:
	export GOOS=windows
	export GOARCH=amd64
	go build -o aliyun.exe main/main.go
	tar zcvf aliyun-cli-windows-${VERSION}-amd64.tgz aliyun.exe
