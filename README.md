English | [简体中文](./README-CN.md)

<h1 align="center">Alibaba Cloud CLI</h1>

<p align="center">
<a href="https://github.com/aliyun/aliyun-cli/actions/workflows/go.yml"><img src="https://github.com/aliyun/aliyun-cli/actions/workflows/go.yml/badge.svg" alt="Go build Status"></a>
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
  <img src="https://img.alicdn.com/tfs/TB1wt1zq9zqK1RjSZFpXXakSXXa-1066-166.png" width="180" alt="cloudshell" />
</a>

## Introduction

The Alibaba Cloud CLI is a tool to manage and use Alibaba Cloud resources through a command line interface. It is written in Go and built on the top of Alibaba Cloud OpenAPI.

> **Note**: Alibaba Cloud CLI access the Alibaba Cloud services through OpenAPI. Before using Alibaba Cloud CLI, make sure that you have activated the service to use and known how to use OpenAPI.

## Troubleshoot

[Troubleshoot](https://api.aliyun.com/troubleshoot?source=github_sdk) Provide OpenAPI diagnosis service to help developers locate quickly and provide solutions for developers through `RequestID` or `error message`.

## CLI Releases

The release notes for the CLI can be found in the [CHANGELOG](./CHANGELOG.md)

## Installation

- **Download installer (Recommended)**

  Download the installer, then extract the installer. You can move the extracted `aliyun` executable file to the `/usr/local/bin` directory or add it to the `$PATH`.

  Download link: (![Latest Stable Version](https://img.shields.io/github/release/aliyun/aliyun-cli.svg))

  - [Mac GUI Installer](https://aliyuncli.alicdn.com/aliyun-cli-latest.pkg)
  - [Mac Universal](https://aliyuncli.alicdn.com/aliyun-cli-macosx-latest-universal.tgz)
  - [Linux (AMD64)](https://aliyuncli.alicdn.com/aliyun-cli-linux-latest-amd64.tgz)
  - [Linux (ARM64)](https://aliyuncli.alicdn.com/aliyun-cli-linux-latest-arm64.tgz)
  - [Windows (64 bit)](https://aliyuncli.alicdn.com/aliyun-cli-windows-latest-amd64.zip)

  All releases please [click here](https://github.com/aliyun/aliyun-cli/releases).

- **Use brew**
If you have installed `brew` in your computer, you can use it to install Alibaba Cloud CLI as following:

```sh
brew install aliyun-cli
```

- **Use one-liner script**

You can paste the following command in a macOS Terminal or Linux shell prompt.

```sh
/bin/bash -c "$(curl -fsSL https://aliyuncli.alicdn.com/install.sh)"
```

If you need detailed installation steps or compile the installation steps, please visit [Installation Guide](https://www.alibabacloud.com/help/zh/doc-detail/121988.html).

## Configure

For detailed configuration instructions, please visit the official website [Configuration Alibaba Cloud CLI](https://www.alibabacloud.com/help/doc-detail/110341.htm).

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

| Authentication methods | Description                                                 |
|------------------------|-------------------------------------------------------------|
| AK                     | Use direct AccessKey ID/Secret as access credentials        |
| RamRoleArn             | Use RAM role assumption to provide access credentials       |
| EcsRamRole             | Use ECS instance role to provide access credentials         |
| OIDC                   | Use OIDC role assumption to provide access credentials      |
| External               | Use external processes to provide access credentials        |
| CredentialsURI         | Use external services to provide access credentials         |
| ChainableRamRoleArn    | Use chainable role assumption to provide access credentials |
| CloudSSO               | Use CloudSSO to provide access credentials                  |

If the --mode is not specified during configuration, the AK mode will be used by default.

### RAM Sub-account Role Assumption

You can specify obtaining credentials through RAM sub-account role assumption by using the --mode RamRoleArn. It works by exchanging temporary
credentials through the AssumeRole method. An example is as follows:

```shell
$ aliyun configure --mode RamRoleArn --profile subaccount
Configuring profile 'subaccount' in 'RamRoleArn' authenticate mode...
Access Key Id []: AccessKey ID
Access Key Secret []: AccessKey Secret
Sts Region []: cn-hangzhou
Ram Role Arn []: acs:ram::******:role/ecs-test
Role Session Name []: sessionname
Expired Seconds []: 900
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json)
Default Language [zh|en] en:
Saving profile[subaccount] ...Done.
```

### Use an external program to get credentials

You can use `--mode External` to specify to obtain credential data through an external program, and CLI will execute the program command and return it as a credential to initiate the call.

Agreement：

1. The output location of the external program is standard output.
2. The output format is json string.
3. The output contains the key fields required by the CLI and credential fields

Key field:

- mode: Specifies the type of credentials returned, currently supports two types of static credentials.

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

#### Example

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
  "Expiration": "2006-01-02T15:04:05Z" // utc time
}
```

Otherwise, CLI treat as failure case.

### Use OIDC to get credentials

You can use the `--mode OIDC` to obtain credentials through OIDC-based SSO role assumption. An example is as follows:

```shell
$ aliyun configure --mode OIDC --profile oidc_p
Configuring profile 'oidc_p' in 'OIDC' authenticate mode...
OIDC Provider ARN []: xxxx
OIDC Token File []: xxx
RAM Role ARN []: xxx
Role Session Name []: xxx
Default Region Id []: xxx
Default Output Format [json]: json (Only support json)
Default Language [zh|en] en: 
Saving profile[oidc_p] ...Done.
```


### Use OAuth to get credentials

You can use the `--mode OAuth` to obtain credentials through OAuth-based SSO role assumption. An example is as follows:

```shell
$ aliyun configure --mode OAuth --profile oauth_p
Configuring profile 'oauth-intl' in 'OAuth' authenticate mode...
OAuth Site Type (CN: 0 or INTL: 1, default: CN): 
1
Please open the following URL in your browser to authorize:
https://signin.alibabacloud.com/oauth2/v1/auth?response_type=code&client_id=xxxx
如果浏览器没有自动打开，请使用以下URL完成登录过程:

登录URL: https://signin.alibabacloud.com/oauth2/v1/auth?response_type=code&client_id=xxxx

现在您可以在浏览器中使用OAuth配置登录您的账户。
OAuth configuration completed. The temporary Access Key Id and Access Key Secret have been set in the profile.
Default Region Id []: 
```

### Use CloudSSO to get credentials

You can use the `--mode CloudSSO` to obtain credentials through CloudSSO. An example is as follows:

```shell
$ aliyun configure --mode CloudSSO --profile cloud_sso
Configuring profile 'cloudsso-test' in 'CloudSSO' authenticate mode...
CloudSSO Sign In Url [https://signin-cn-shanghai.alibabacloudsso.com/start/login]:
# CloudSSO Sign In Url is required, please input it.
# then follow the instructions to sign in.
```


### Enable bash/zsh auto-completion

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
aliyun ecs DescribeRegions
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

### Use the `--output` parameter

The query interface of Alibaba Cloud products will return json structured data, which is inconvenient to read. Example:

```sh
aliyun ecs DescribeInstances
```

Executing the above command will get the following json result:

```sh
{
  "PageNumber": 1,
  "TotalCount": 2,
  "PageSize": 10,
  "RequestId": "2B76ECBD-A296-407E-BE17-7E668A609DDA",
  "Instances": {
    "Instance": [
      {
        "ImageId": "ubuntu_16_0402_64_20G_alibase_20171227.vhd",
        "InstanceTypeFamily": "ecs.xn4",
        "VlanId": "",
        "InstanceId": "i-12345678912345678123",
        "Status": "Stopped",
        //omit some fields
      },
      Instance": [
      {
        "ImageId": "ubuntu_16_0402_64_20G_alibase_20171227.vhd",
        "InstanceTypeFamily": "ecs.xn4",
        "VlanId": "",
        "InstanceId": "i-abcdefghijklmnopqrst",
        "Status": "Running",
        //omit some fields
      },
    ]
  }
}
```

You can use the `--output` parameter to extract the fields of interest in the results and perform tabular output. Example:

```sh
aliyun ecs DescribeInstances --output cols=InstanceId,Status rows=Instances.Instance[]
```

Executing the above command will get the following result:

```sh
InstanceId             | Status
-----------------------|--------
i-12345678912345678123 | Stopped
i-abcdefghijklmnopqrst | Running
```

When using the `--output` parameter, the following sub-parameters must be specified:

- `cols`: The column names of the table, which need to correspond to the fields in the json data. For example, the InstanceId and Status fields in the result returned by the ECS DescribeInstances interface.

Optional sub-parameters:

- `rows`: Use the jmespath query statement to specify the data source of the table row in the json result.

### Use `--waiter` parameter

This parameter is used to poll the instance information until a specific state appears.

For example, after creating an instance using ECS, the instance will have a startup process. We will continue to query the running status of the instance until the status becomes "Running".

Example:

```sh
aliyun ecs DescribeInstances --InstanceIds '["i-12345678912345678123"]' --waiter expr='Instances.Instance[0].Status' to=Running
```

After executing the above command, the command line program will poll the instance status at a certain time interval and stop polling when the instance status becomes Running.

When using the `--waiter` parameter, you must specify the following two sub-parameters:

- `expr`: Specify the polled field in the json result through the jmespath query statement.
- `to`: The target value of the polled field.

Optional sub-parameters:

- `timeout`: polling timeout time (seconds).
- `interval`: polling interval (seconds).

### Special argument

When you input some argument like "-PortRange -1/-1", will cause parse error. In this case, you could assign value like this:
`--PortRange=-1/-1`.

## Supported environment variables

We support the following environment variables:

- `ALIBABA_CLOUD_PROFILE`: When the `--profile` flag is not specified, the CLI uses it.
- `ALIBABA_CLOUD_IGNORE_PROFILE=TRUE`: When this variable is specified, the CLI ignores any configuration files.
- `ALIBABA_CLOUD_ACCESS_KEY_ID`: When no Access Key Id is specified, the CLI uses it.
- `ALIBABA_CLOUD_ACCESS_KEY_SECRET`: When no Access Key Secret is specified, the CLI uses it.
- `ALIBABA_CLOUD_SECURITY_TOKEN`: When no Security Token is specified, the CLI uses it.
- `ALIBABA_CLOUD_REGION_ID`: When no Region Id is specified, the CLI uses it.
- `ALIBABA_CLOUD_SSO_CLIENT_ID`: Use this variable to override the client ID of the SSO application.
- `DEBUG=sdk`：Through this variable, the CLI can display HTTP request information, which is helpful for troubleshooting.

## Getting Help

We use GitHub issues to track bugs and feature requests for user feedback. Please visit the following site for assistance:

- Please visit [Alibaba Cloud CLI](https://www.alibabacloud.com/help/doc-detail/110244.htm?spm=a2c63.p38356.b99.2.58e54573sCfIan) for the manual.
- Ask a question on [Stack Overflow](https://stackoverflow.com/) and tag it with [aliyun-cli](https://stackoverflow.com/questions/tagged/aliyun-cli)
- If you find a bug or want to add a feature, please [submit issue](https://github.com/aliyun/aliyun-cli/issues/new/choose).
