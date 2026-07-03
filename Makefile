export VERSION=3.0.0-beta
export RELEASE_PATH="releases/aliyun-cli-${VERSION}"

all: build
publish: build build_mac build_linux build_windows build_linux_arm64 gen_version

deps:
	git submodule update --init --recursive

clean:
	rm -rf out/*

build: deps
	go build -ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'" -o out/aliyun main/main.go

install: build
	cp out/aliyun /usr/local/bin

build_mac:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'" -o out/aliyun main/main.go
	tar zcvf out/aliyun-cli-macosx-${VERSION}-amd64.tgz -C out aliyun
	aliyun oss cp out/aliyun-cli-macosx-${VERSION}-amd64.tgz oss://aliyun-cli --force --profile oss

build_linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'" -o out/aliyun main/main.go
	tar zcvf out/aliyun-cli-linux-${VERSION}-amd64.tgz -C out aliyun
	aliyun oss cp out/aliyun-cli-linux-${VERSION}-amd64.tgz oss://aliyun-cli --force --profile oss

build_windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'" -o aliyun.exe main/main.go
	zip -r out/aliyun-cli-windows-${VERSION}-amd64.zip aliyun.exe
	aliyun oss cp out/aliyun-cli-windows-${VERSION}-amd64.zip oss://aliyun-cli --force --profile oss
	rm aliyun.exe

build_linux_arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'" -o out/aliyun main/main.go
	tar zcvf out/aliyun-cli-linux-${VERSION}-arm64.tgz -C out aliyun
	aliyun oss cp out/aliyun-cli-linux-${VERSION}-arm64.tgz oss://aliyun-cli --force --profile oss

gen_version:
	-rm out/version
	echo ${VERSION} >> out/version
	aliyun oss cp out/version oss://aliyun-cli --force --profile oss

git_release: clean build make_release_dir release_mac release_linux release_linux_arm64 release_windows

make_release_dir:
	mkdir -p ${RELEASE_PATH}

release_mac:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'" -o out/aliyun main/main.go
	tar zcvf ${RELEASE_PATH}/aliyun-cli-darwin-amd64.tar.gz -C out aliyun

release_mac_arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'" -o out/aliyun main/main.go
	tar zcvf ${RELEASE_PATH}/aliyun-cli-darwin-arm64.tar.gz -C out aliyun

release_linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'" -o out/aliyun main/main.go
	tar zcvf ${RELEASE_PATH}/aliyun-cli-linux-amd64.tar.gz -C out aliyun

release_linux_arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'" -o out/aliyun main/main.go
	tar zcvf ${RELEASE_PATH}/aliyun-cli-linux-arm64.tar.gz -C out aliyun

release_windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X 'github.com/aliyun/aliyun-cli/cli.Version=${VERSION}'" -o aliyun.exe main/main.go
	zip -r ${RELEASE_PATH}/aliyun-cli-windows-amd64.exe.zip aliyun.exe
	rm aliyun.exe

fmt:
	go fmt ./util/... ./cli/... ./config/... ./i18n/... ./main/... ./openapi/... ./oss/... ./resource/... ./meta/...

test:
	LANG="en_US.UTF-8" go test -race -coverprofile=coverage.txt -covermode=atomic ./util/... ./cli/... ./config/... ./i18n/... ./main/... ./openapi/... ./meta/...
	go tool cover -html=coverage.txt -o coverage.html

TLA2TOOLS_JAR ?= $(HOME)/.cache/tla2tools/tla2tools.jar
TLC_METADIR ?= /tmp/aliyun-cli-tlc-parser-states
TLC_CASE_METADIR ?= /tmp/aliyun-cli-tlc-parser-case-enum
PARSER_CASE_DUMP ?= formal/parser/generated/abstract_cases.dump
PARSER_CASE_JSON ?= cli/testdata/parser_contract_cases.json
PARSER_CASE_COVERAGE ?= docs/formal/parser-case-coverage.md

formal-parser:
	java -jar $(TLA2TOOLS_JAR) -metadir $(TLC_METADIR) -config formal/parser/Parser.cfg formal/parser/Parser.tla

formal-parser-cases:
	mkdir -p formal/parser/generated
	java -jar $(TLA2TOOLS_JAR) -metadir $(TLC_CASE_METADIR) -config formal/parser/ParserCases.cfg -dump $(PARSER_CASE_DUMP) formal/parser/Parser.tla

parser-contract-gen:
	go run ./tools/parser-contract-gen -dump $(PARSER_CASE_DUMP) -out $(PARSER_CASE_JSON) -coverage $(PARSER_CASE_COVERAGE)

parser-contract-test:
	go test -tags parser_contract ./cli -run TestParserContractCases -count=1
