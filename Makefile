export VERSION=3.0.1

all: build
publish: build build_mac build_linux build_windows

deps:
	-go get github.com/aliyun/aliyun-openapi-meta
	go get -u github.com/jteeuwen/go-bindata/...
	go get -u github.com/Masterminds/glide
	glide install

testdeps: deps

metas: deps
	go-bindata -o resource/metas.go -pkg resource -prefix ../aliyun-openapi-meta ../aliyun-openapi-meta/**/* ../aliyun-openapi-meta/products.json

clean:
	rm -f resource/metas.go
	rm -rf out/*

build: metas
	go build -ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'" -o out/aliyun main/main.go

install: build
	cp out/aliyun /usr/local/bin

build_mac:
	GOOS=darwin GOARCH=amd64 go build -o out/aliyun main/main.go
	tar zcvf out/aliyun-cli-macosx-${VERSION}-amd64.tgz -C out aliyun
	aliyun oss cp out/aliyun-cli-macosx-${VERSION}-amd64.tgz oss://aliyun-cli --force --profile oss

build_linux:
	GOOS=linux GOARCH=amd64 go build -o out/aliyun main/main.go
	tar zcvf out/aliyun-cli-linux-${VERSION}-amd64.tgz -C out aliyun
	aliyun oss cp out/aliyun-cli-linux-${VERSION}-amd64.tgz oss://aliyun-cli --force --profile oss

build_windows:
	GOOS=windows GOARCH=amd64 go build -o aliyun.exe main/main.go
	zip -r out/aliyun-cli-windows-${VERSION}-amd64.zip aliyun.exe
	aliyun oss cp out/aliyun-cli-windows-${VERSION}-amd64.zip oss://aliyun-cli --force --profile oss
	rm aliyun.exe

