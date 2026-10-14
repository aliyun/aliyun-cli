# Aliyun OSSUTIL

本文保留独立 ossutil 工具的说明，适用于本目录所记录的版本。阿里云 CLI 内置 `aliyun oss` 的用法见[OSS 自动化指南](../docs/zh-CN/oss.md)。

[![GitHub version](https://badge.fury.io/gh/aliyun%2Fossutil.svg)](https://badge.fury.io/gh/aliyun%2Fossutil)
[![Build Status](https://travis-ci.org/aliyun/ossutil.svg?branch=master)](https://travis-ci.org/aliyun/ossutil)
[![Coverage Status](https://coveralls.io/repos/github/aliyun/ossutil/badge.svg?branch=master)](https://coveralls.io/github/aliyun/ossutil?branch=master)

### [README of English](https://github.com/aliyun/ossutil/blob/master/README.md)

## 关于

ossutil 是使用 Go 编写的 OSS 命令行工具，基于阿里云 OSS SDK 构建，可用于管理存储空间、对象和分片上传任务。各命令的支持范围以命令帮助为准。

## 版本
- 本目录记录的版本：v1.7.19

## 运行环境
- Linux
- Windows
- macOS

## 依赖的库
- goopt (github.com/droundy/goopt)
- configparser (github.com/alyu/configparser)
- leveldb (github.com/syndtr/goleveldb/leveldb)
- oss (github.com/aliyun/aliyun-oss-go-sdk/oss)
- gopkg.in/check.v1 (gopkg.in/check.v1)

## 快速使用
### 获取命令列表
```sh
    ./ossutil
    或 ./ossutil help
```

### 查看某命令的帮助文档
```sh
    ./ossutil help cmd
```

### 配置 ossutil
```sh
    ./ossutil config
```

### 列举存储空间（Bucket）
```sh
    ./ossutil ls
    或 ./ossutil ls oss://
```

### 列举对象和分片上传任务
```sh
    ./ossutil ls -a
    或 ./ossutil ls oss:// -a
```

### 上传文件
```sh
    ./ossutil cp localfile oss://bucket
```

### 更多命令
运行 `./ossutil help <命令名>` 查看指定命令的帮助。

## 注意事项
### 运行
- 首先配置Go 工程目录。
- 使用 `go get` 获取依赖库。
- 运行 `go get github.com/aliyun/ossutil`。
- 在旧版 GOPATH 开发环境中，进入 `src` 目录，运行 `go build github.com/aliyun/ossutil/ossutil.go` 构建工具。
- 参考上面示例运行ossutil工具。

### 测试
- 在独立 ossutil 源码中，按测试要求配置 Endpoint 和测试凭证。不要提交真实凭证。
- 在 `lib` 目录下执行`go test`。

## 联系我们
- [阿里云OSS官方网站](http://oss.aliyun.com)
- [阿里云OSS官方论坛](http://bbs.aliyun.com)
- [阿里云OSS官方文档中心](http://www.aliyun.com/product/oss#Docs)

## 作者
- [Ting Zhang](https://github.com/dengwu12)

## License
- [MIT](https://github.com/aliyun/ossutil/blob/master/LICENSE)
