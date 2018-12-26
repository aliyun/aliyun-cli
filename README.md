# Aliyun Command Line Interface
[![Build Status](https://travis-ci.org/aliyun/aliyun-cli.svg?branch=master)](https://travis-ci.org/aliyun/aliyun-cli)

[中文文档](./README_zh.md)

The Alibaba Cloud CLI is an open source tool, you can get the latest version from [GitHub](https://github.com/aliyun/aliyun-cli).

This version is a new version built on Go. If you want to use the old version (built on Alibaba Cloud Python SDK), click [here](https://github.com/aliyun/aliyun-cli/tree/python_final).

<a href="https://shell.aliyun.com/" target="cloudshell">
  <img src="https://img.alicdn.com/tfs/TB1wt1zq9zqK1RjSZFpXXakSXXa-1066-166.png" width="180" />
</a>

## Introduction

The Alibaba Cloud CLI is a tool to manage and use Alibaba Cloud resources through a command line interface. It is written in Go and built on the top of Alibaba Cloud OpenAPI.

If you encounter an issue when using Alibaba Cloud CLI, please submit your issue through [GitHub Issue](https://github.com/aliyun/aliyun-cli/issues/new) or joining the official SDK & CLI customer service group (DingDing Group No. 11771185).

> **Note**: Alibaba Cloud CLI access the Alibaba Cloud services through OpenAPI. Before using Alibaba Cloud CLI, make sure that you have activated the service to use and known how to use OpenAPI.

## Install Alibaba Cloud CLI

You can install Alibaba Cloud CLI either through the installer or the source code:

- **Download installer (Recommended)**

	Download the installer, then extract the installer. You can move the extracted `aliyun` executable file to the `/usr/local/bin` directory or add it to the `$PATH`.

	Download link: (3.0.7)

	- [Mac](https://aliyuncli.alicdn.com/aliyun-cli-macosx-3.0.7-amd64.tgz)
	- [Linux](https://aliyuncli.alicdn.com/aliyun-cli-linux-3.0.7-amd64.tgz)
	- [Windows (64 bit)](https://aliyuncli.alicdn.com/aliyun-cli-windows-3.0.7-amd64.zip)

- **Compile source code**

	If you can access to [golang.org](https://golang.org/), and have configured Golang(go1.10.1), run the following command to install the CLI:

	```
	$ mkdir -p $GOPATH/src/github.com/aliyun
	$ cd $GOPATH/src/github.com/aliyun
	$ git clone http://github.com/aliyun/aliyun-cli.git
	$ git clone http://github.com/aliyun/aliyun-openapi-meta.git
	$ cd aliyun-cli
	$ make install
	```

## Configure Alibaba Cloud CLI

Before using the CLI, you must complete the basic configurations.

### Basic configurations

Before using the CLI, you must run the `aliyun configure` command to complete the CLI configuration. An Alibaba Cloud account and a pair of AccessKey ID and AccessKey Secret are required. You can get the AccessKey on the [AccessKey](https://ak-console.aliyun.com/#/accesskey) page or get it from your system administrator.

A default profile is created with information provided.

```
$ aliyun configure
Configuring profile 'default' ...
Aliyun Access Key ID [None]: <Your AccessKey ID>
Aliyun Access Key Secret [None]: <Your AccessKey Secret>
Default Region Id [None]: cn-hangzhou
Default output format [json]: json
Default Languate [zh]: zh
```

### Configure multiple user profiles

Alibaba Cloud CLI supports configuring multiple user profiles. You can specify which user profile is used to call the API by using the `aliyun configure` command with the `--profile` option as shown in the following example:

`$ aliyun configure --profile user1`


Run the `aliyun configure list` command to view the configured user profiles. The profile followed by an asterisk (\*) is the profile current in use.

```
Profile   | Credential         | Valid   | Region           | Language
--------- | ------------------ | ------- | ---------------- | --------
default * | AK:***f9b          | Valid   | cn-beijing       | zh
aaa       | AK:******          | Invalid |                  |
test      | AK:***456          | Valid   |                  | en
ecs       | EcsRamRole:EcsTest | Valid   | cn-beijing       | en
```

### Configure authentication methods

You can specify the authentication method to use by using the `configure` command with the `--mode <authenticationMethod>` option.

The following are supported authentication methods:

| Authentication methods  | Description |
| --------       | -------- |
| AK             | Use AccessKey ID and Secret to access Alibaba Cloud services |
| StsToken       | Use STS token to access Alibaba Cloud services    |
| RamRoleArn     | Use the AssumeRole to access Alibaba Cloud services    |
| EcsRamRole     | Use the EcsRamRole to access ECS resources   |


### Enable bash/zsh auto completion

- Use `aliyun auto-completion` command to enable auto completion in zsh/bash
- Use `aliyun auto-completion --uninstall` command to disable auto completion.

## Use Alibaba Cloud CLI

The Alibaba Cloud OpenAPI has two styles, RPC style and RESTful style. Most of the Alibaba Cloud products use the RPC style. The way of calling an API varies depending on the API style.

You can distinguish the API style from the following characteristics:

- The API requiring the `Action` parameter is the RPC style, while the API requiring the `PathPattern` parameter is the RESTful style.
- In general, the API style for a product is consistent.
- Each API only supports one style. If an incorrect calling method is used, another API may be called or an error `ApiNotFound` is returned.

### Call RPC APIs

The following statement shows how to call RPC APIs in the Alibaba Cloud CLI:

```
$ aliyun <product> <operation> --parameter1 value1 --parameter2 value2 ...
```

Examples:

```
$ aliyun rds DescribeDBInstances --PageSize 50
$ aliyun ecs DescribeRegions
$ aliyun rds DescribeDBInstanceAttribute --DBInstanceId xxxxxx
```


### Call RESTful APIs

APIs of some products such as Container Service are RESTful style. The way to call RESTful APIs is different from RPC APIs.

The following examples show how to call RESTful APIs in the Alibaba Cloud CLI:

- GET request:

	```
	$ aliyun cs GET /clusters
	```

- POST request:

	```
	$ aliyun cs POST /clusters --body "$(cat input.json)"
	```

- DELETE request:

	```
	$ aliyun cs DELETE /clusters/ce2cdc26227e09c864d0ca0b2d5671a07
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

```
$ aliyun newproduct --version 2018-01-01 --endpoint newproduct.aliyuncs.com --param1 ... --force
```

The following two options are required when using the `--force` option:

- `--version`: the API version. You can find the API version in the API documentation. For example, the ECS API version is `2014-05-26`.
- `--endpoint`: the product endpoint. Most of the product endpoints are in the format of `product.aliyuncs.com`, while some product endpoints are `product.en-central-1.aliyuncs.com`. Get the product endpoint in the corresponding API documentation.

### Special argument:

When you input some argument like "-PortRange -1/-1", will cause parse error.In this case, you could assign value like this:
--PortRange=-1/-1.
