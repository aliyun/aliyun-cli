# Alibaba Cloud OSSUTIL
 
[![GitHub Version](https://badge.fury.io/gh/aliyun%2Fossutil.svg)](https://badge.fury.io/gh/aliyun%2Fossutil)
[![Build Status](https://travis-ci.org/aliyun/ossutil.svg?branch=master)](https://travis-ci.org/aliyun/ossutil)
[![Coverage Status](https://coveralls.io/repos/github/aliyun/ossutil/badge.svg?branch=master)](https://coveralls.io/github/aliyun/ossutil?branch=master)

### [README of Chinese](https://github.com/aliyun/ossutil/blob/master/README-CN.md)

## About
- This tool is developed with Go and built on the official GO SDK of OSS [Alibaba Cloud Object Storage Service](http://www.aliyun.com/product/oss/).
- OSS is a cloud storage service provided by Alibaba Cloud, featuring massive capacity, security, low cost, and high reliability.
- OSS can store any type of files. It applies to various websites, development enterprises and developers.
- This tool aims to provide a convenient-to-use command line for users to manage data in OSS.
- The current version provides to list and delete multipart upload tasks.
- The current version does not support bucket management. The feature will be available in future versions.

## Version
- Current version: v1.7.19

## Run environment
- Linux
- Windows
- Mac

## Dependent libraries 
- goopt (github.com/droundy/goopt) 
- configparser (github.com/alyu/configparser)
- leveldb (github.com/syndtr/goleveldb/leveldb)
- oss (github.com/aliyun/aliyun-oss-go-sdk/oss)
- gopkg.in/check.v1 (gopkg.in/check.v1)

## Quick use
#### Get the command list
```go
    ./ossutil
    or ./ossutil help
```

#### View the help documentation for a command
```go
    ./ossutil help cmd 
```
    
#### Configure OSSUTIL 
```go
    ./ossutil config
```

#### List buckets
```go
    ./ossutil ls
    or ./ossutil ls oss://
```

#### List objects and multipart upload tasks
```go
    ./ossutil ls -a
    or ./ossutil ls oss:// -a
```

#### Upload a file
```go
    ./ossutil cp localfile oss://bucket
```

#### Others
You can use `./ossutil help cmd` to view the help documentation for the command you want to use. 

## Notes
### Run OSSUTIL
- First, configure your Go project directory. 
- Use `go get` to get the library that ossutil depends on.
- Run `go get github.com/aliyun/ossutil`.
- Enter the *src* directory under the Go project directory and build to generate the OSSUTIL tool. For example, on Linux, you can run `go build github.com/aliyun/ossutil/ossutil.go`.
- Refer to the example above to run the OSSUTIL tool.

### Test
- Enter the *src* directory under the Go project directory and modify the endpoint, AccessKeyId, AccessKeySecret and STSToken configuration items in the *github.com/aliyun/ossutil/lib/command_test.go*.
- Run `go test` under the *lib* directory.

## Contact us
- [Alibaba Cloud OSS official website](http://oss.aliyun.com).
- [Alibaba Cloud OSS official forum](http://bbs.aliyun.com).
- [Alibaba Cloud OSS official documentation center](http://www.aliyun.com/product/oss#Docs).

## Author
- [Ting Zhang](https://github.com/dengwu12)

## License
- [MIT](https://github.com/aliyun/ossutil/blob/master/LICENSE)
