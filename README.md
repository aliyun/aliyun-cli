English | [简体中文](./README-CN.md)

<h1 align="center">Alibaba Cloud CLI</h1>

<p align="center">
<a href="https://github.com/aliyun/aliyun-cli/actions/workflows/go.yml"><img src="https://github.com/aliyun/aliyun-cli/actions/workflows/go.yml/badge.svg" alt="Go build Status"></a>
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

## Troubleshoot
[Troubleshoot](https://troubleshoot.api.aliyun.com/?source=github_sdk) Provide OpenAPI diagnosis service to help developers locate quickly and provide solutions for developers through `RequestID` or `error message`.

## CLI Releases

The release notes for the CLI can be found in the [CHANGELOG](./CHANGELOG.md)

## Installation

- **Download installer (Recommended)**

  Download the installer, then extract the installer. You can move the extracted `aliyun` executable file to the `/usr/local/bin` directory or add it to the `$PATH`.

  Download link: (<img src="https://img.shields.io/github/release/aliyun/aliyun-cli.svg" alt="Latest Stable Version" />)

  - [Mac (AMD64)](https://aliyuncli.alicdn.com/aliyun-cli-macosx-latest-amd64.tgz)
  - [Mac (ARM64)](https://aliyuncli.alicdn.com/aliyun-cli-macosx-latest-arm64.tgz)
  - [Linux (AMD64)](https://aliyuncli.alicdn.com/aliyun-cli-linux-latest-amd64.tgz)
  - [Linux (ARM64)](https://aliyuncli.alicdn.com/aliyun-cli-linux-latest-arm64.tgz)
  - [Windows (64 bit)](https://aliyuncli.alicdn.com/aliyun-cli-windows-latest-amd64.zip)

  All releases please [click here](https://github.com/aliyun/aliyun-cli/releases).

- **Use brew**
If you have installed `brew` in your computer, you can use it to install Alibaba Cloud CLI as following:

```sh
brew install aliyun-cli
```

If you need detailed installation steps or compile the installation steps, please visit [Installation Guide](https://www.alibabacloud.com/help/zh/doc-detail/121988.html).

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
Default Language [zh]: zh
```

### Configure authentication methods

You can specify the authentication method to use by using the `configure` command with the `--mode <authenticationMethod>` option.

The following are supported authentication methods:

| Authentication methods | Description                                                  |
| ---------------------- | ------------------------------------------------------------ |
| AK                     | Use AccessKey ID and Secret to access Alibaba Cloud services |
| StsToken               | Use STS token to access Alibaba Cloud services               |
| RamRoleArn             | Use the AssumeRole to access Alibaba Cloud services          |
| EcsRamRole             | Use the EcsRamRole to access ECS resources                   |

### Use an external program to get credentials

You can use `--mode External` to specify to obtain credential data through an external program, and CLI will execute the program command and return it as a credential to initiate the call.

Agreement： 
1. The output location of the external program is standard output.
2. The output format is json string.
3. The output contains the key fields required by the CLI and credential fields

Key field:
- mode: Specify the type of credentials returned

Example of the return of each credential type:
- AK

```json
{
  "mode": "AK",
  "access_key_id": "accessKeyId",
  "access_key_secret": "accessKeySecret"
}
```

- StsToken

```json
{
  "mode": "StsToken",
  "access_key_id": "accessKeyId",
  "access_key_secret": "accessKeySecret",
  "sts_token": "stsToken"
}
```

- RamRoleArn

```json
{
  "mode": "RamRoleArn",
  "access_key_id": "accessKeyId",
  "access_key_secret": "accessKeySecret",
  "ram_role_arn": "ramRoleArn",
  "ram_session_name": "ramSessionName"
}
```

- EcsRamRole

```json
{
  "mode": "EcsRamRole",
  "ram_role_name": "ramRoleName"
}
```

#### Example:
```shell
$ aliyun configure --mode External --profile externalTest
Configuring profile 'externalTest' in 'External' authenticate mode...
Process Command []: <getCredential ak>
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json)
Default Language [zh|en] en: 
Saving profile[externalTest] ...Done.
```

### Use chainable RamRoleArn

You can use `--mode ChainableRamRoleArn` to combile a source profile and RAM role ARN flow. The following example
get intermediate credentials from source profile `cli-test`, then use it to call AssumeRole for get final credentials.

```json
{
  "profiles": [
    {
      "name": "chain",
      "mode": "ChainableRamRoleArn",
      "ram_role_arn": "acs:ram::<Account ID>:role/<Role Name>",
      "ram_session_name": "session",
      "source_profile": "cli-test"
    },
    {
      "name": "cli-test",
      "mode": "AK",
      "access_key_id": "<Access Key ID>",
      "access_key_secret": "<Access Key Secret>"
    }
  ]
}
```

### Use Credentials URI

You can use `--mode CredentialsURI` to get credentials from local/remote URI.

```json
{
  "profiles": [
    {
      "name": "uri",
      "mode": "CredentialsURI",
      "credentials_uri": "http://localhost:6666/?user=jacksontian"
    }
  ]
}
```

The Credentials URI must be response with status code 200, and following body:

```json
{
  "Code": "Success",
  "AccessKeyId": "<ak id>",
  "AccessKeySecret": "<ak secret>",
  "SecurityToken": "<security token>",
  "Expiration" "2006-01-02T15:04:05Z" // utc time
}
```

Otherwise, CLI treate as failure case.

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

## Support for environment variables

We supported following environment variables:

- `ALIBABACLOUD_PROFILE`/`ALIBABA_CLOUD_PROFILE`/`ALICLOUD_PROFILE`: When `--profile` flag is not specified, CLI use it.
- `ALIBABACLOUD_IGNORE_PROFILE=TRUE`: When this variable is specified, CLI ignores any configuration files.
- `ALIBABACLOUD_ACCESS_KEY_ID`/`ALICLOUD_ACCESS_KEY_ID`/`ACCESS_KEY_ID`: When no any specified Access Key Id, CLI use it.
- `ALIBABACLOUD_ACCESS_KEY_SECRET`/`ALICLOUD_ACCESS_KEY_SECRET`/`ACCESS_KEY_SECRET`: When no any specified Access Key Secret, CLI use it.
- `ALIBABACLOUD_SECURITY_TOKEN`/`ALIBABACLOUD_SECURITY_TOKEN`/`SECURITY_TOKEN`: When no any specified Security Token, CLI use it.
- `ALIBABACLOUD_REGION_ID`/`ALICLOUD_REGION_ID`/`REGION`: When no any specified Region Id, CLI use it.
- `DEBUG=sdk`：Through this variable, CLI display http request information. It helpful for troubleshooting.

## Getting Help

We use GitHub issues to track bugs and feature requests for user feedback. Please visit the following site for assistance:

- Please visit [Alibaba Cloud CLI](https://www.alibabacloud.com/help/doc-detail/110244.htm?spm=a2c63.p38356.b99.2.58e54573sCfIan) for the manual.
- Ask a question on [Stack Overflow](https://stackoverflow.com/) and tag it with [aliyun-cli](https://stackoverflow.com/questions/tagged/aliyun-cli)
- If you find a bug or want to add a feature, please [submit issue](https://github.com/aliyun/aliyun-cli/issues/new/choose).
