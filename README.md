English | [简体中文](./README-CN.md)

<h1 align="center">Alibaba Cloud CLI</h1>

<p align="center">
<a href="https://travis-ci.org/aliyun/aliyun-cli"><img src="https://travis-ci.org/aliyun/aliyun-cli.svg?branch=master" alt="Travis Build Status"></a>
<a href="https://ci.appveyor.com/project/aliyun/aliyun-cli"><img src="https://ci.appveyor.com/api/projects/status/avxoqqcmgksbt3d8/branch/master?svg=true" alt="Appveyor Build Status"></a>
<a href="https://codecov.io/gh/aliyun/aliyun-cli"><img src="https://codecov.io/gh/aliyun/aliyun-cli/branch/master/graph/badge.svg" alt="codecov"></a>
<a href="https://github.com/aliyun/aliyun-cli/blob/master/LICENSE"><img src="https://img.shields.io/github/license/aliyun/aliyun-cli.svg" alt="License"></a>
<a href="https://goreportcard.com/report/github.com/aliyun/aliyun-cli"><img src="https://goreportcard.com/badge/github.com/aliyun/aliyun-cli" alt="Go Report" ></a>
<br/>
<a href="https://github.com/aliyun/aliyun-cli/releases/latest"><img src="https://img.shields.io/github/release/aliyun/aliyun-cli.svg" alt="Latest Stable Version" ></a>
<a href="https://github.com/aliyun/aliyun-cli/releases/latest"><img src="https://img.shields.io/github/release-date/aliyun/aliyun-cli.svg" alt="GitHub Release Date"></a>
<br/>
<a href="https://github.com/aliyun/aliyun-cli/releases"><img src="https://img.shields.io/github/downloads/aliyun/aliyun-cli/total.svg" alt="GitHub All Releases" ></a>
<p>

The Alibaba Cloud CLI is an open source tool, you can get the latest version from [GitHub](https://github.com/aliyun/aliyun-cli).
You can also try it out in the Cloud Shell before installing the CLI.

<a href="https://shell.aliyun.com/" target="cloudshell">
  <img src="https://img.alicdn.com/tfs/TB1wt1zq9zqK1RjSZFpXXakSXXa-1066-166.png" width="180" />
</a>

## Introduction

The Alibaba Cloud CLI is a tool to manage and use Alibaba Cloud resources through a command line interface. It is written in Go and built on the top of Alibaba Cloud OpenAPI.

> **Note**: Alibaba Cloud CLI access the Alibaba Cloud services through OpenAPI. Before using Alibaba Cloud CLI, make sure that you have activated the service to use and known how to use OpenAPI.

## CLI Releases

The release notes for the CLI can be found in the [CHANGELOG](./CHANGELOG.md)

## Installation

- **Download installer (Recommended)**

  Download the installer, then extract the installer. You can move the extracted `aliyun` executable file to the `/usr/local/bin` directory or add it to the `$PATH`.

  Download link: (3.0.37)

  - [Mac](https://aliyuncli.alicdn.com/aliyun-cli-macosx-3.0.37-amd64.tgz)
  - [Linux](https://aliyuncli.alicdn.com/aliyun-cli-linux-3.0.37-amd64.tgz)
  - [Windows (64 bit)](https://aliyuncli.alicdn.com/aliyun-cli-windows-3.0.37-amd64.zip)

- **Use brew**
If you have installed `brew` in your computer, you can use it to install Alibaba Cloud CLI as following:

```sh
brew install aliyun-cli
```

If you need detailed installation steps or compile the installation steps, please visit [Installation Guide](https://www.alibabacloud.com/help/doc-detail/110343.htm?spm=a2c63.p38356.b99.4.56416f39SuRUr3).

## Configure

For detailed configuration instructions, please visit the official website [Configuration Alibaba Cloud CLI](https://www.alibabacloud.com/help/doc-detail/110341.htm?spm=a2c63.p38356.b99.12.77d468f5YJVFg1).

Before using Alibaba Cloud CLI to invoke the services, you need to configure the credential information, region, language, etc.

You can run the `aliyun configure` command for quick configuration.

```sh
$ aliyun configure
Configuring profile 'default' ...
Aliyun Access Key ID [None]: <Your AccessKey ID>
Aliyun Access Key Secret [None]: <Your AccessKey Secret>
Default Region Id [None]: cn-hangzhou
Default output format [json]: json
Default Languate [zh]: zh
```

### Configure authentication methods

You can specify the authentication method to use by using the `configure` command with the `--mode <authenticationMethod>` option.

The following are supported authentication methods:

| Authentication methods | Description                                                  |
|------------------------|--------------------------------------------------------------|
| AK                     | Use AccessKey ID and Secret to access Alibaba Cloud services |
| StsToken               | Use STS token to access Alibaba Cloud services               |
| RamRoleArn             | Use the AssumeRole to access Alibaba Cloud services          |
| EcsRamRole             | Use the EcsRamRole to access ECS resources                   |

### Enable bash/zsh auto completion

- Use `aliyun auto-completion` command to enable auto completion in zsh/bash
- Use `aliyun auto-completion --uninstall` command to disable auto completion.

## Use Alibaba Cloud CLI

Here is the basic usage guidelines. If you need a detailed manual, please visit [Use Alibaba Cloud CLI](https://www.alibabacloud.com/help/doc-detail/110344.htm?spm=a2c63.p38356.b99.18.ab77442ekAv3Yr)

The Alibaba Cloud OpenAPI has two styles, RPC style and RESTful style. Most of the Alibaba Cloud products use the RPC style. The way of calling an API varies depending on the API style.

You can distinguish the API style from the following characteristics:

- The API requiring the `Action` parameter is the RPC style, while the API requiring the `PathPattern` parameter is the RESTful style.
- In general, the API style for a product is consistent.
- Each API only supports one style. If an incorrect calling method is used, another API may be called or an error `ApiNotFound` is returned.

### Call RPC APIs

The following statement shows how to call RPC APIs in the Alibaba Cloud CLI:

```sh
aliyun <product> <operation> --parameter1 value1 --parameter2 value2 ...
```

Examples:

```sh
aliyun rds DescribeDBInstances --PageSize 50
```

```sh
aliyun ecs DescribeRegions
```

```sh
aliyun rds DescribeDBInstanceAttribute --DBInstanceId xxxxxx
```

### Call RESTful APIs

APIs of some products such as Container Service are RESTful style. The way to call RESTful APIs is different from RPC APIs.

The following examples show how to call RESTful APIs in the Alibaba Cloud CLI:

- GET request:

    ```sh
    aliyun cs GET /clusters
    ```

- POST request:

    ```sh
    aliyun cs POST /clusters --body "$(cat input.json)"
    ```

- DELETE request:

    ```sh
    aliyun cs DELETE /clusters/ce2cdc26227e09c864d0ca0b2d5671a07
    ```

### Get help information

Alibaba Cloud CLI integrates API descriptions for some products, you can get help by using the following commands:

- `aliyun help`: get product list

- `aliyun help <product>`: get the API information of a specific product

	For example, get help of ECS APIs: `$ aliyun help ecs`

- `$ aliyun help <product> <apiName>`: get the detailed API information of a specific APU

	For example, get the help information of the CreateInstance API: `aliyun help ecs CreateInstance`

### Use the `--force` option

Alibaba Cloud CLI integrates the product metadata of some products. It will validate API parameters when calling the API. If an API or a parameter that is not included in the metadata is used, an error `unknown api` or `unknown parameter` will be returned. You can use the `--force` option to skip the validation and call the API by force as shown in the following example:

```sh
aliyun newproduct --version 2018-01-01 --endpoint newproduct.aliyuncs.com --param1 ... --force
```

The following two options are required when using the `--force` option:

- `--version`: the API version. You can find the API version in the API documentation. For example, the ECS API version is `2014-05-26`.
- `--endpoint`: the product endpoint. Get the product endpoint in the corresponding API documentation.

### Special argument

When you input some argument like "-PortRange -1/-1", will cause parse error.In this case, you could assign value like this:
--PortRange=-1/-1.

## Getting Help

We use GitHub issues to track bugs and feature requests for user feedback. Please visit the following site for assistance:

- Please visit [Alibaba Cloud CLI](https://www.alibabacloud.com/help/doc-detail/110244.htm?spm=a2c63.p38356.b99.2.58e54573sCfIan) for the manual.

- Ask a question on [Stack Overflow](https://stackoverflow.com/) and tag it with [aliyun-cli](https://stackoverflow.com/questions/tagged/aliyun-cli)

- If you find a bug or want to add a feature, please [submit issue](https://github.com/aliyun/aliyun-cli/issues/new/choose).
