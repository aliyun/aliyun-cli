[English](./README.md) | 简体中文

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

阿里云命令行工具是开源项目，您可以从 [Github](https://github.com/aliyun/aliyun-cli) 上获取最新版本的 CLI。
您也可以在安装CLI前在Cloud Shell进行试用：

<a href="https://shell.aliyun.com/" target="cloudshell">
  <img src="https://img.alicdn.com/tfs/TB1wt1zq9zqK1RjSZFpXXakSXXa-1066-166.png" width="180" />
</a>

## 简介

阿里云命令行工具是用 Go 语言编写的, 基于阿里云 OpenAPI 打造的，用于管理阿里云资源的工具。通过下载和配置该工具，您可以在一个命令行方式下管理多个阿里云产品资源。

如果您在使用 CLI 的过程中遇到任何问题，欢迎前往[阿里云 CLI 问答社区](https://yq.aliyun.com/tags/type_ask-tagid_33502)提问。亦可在当前 GitHub提交 Issues。

**注意**：阿里云CLI使用OpenAPI方式访问云产品，确保您已经开通了要使用的云产品并了解该产品的OpenAPI的使用。您可以在[阿里云API平台](https://developer.aliyun.com/api)获取产品 API 文档，了解 API 的使用方式及参数列表。

## CLI Releases

CLI 版本更改说明请参考 [CHANGELOG](./CHANGELOG.md)

## 安装

- **下载安装包 (推荐)**  

  阿里云CLI工具下载、解压后即可使用，支持Mac, Linux, Windows平台(x64版本)。您可以将解压的`aliyun`可执行文件移至`/usr/local/bin`目录下，或添加到`$PATH`中。

  下载链接如下 (3.0.57)：

  - [Mac](https://aliyuncli.alicdn.com/aliyun-cli-macosx-3.0.57-amd64.tgz)
  - [Linux](https://aliyuncli.alicdn.com/aliyun-cli-linux-3.0.57-amd64.tgz)
  - [Windows (64 bit)](https://aliyuncli.alicdn.com/aliyun-cli-windows-3.0.57-amd64.zip)

- **使用brew**

  如果你的电脑上安装了 `brew`, 你可以使用它来安装阿里云命令行工具:

  ```sh
  brew install aliyun-cli
  ```

如果需要详细安装步骤或者编译安装步骤请访问官网文档 [安装 CLI](https://help.aliyun.com/document_detail/110343.html?spm=a2c4g.11186623.6.544.47ad1b18WHuF84)

## 配置

详细配置指引请访问官网 [配置 CLI](https://help.aliyun.com/document_detail/110341.html?spm=a2c4g.11186623.6.552.27f61b18o04a6s)

在使用阿里云CLI之前，您需要配置调用阿里云资源所需的凭证信息、地域、语言等。

你可以运行`aliyun configure`命令进行快速配置

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

### 其他认证方式

阿里云CLI，可通过在`configure`命令后增加`--mode <authenticationMethod>`参数的方式来使用不同的认证方式，目前支持的认证方式如下：

| 验证方式   | 说明                                  |
|------------|-------------------------------------|
| AK         | 使用AccessKey ID/Secret访问           |
| StsToken   | 使用STS Token访问                     |
| RamRoleArn | 使用RAM子账号的AssumeRole方式访问     |
| EcsRamRole | 在ECS实例上通过EcsRamRole实现免密验证 |

### 启用zsh/bash自动补全

- 使用`aliyun auto-completion`命令开启自动补全，目前支持zsh/bash
- 使用`aliyun auto-completion --uninstall`命令关闭自动补全

## 使用阿里云CLI

这里是基础使用指引，如需要详细使用手册，请访问 [这里](https://help.aliyun.com/document_detail/110344.html?spm=a2c4g.11186623.6.558.339122a6nSODBj)。

阿里云云产品的 OpenAPI 有 RPC 和 RESTful 两种风格，大部分产品使用的是RPC风格。不同风格的API的调用方法也不同。

您可以通过以下特点判断API风格：

- API参数中包含`Action`字段的是RPC风格，需要`PathPattern`参数的是Restful风格。
- 一般情况下，每个产品内，所有API的调用风格是统一的。
- 每个API仅支持特定的一种风格，传入错误的标识，可能会调用到其他API，或收到“ApiNotFound”的错误信息。

### 调用RPC风格的API

阿里云CLI中RPC风格的API调用的基本结构如下：

```sh
aliyun <product> <operation> [--parameter1 value1 --parameter2 value2 ...]
```

代码示例：

```sh
aliyun rds DescribeDBInstances --PageSize 50
aliyun ecs DescribeRegions
aliyun rds DescribeDBInstanceAttribute --DBInstanceId xxxxxx
```

### 调用RESTful风格的API

部分阿里云产品如容器服务的OpenAPI为Restful风格，调用Restful风格的接口与调用RPC风格的接口方式不同。参考以下代码示例，调用RESTful API。

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

阿里云CLI集成了一部分产品的API和参数列表信息, 您可以使用如下命令来获取帮助：

- `$ aliyun help`: 获取产品列表

- `$ aliyun help <product>`: 获取产品的API信息

  如获取ECS的API信息：`$ aliyun help ecs`

- `$ aliyun help <product> <apiName>`: 获取API的调用信息

  如获取ECS的CreateInstance的信息： `aliyun help ecs CreateInstance`

### 使用`--force`参数

阿里云CLI集成了一部分云产品的元数据，在调用时会对参数的合法性进行检查。如果使用了一个元数据中未包含的API或参数会导致`unknown api`或`unknown parameter`错误。可以使用`--force`参数跳过API和参数检查，强制调用元数据列表外的API和参数，如:

```sh
aliyun newproduct --version 2018-01-01 --endpoint newproduct.aliyuncs.com --param1 ... --force
```

在使用`--force`参数时，必须指定以下两个参数：

- `--version`: 指定API的版本，你可以在API文档中找到版本号，如ECS的版本号是`2014-05-26`。
- `--endpoint`: 指定产品的接入地址。请参考各产品的API文档。

#### 使用`--output`参数

阿里云产品的查询接口会返回json结构化数据，不方便阅读。例如：

```sh
aliyun ecs DescribeInstances
```

执行以上命令将得到以下json结果：

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

在使用`--waiter`参数时，必须指定以下两个子参数：

- `expr`: 通过 [jmespath](http://jmespath.org/) 查询语句来指定json结果中的被轮询字段。
- `to`: 被轮询字段的目标值。

可选子参数：

- `timeout`: 轮询的超时时间(秒)。
- `interval`: 轮询的间隔时间(秒)。

## 获取帮助

我们使用 GitHub issues 追踪用户反馈的 bug 和功能请求。请访问以下站点获取帮助：

- 基本使用方法请访问官网 [阿里云CLI](https://help.aliyun.com/document_detail/110244.html?spm=a2c4g.11174283.6.542.553a474fAUytL0)
- 在 [云栖社区](https://yq.aliyun.com/) 上提问并使用标签 [阿里云CLI](https://yq.aliyun.com/tags/type_ask-tagid_33502)
- 在 [Stack Overflow](https://stackoverflow.com/) 上提问并使用标签 [aliyun-cli](https://stackoverflow.com/questions/tagged/aliyun-cli)
- 如果您发现了一个 BUG 或是希望新增一个特性，请[提交 issue](https://github.com/aliyun/aliyun-cli/issues/new/choose)。
