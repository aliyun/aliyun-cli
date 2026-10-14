# Alibaba Cloud OSSUTIL

This document describes the standalone ossutil version recorded in this directory. For the built-in `aliyun oss` commands, see the [OSS automation guide](../docs/en/oss.md).

[![GitHub Version](https://badge.fury.io/gh/aliyun%2Fossutil.svg)](https://badge.fury.io/gh/aliyun%2Fossutil)
[![Build Status](https://travis-ci.org/aliyun/ossutil.svg?branch=master)](https://travis-ci.org/aliyun/ossutil)
[![Coverage Status](https://coveralls.io/repos/github/aliyun/ossutil/badge.svg?branch=master)](https://coveralls.io/github/aliyun/ossutil?branch=master)

### [README of Chinese](https://github.com/aliyun/ossutil/blob/master/README-CN.md)

## About

ossutil is a Go command-line tool built on the Alibaba Cloud OSS SDK. Use it to manage buckets, objects, and multipart upload tasks. Check command help for supported operations.

## Version
- Version recorded in this directory: v1.7.19

## Supported platforms
- Linux
- Windows
- macOS

## Dependencies
- goopt (github.com/droundy/goopt)
- configparser (github.com/alyu/configparser)
- leveldb (github.com/syndtr/goleveldb/leveldb)
- oss (github.com/aliyun/aliyun-oss-go-sdk/oss)
- gopkg.in/check.v1 (gopkg.in/check.v1)

## Quick start
### Get the command list
```sh
    ./ossutil
    # Alternatively:
    ./ossutil help
```

### View the help documentation for a command
```sh
    ./ossutil help cmd
```

### Configure OSSUTIL
```sh
    ./ossutil config
```

### List buckets
```sh
    ./ossutil ls
    # Alternatively:
    ./ossutil ls oss://
```

### List objects and multipart upload tasks
```sh
    ./ossutil ls -a
    # Alternatively:
    ./ossutil ls oss:// -a
```

### Upload a file
```sh
    ./ossutil cp localfile oss://bucket
```

### More commands
Run `./ossutil help <command>` to view help for a command.

## Notes
### Run OSSUTIL
- For a legacy GOPATH development environment, configure your Go project directory.
- Use `go get` to get the library that ossutil depends on.
- Run `go get github.com/aliyun/ossutil`.
- Enter the *src* directory under the Go project directory and build to generate the OSSUTIL tool. For example, on Linux, you can run `go build github.com/aliyun/ossutil/ossutil.go`.
- Refer to the example above to run the OSSUTIL tool.

### Test
- Configure the endpoint and test credentials required by the standalone ossutil tests. Do not commit real credentials.
- Run `go test` under the *lib* directory.

## Contact us
- [Alibaba Cloud OSS official website](http://oss.aliyun.com).
- [Alibaba Cloud OSS official forum](http://bbs.aliyun.com).
- [Alibaba Cloud OSS official documentation center](http://www.aliyun.com/product/oss#Docs).

## Author
- [Ting Zhang](https://github.com/dengwu12)

## License
- [MIT](https://github.com/aliyun/ossutil/blob/master/LICENSE)
