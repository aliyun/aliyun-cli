export VERSION=0.70

all: build
publish: build build_mac build_linux build_windows

deps:
	go get gopkg.in/ini.v1
	go get github.com/droundy/goopt
	go get github.com/alyu/configparser
	go get github.com/syndtr/goleveldb/leveldb
	go get github.com/aliyun/aliyun-oss-go-sdk/oss
	go get -u github.com/jteeuwen/go-bindata/...
	go get github.com/jmespath/go-jmespath
	go get github.com/aliyun/alibaba-cloud-sdk-go/sdk
	go get github.com/aliyun/aliyun-openapi-meta

testdeps:
	go get -v github.com/onsi/ginkgo/ginkgo
	go get -v github.com/onsi/gomega
	go install github.com/onsi/ginkgo/ginkgo

metas:
	go-bindata -o resource/metas.go -pkg resource -prefix ../aliyun-openapi-meta ../aliyun-openapi-meta/**/* ../aliyun-openapi-meta/products.json

clean:
	rm -f resource/metas.go
	rm -rf out/*

build: metas
	go build -o out/aliyun main/main.go

install: build
	cp out/aliyun /usr/local/bin

build_mac:
	GOOS=darwin GOARCH=amd64 go build -o out/aliyun main/main.go
	tar zcvf out/aliyun-cli-macosx-${VERSION}-amd64.tgz -C out aliyun
	aliyun oss cp out/aliyun-cli-macosx-${VERSION}-amd64.tgz oss://aliyun-cli --force

build_linux:
	GOOS=linux GOARCH=amd64 go build -o out/aliyun main/main.go
	tar zcvf out/aliyun-cli-linux-${VERSION}-amd64.tgz -C out aliyun
	aliyun oss cp out/aliyun-cli-linux-${VERSION}-amd64.tgz oss://aliyun-cli --force

build_windows:
	GOOS=windows GOARCH=amd64 go build -o aliyun.exe main/main.go
	zip -r out/aliyun-cli-windows-${VERSION}-amd64.zip aliyun.exe
	aliyun oss cp out/aliyun-cli-windows-${VERSION}-amd64.zip oss://aliyun-cli --force
	rm aliyun.exe

