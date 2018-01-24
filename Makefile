export VERSION=0.16
all: build
release: build build_mac build_linux build_windows

testdeps:
	go get -v github.com/onsi/ginkgo/ginkgo
	go get -v github.com/onsi/gomega
	go install github.com/onsi/ginkgo/ginkgo

build:
	go build -o out/aliyun main/main.go

install: build
	cp out/aliyun /usr/local/bin

build_mac:
	GOOS=darwin GOARCH=amd64 go build -o out/aliyun main/main.go
	tar zcvf out/aliyun-cli-macosx-${VERSION}-amd64.tgz -C out aliyun

build_linux:
	GOOS=linux GOARCH=amd64 go build -o out/aliyun main/main.go
	tar zcvf out/aliyun-cli-linux-${VERSION}-amd64.tgz -C out aliyun

build_windows:
	GOOS=windows GOARCH=amd64 go build -o out/aliyun.exe main/main.go
	tar zcvf out/aliyun-cli-windows-${VERSION}-amd64.tgz -C out aliyun.exe
