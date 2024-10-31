[English](./README.md) | 简体中文

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

阿里云命令行工具是开源项目，您可以从 [Github](https://github.com/aliyun/aliyun-cli) 上获取最新版本的 CLI。
您也可以在安装 CLI 前在 Cloud Shell 进行试用：

<a href="https://shell.aliyun.com/" target="cloudshell">
  <img src="https://img.alicdn.com/tfs/TB1wt1zq9zqK1RjSZFpXXakSXXa-1066-166.png" width="180" alt="cloudshell" />
</a>

## 简介

阿里云命令行工具是用 Go 语言编写的, 基于阿里云 OpenAPI 打造的，用于管理阿里云资源的工具。通过下载和配置该工具，您可以在一个命令行方式下管理多个阿里云产品资源。

如果您在使用 CLI 的过程中遇到任何问题，请直接提交 Issues。

**注意**：阿里云 CLI 使用 OpenAPI 方式访问云产品，确保您已经开通了要使用的云产品并了解该产品的 OpenAPI 的使用。您可以在[阿里云 OpenAPI 开发者门户](https://api.aliyun.com/)查看产品 API 文档，了解 API 的使用方式及参数列表。

## 使用诊断

[Troubleshoot](https://api.aliyun.com/troubleshoot?source=github_sdk) 提供 OpenAPI 使用诊断服务，通过 `RequestID` 或 `报错信息` ，帮助开发者快速定位，为开发者提供解决方案。

## CLI Releases

CLI 版本更改说明请参考 [CHANGELOG](./CHANGELOG.md)

## 安装

- **下载安装包 (推荐)**  

  阿里云 CLI 工具下载、解压后即可使用，支持 Mac、Linux(amd64/arm64)、Windows 平台(x64版本)。您可以将解压的`aliyun` 可执行文件移至 `/usr/local/bin` 目录下，或添加到 `$PATH` 中。

  下载链接如下 (![Latest Stable Version](https://img.shields.io/github/release/aliyun/aliyun-cli.svg))：

  - [Mac 图形界面安装器](https://aliyuncli.alicdn.com/aliyun-cli-latest.pkg)
  - [Mac Universal](https://aliyuncli.alicdn.com/aliyun-cli-macosx-latest-universal.tgz)
  - [Linux (AMD64)](https://aliyuncli.alicdn.com/aliyun-cli-linux-latest-amd64.tgz)
  - [Linux (ARM64)](https://aliyuncli.alicdn.com/aliyun-cli-linux-latest-arm64.tgz)
  - [Windows (64 bit)](https://aliyuncli.alicdn.com/aliyun-cli-windows-latest-amd64.zip)

  点击查看[所有版本](https://github.com/aliyun/aliyun-cli/releases)。

- **使用 brew**

  如果你的电脑上安装了 `brew`, 你可以使用它来安装阿里云命令行工具:

  ```sh
  brew install aliyun-cli
  ```

- **使用一键安装脚本**

  你可以在 macOS 或 Linux 的命令行终端运行下面的命令：

  ```sh
  /bin/bash -c "$(curl -fsSL https://aliyuncli.alicdn.com/install.sh)"
  ```

如果需要详细安装步骤或者编译安装步骤请访问官网文档 [安装 CLI](https://help.aliyun.com/document_detail/121988.html)

## 配置

详细配置指引请访问官网 [配置 CLI](https://help.aliyun.com/document_detail/110341.html)

在使用阿里云 CLI 之前，您需要配置调用阿里云资源所需的凭证信息、地域、语言等。

你可以运行 `aliyun configure` 命令进行快速配置：

```sh
$ aliyun configure
Configuring profile 'default' in '' authenticate mode...
Access Key Id []: AccessKey ID
Access Key Secret []: AccessKey Secret
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json))
Default Language [zh|en] en:
Saving profile[akProfile] ...Done.
```

这将会以 AK 的认证模式对 default 进行凭证和其它配置。

### 所有凭证模式

可通过在 `configure` 命令后增加 `--mode <authenticationMethod>` 参数的方式来使用不同的凭证方式，目前支持的认证方式如下：

| 凭证模式              | 说明                                    |
|---------------------|-----------------------------------------|
| AK                  | 使用直接的 AccessKey ID/Secret 访问凭证    |
| RamRoleArn          | 使用 RAM 子账号角色扮演提供访问凭证          |
| EcsRamRole          | 使用 ECS 实例角色提供访问凭证               |
| OIDC                | 使用 OIDC 角色扮演的方式访问                |
| External            | 使用外部进程提供访问凭证                    |
| CredentialsURI      | 使用外部服务提供访问凭证                    |
| ChainableRamRoleArn | 使用链式角色扮演的方式提供访问凭证            |

如果在配置时不传递 `--mode`，将默认使用 AK 模式。

### RAM 子账号角色扮演

您可以使用 `--mode RamRoleArn` 指定通过 RAM 子账号进行角色扮演来获取凭证。它的底层是通过 AssumeRole 方法来换取
临时凭证。示例如下：

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

### 使用外部程序获取凭证

您可以使用 `--mode External` 指定通过外部程序获取凭证数据，CLI 将会以执行该程序命令并将其返回作为凭证来使用。

约定：

1. 外部程序输出位置为标准输出
2. 输出为json结构字符串
3. 输出包含关键字段以及凭证字段

关键字段:

- mode: 指定返回凭证类型，目前支持两种静态的凭证。

各凭证返回结构示例:

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

#### 示例

```shell
$ aliyun configure --mode External --profile externalTest
Configuring profile 'externalTest' in 'External' authenticate mode...
Process Command []: <getCredential ak>
Default Region Id []: cn-hangzhou
Default Output Format [json]: json (Only support json)
Default Language [zh|en] en: 
Saving profile[externalTest] ...Done.
```

### 使用链式 RamRoleArn

通过 ChainableRamRoleArn ，可以指定一个前置凭证配置，然后再进行角色扮演。前置凭证的设置会更灵活，它可以是子账号的 AK，也可以是通过其它方式换取的 STS，比如 EcsRamRole。

你可以使用 `--mode ChainableRamRoleArn` 来组合源配置和 RamRoleARN 的角色扮演流程。下面的例子从源配置中获取中间凭证，再基于中间凭证完成角色扮演，获取最终的凭证。

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

### 使用 Credentials URI

你可以通过 `--mode CredentialsURI` 来从一个本地或远程的 URI 地址实现 Credentials 的获取。

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

这个 Credentials URI 必须相应 200 和如下的结构：

```json
{
  "Code": "Success",
  "AccessKeyId": "<ak id>",
  "AccessKeySecret": "<ak secret>",
  "SecurityToken": "<security token>",
  "Expiration": "2006-01-02T15:04:05Z" // utc time
}
```

其他情况，CLI 会当作失败案例处理。

### 使用 OIDC 获取凭证

你可以通过 `--mode OIDC` 来使用基于 OIDC 的 SSO 角色扮演获取凭证。示例如下：

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

### 启用 zsh/bash 自动补全

- 使用 `aliyun auto-completion` 命令开启自动补全，目前支持 zsh/bash
- 使用 `aliyun auto-completion --uninstall` 命令关闭自动补全

## 使用阿里云 CLI

这里是基础使用指引，如需要详细使用手册，请访问 [这里](https://help.aliyun.com/document_detail/110344.html)。

阿里云云产品的 OpenAPI 有 RPC 和 RESTful 两种风格，大部分产品使用的是 RPC 风格。不同风格的 API 的调用方法也不同。

您可以通过以下特点判断API风格：

- API 参数中包含 `Action` 字段的是RPC风格，需要 `PathPattern` 参数的是 Restful 风格。
- 一般情况下，每个产品内，所有 API 的调用风格是统一的。
- 每个 API 仅支持特定的一种风格，传入错误的标识，可能会调用到其他 API，或收到“ApiNotFound”的错误信息。

### 调用 RPC 风格的 API

阿里云 CLI 中 RPC 风格的 API 调用的基本结构如下：

```sh
aliyun <product> <operation> [--parameter1 value1 --parameter2 value2 ...]
```

代码示例：

```sh
aliyun rds DescribeDBInstances --PageSize 50
aliyun ecs DescribeRegions
aliyun rds DescribeDBInstanceAttribute --DBInstanceId xxxxxx
```

### 调用 RESTful 风格的 API

部分阿里云产品如容器服务的 OpenAPI 为 Restful 风格，调用 Restful 风格的接口与调用 RPC 风格的接口方式不同。参考以下代码示例，调用 RESTful API。

- GET请求示例：

    ```sh
    aliyun cs GET /clusters
    ```

- POST请求示例：

    ```sh
    aliyun cs POST /clusters --body "$(cat input.json)"
    ```

- DELETE请求示例：

    ```sh
    aliyun cs DELETE /clusters/ce2cdc26227e09c864d0ca0b2d5671a07
   ```

### 获取帮助信息

阿里云 CLI 集成了一部分产品的 API 和参数列表信息, 您可以使用如下命令来获取帮助：

- `$ aliyun help`: 获取产品列表

- `$ aliyun help <product>`: 获取产品的API信息

  如获取 ECS 的 API 信息：`$ aliyun help ecs`

- `$ aliyun help <product> <apiName>`: 获取 API 的调用信息

  如获取 ECS 的 CreateInstance 的信息： `aliyun help ecs CreateInstance`

### 使用`--force`参数

阿里云 CLI 集成了一部分云产品的元数据，在调用时会对参数的合法性进行检查。如果使用了一个元数据中未包含的API或参数会导致`unknown api`或`unknown parameter`错误。可以使用`--force`参数跳过API和参数检查，强制调用元数据列表外的API和参数，如:

```sh
aliyun newproduct --version 2018-01-01 --endpoint newproduct.aliyuncs.com --param1 ... --force
```

在使用`--force`参数时，必须指定以下两个参数：

- `--version`: 指定API的版本，你可以在API文档中找到版本号，如ECS的版本号是`2014-05-26`。
- `--endpoint`: 指定产品的接入地址。请参考各产品的API文档。

#### 使用`--output`参数

阿里云产品的查询接口会返回 JSON 结构化数据，不方便阅读。例如：

```sh
aliyun ecs DescribeInstances
```

执行以上命令将得到以下 JSON 结果：

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

可以使用`--output`参数提取结果中感兴趣的字段，并进行表格化输出。例如：

```sh
aliyun ecs DescribeInstances --output cols=InstanceId,Status rows=Instances.Instance[]
```

执行以上命令将得到以下形式的结果：

```sh
InstanceId             | Status
-----------------------|--------
i-12345678912345678123 | Stopped
i-abcdefghijklmnopqrst | Running
```

在使用`--output`参数时，必须指定以下子参数：

- `cols`: 表格的列名，需要与json数据中的字段相对应。如ECS DescribeInstances 接口返回结果中的字段`InstanceId` 以及 `Status`。

可选子参数：

- `rows`: 通过 [jmespath](http://jmespath.org/) 查询语句来指定表格行在json结果中的数据来源。
  
### 使用`--waiter`参数

该参数用于轮询实例信息直到出现特定状态。

例如使用ECS创建实例后，实例会有启动的过程。我们会不断的查询实例的运行状态，直到状态变为"Running"。

例如：

```sh
aliyun ecs DescribeInstances --InstanceIds '["i-12345678912345678123"]' --waiter expr='Instances.Instance[0].Status' to=Running
```

执行以上命令后,命令行程序将以一定时间间隔进行实例状态轮询，并在实例状态变为`Running`时停止轮询。

在使用 `--waiter` 参数时，必须指定以下两个子参数：

- `expr`: 通过 [jmespath](http://jmespath.org/) 查询语句来指定json结果中的被轮询字段。
- `to`: 被轮询字段的目标值。

可选子参数：

- `timeout`: 轮询的超时时间(秒)。
- `interval`: 轮询的间隔时间(秒)。

## 环境变量支持

我们支持下面的环境变量：

- `ALIBABA_CLOUD_PROFILE`： 当 `--profile` 没有指定，CLI 将使用该环境变量。
- `ALIBABA_CLOUD_IGNORE_PROFILE=TRUE`: 当这个变量被指定，CLI 忽略配置文件。
- `ALIBABA_CLOUD_ACCESS_KEY_ID`： 当没有任何 Access Key Id 的指定，CLI 将使用该环境变量。
- `ALIBABA_CLOUD_ACCESS_KEY_SECRET`： 当没有任何 Access Key Secret 的指定，CLI 将使用该环境变量。
- `ALIBABA_CLOUD_SECURITY_TOKEN`： 当没有任何 Security Token 的指定，CLI 将使用该环境变量。
- `ALIBABA_CLOUD_REGION_ID`： 当没有任何 RegionId 的指定，CLI 将使用该环境变量。
- `DEBUG=sdk`：通过该环境变量，CLI 将打印 HTTP 请求信息。这对于排查故障非常有用。

## 获取帮助

我们使用 GitHub issues 追踪用户反馈的 bug 和功能请求。请访问以下站点获取帮助：

- 基本使用方法请访问官网 [阿里云 CLI](https://help.aliyun.com/document_detail/110244.html)
- 在 [Stack Overflow](https://stackoverflow.com/) 上提问并使用标签 [aliyun-cli](https://stackoverflow.com/questions/tagged/aliyun-cli)
- 如果您发现了一个 BUG 或是希望新增一个特性，请[提交 issue](https://github.com/aliyun/aliyun-cli/issues/new/choose)。
