# Aliyun Command Line Interface

[中文文档](./README_cn.md)

This is a refactoring **BETA** version rewrite with golang, for old stable python verison switch to [python branch](https://github.com/aliyun/aliyun-cli/tree/python_final)

## Overview

Aliyun Command Line Interface `aliyun` is a unified tool to manage your Aliyun services. Using this tool you can easily invoke the Aliyun OpenAPI to control multiple Aliyun services from the command line and also automate them through scripts, for instance using the Bash shell or Python.

## Install

### Download

You can download binary release (0.31) in the following links:

- [Mac](http://aliyun-cli.oss-cn-hangzhou.aliyuncs.com/aliyun-cli-macosx-0.33-amd64.tgz)
- [Linux](http://aliyun-cli.oss-cn-hangzhou.aliyuncs.com/aliyun-cli-linux-0.33-amd64.tgz)
- [Windows](http://aliyun-cli.oss-cn-hangzhou.aliyuncs.com/aliyun-cli-windows-0.33-amd64.tgz)

Unzip and use. You can move `aliyun` to `/usr/local/bin` or add to `$PATH` to quick access.

or

### Clone source and build

- When platform not suppoted or other requirement, you can clone code and make it by yourself. Install and configure [golang](golang.org) first.

```
$ mkdir -p $GOPATH/src/github.com/aliyun
$ cd $GOPATH/src/github.com/aliyun
$ git clone http://github.com/aliyun/aliyun-cli.git
$ git clone http://github.com/aliyun/aliyun-openapi-meta.git
$ cd aliyun-cli
$ make install
```

## Configure

- Before using `aliyun`, you should run `aliuyun configure` first
- Configure need a vaild AccessKeyId/Secret, you can create a AccessKeyId/Secret from [AccessKey Console](https://ak-console.aliyun.com/#/accesskey), or contact your Admin.
- CLI use OpenAPI to access cloud product, you need provisioning it first from console.

## Basic Configure

```
$ aliyun configure
Configuring profile 'default' ...
Aliyun Access Key ID [None]: <Your aliyun access key id>
Aliyun Access Key Secret [None]: <Your aliyun access key secret>
Default Region Id [None]: cn-hangzhou
Default output format [json]: table
Default Language [zh]: en
```

### Multi-User Configure

- `aliyun` support multi-user configure, Use `$ aliyun configure --profile user1` you can configure profile of assigned user name
- Use `$ aliyun configure list`, you can list all configured profiles.
- You can add flag `--profile user1` to use assigned profile, in invoke commands.

```
Profile   | CertificationMode | Valid   | AccessKeyId
--------- | ----------------- | ------- | ----------------
default * | AK                | Valid   | *************ac8
ram       | EcsRamRole        | Invalid |
test      | StsToken          | Invalid | **
leo       | AK                | Valid   | *************bb2
```

### Other Certification Mode

- When use `aliyun configure` you can add flag `--mode ...` to specific certification mode

| Certificated Mode  | Description |
| --------       | -------- |
| AK             | Use AccessKeyId/Secret to access  |
| StsToken       | Use StsToken to access   |
| RamRoleArn     | Use RAM and AssumeRole method to access     |
| EcsRamRole     | Use EcsRamRole in ECS instance to access without key |
| RsaKeyPair     | Use Rsa Key Pair (Only supprted in Japen Site)     |

### Use Auto Completion

- TODO

## Usage

### Basic Usage

Usage:

```
$ aliyun <product> <operation> --parameter1 value1 --parameter2 value2 ...
```

Samples:

```
$ aliyun rds DescribeDBInstances --PageSize 50
$ aliyun ecs DescribeRegions
$ aliyun rds DescribeDBInstanceAttribute --DBInstanceId xxxxxx
```

### Help Message

`aliyun` integrated a part of alibaba cloud product's meta data, include product, api, parameter. Use the following command to get help information:

- `$ aliyun help`: print product list
- `$ aliyun help ecs`: print product info
- `$ aliyuh help ecs CreateInstance`: print api info

### Restful Invoke

A part of OpenAPI is Restful style，invoke Restful style is defferent with RPC style.

- Sample(GET):

```
$ aliyun cs GET /clusters
```

- Sample(POST):

```
$ aliyun cs POST /clusters --body "$(cat input.json)"
```

- Sample(DELETE)

```
$ aliyun cs DELETE /clusters/ce2cdc26227e09c864d0ca0b2d5671a07
```

How to regconize openapi style is RPC or Restful, when reading OpenAPI product doc.

- References to API parameters，`Action` indicate RPC style, `PathPattern` indicates Restful style。
- Generally, APIs of one product have accordance style.
- Every API only support one style, When call with error parameters, you may invoke to error API or receive `ApiNotFound` error message.

### Use `--force` Flag

`aliyun` CLI integrated a part of products meta data. Before invoke CLI will check api and parameters, use a unknown api or parameters will cause `unknown api` or `unknown parameter` error. If you are using API or parameters not integrated in our meta, add `--force` flag to skip meta check.

```
$ aliyun newproduct --version 2018-01-01 --endpoint newproduct.aliyuncs.com --param1 ... --force
```

Please read Alibaba Cloud OpenAPI document, find api style and parameters. There are two flags is important

- `--version`: assign OpenAPI version, you can find this Version from OpenAPI document, for example: Ecs's version is 2014-05-26
- `--endpoint`: assign product domain, you can find Domain from OpenAPI document, in general the endpoint may be `product.aliyuncs.com` or `product.en-central-1.aliyuncs.com`


### Additional Flags

- `--profile`  use configured profile
- `--force`    call OpenAPI without check
- `--header`   add custom HTTP header with --header x-foo=bar
- `--endpoint` use assigned endpoint
- `--region`   use assigned region
- `--version`  assign product version



