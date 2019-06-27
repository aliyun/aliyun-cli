[English](./README.md) | 简体中文

<p align="center">
<a href=" https://www.alibabacloud.com"><img src="https://aliyunsdk-pages.alicdn.com/icons/Aliyun.svg"></a>
</p>

<h1 align="center">Alibaba Cloud CLI</h1>

<p align="center">
<a href="https://travis-ci.org/aliyun/aliyun-cli"><img src="https://travis-ci.org/aliyun/aliyun-cli.svg?branch=master" alt="Travis Build Status"></a>
<a href="https://codecov.io/gh/aliyun/aliyun-cli"><img src="https://codecov.io/gh/aliyun/aliyun-cli/branch/master/graph/badge.svg" alt="codecov"></a>
<a href="https://github.com/aliyun/aliyun-cli/blob/master/LICENSE"><img src="https://img.shields.io/github/license/aliyun/aliyun-cli.svg" alt="License"></a>
<a href="https://goreportcard.com/report/github.com/aliyun/aliyun-cli"><img src="https://goreportcard.com/badge/github.com/aliyun/aliyun-cli" alt="Go Report" ></a>
<br/>
<a href="https://github.com/aliyun/aliyun-cli/releases/latest"><img src="https://img.shields.io/github/release/aliyun/aliyun-cli.svg" alt="Latest Stable Version" ></a>
<a href="https://github.com/aliyun/aliyun-cli/releases/latest"><img src="https://img.shields.io/github/release-date/aliyun/aliyun-cli.svg" alt="GitHub Release Date"></a>
<br/>
<a href="https://github.com/aliyun/aliyun-cli/releases"><img src="https://img.shields.io/github/downloads/aliyun/aliyun-cli/total.svg" alt="GitHub All Releases" ></a>
<p>

阿里云命令行工具是开源项目，您可以从[Github](https://github.com/aliyun/aliyun-cli)上获取最新版本的CLI。

<a href="https://shell.aliyun.com/" target="cloudshell">
  <img src="https://img.alicdn.com/tfs/TB1wt1zq9zqK1RjSZFpXXakSXXa-1066-166.png" width="180" />
</a>

## 在线示例
**[API Explorer](https://api.aliyun.com)** 提供在线调用云产品 OpenAPI、并动态生成 SDK Example 代码和快速检索接口等能力，能显著降低使用云 API 的难度，强烈推荐使用

<a href="https://api.aliyun.com" target="api_explorer">
  <img src="https://img.alicdn.com/tfs/TB12GX6zW6qK1RjSZFmXXX0PFXa-744-122.png" width="180" />
</a>

## 简介

阿里云命令行工具是用Go语言编写的, 基于阿里云OpenAPI打造的，用于管理阿里云资源的工具。通过下载和配置该工具，您可以在一个命令行方式下使用多个阿里云产品。

如果您在使用SDK的过程中遇到任何问题，欢迎前往[阿里云SDK问答社区](https://yq.aliyun.com/tags/type_ask-tagid_23350)提问，提问前请阅读[提问引导](https://help.aliyun.com/document_detail/93957.html)。亦可在当前GitHub提交Issues。

**注意**：阿里云CLI使用OpenAPI方式访问云产品，确保您已经开通了要使用的云产品并了解该产品的OpenAPI的使用。您可以在[阿里云API平台](https://developer.aliyun.com/api)获取产品API文档，了解API的使用方式及参数列表。


## 安装

您可以通过下载安装包或者直接编译源码的方式安装阿里云CLI：

- **下载安装包 (推荐)**

	阿里云CLI工具下载、解压后即可使用，支持Mac, Linux, Windows平台(x64版本)。	您可以将解压的`aliyun`可执行文件移至`/usr/local/bin`目录下，或添加到`$PATH`中。

	下载链接如下 (3.0.18)：

	- [Mac](https://aliyuncli.alicdn.com/aliyun-cli-macosx-3.0.18-amd64.tgz)
	- [Linux](https://aliyuncli.alicdn.com/aliyun-cli-linux-3.0.18-amd64.tgz)
	- [Windows (64 bit)](https://aliyuncli.alicdn.com/aliyun-cli-windows-3.0.18-amd64.zip)

- **编译源码**

	如果您能访问[golang.org](https://golang.org/), 并安装配置好Golang环境(go1.10.1)，请按照如下步骤下载源码并编译。

	```
	$ mkdir -p $GOPATH/src/github.com/aliyun
	$ cd $GOPATH/src/github.com/aliyun
	$ git clone http://github.com/aliyun/aliyun-cli.git
	$ git clone http://github.com/aliyun/aliyun-openapi-meta.git
	$ cd aliyun-cli
	$ make install
	```

- **使用brew**

   如果你的电脑上安装了 `brew`, 你可以使用它来安装阿里云命令行工具:
   
   ```
   $ brew install aliyun-cli
   ```

## 配置

在使用阿里云CLI前，您需要运行`aliyun configure`命令进行配置。在配置阿里云CLI时，您需要提供阿里云账号以及一对AccessKeyId和AccessKeySecret。

您可以在阿里云控制台的[AccessKey页面](https://ak-console.aliyun.com/#/accesskey)创建和查看您的AccessKey，或者联系您的系统管理员获取AccessKey。

#### 基本配置

```
$ aliyun configure
Configuring profile 'default' ...
Aliyun Access Key ID [None]: <Your AccessKey ID>
Aliyun Access Key Secret [None]: <Your AccessKey Secret>
Default Region Id [None]: cn-hangzhou
Default output format [json]: json
Default Languate [zh]: zh
```

#### 多用户配置

阿里云CLI支持多用户配置。您可以使用`$ aliyun configure --profile user1`命令指定使用哪个账号调用OpenAPI。

执行`$ aliyun configure list`命令可以查看当前的用户配置, 如下表。 其中在Profile后面有星号（*）标志的为当前使用的默认用户配置。

```
Profile   | Credential         | Valid   | Region           | Language
--------- | ------------------ | ------- | ---------------- | --------
default * | AK:***f9b          | Valid   | cn-beijing       | zh
aaa       | AK:******          | Invalid |                  |
test      | AK:***456          | Valid   |                  | en
ecs       | EcsRamRole:EcsTest | Valid   | cn-beijing       | en
```

#### 其他认证方式

阿里云CLI，可通过在`configure`命令后增加`--mode <authenticationMethod>`参数的方式来使用不同的认证方式，目前支持的认证方式如下：

| 验证方式  | 说明 |
| --------       | -------- |
| AK             | 使用AccessKey ID/Secret访问 |
| StsToken       | 使用STS Token访问    |
| RamRoleArn     | 使用RAM子账号的AssumeRole方式访问     |
| EcsRamRole     | 在ECS实例上通过EcsRamRole实现免密验证   |

#### 启用zsh/bash自动补全

- 使用`aliyun auto-completion`命令开启自动补全，目前支持zsh/bash
- 使用`aliyun auto-completion --uninstall`命令关闭自动补全

#### 配置Pretty JSON

阿里云CLI返回的执行结果为Raw JSON。如需Pretty JSON，可安装[jq](https://stedolan.github.io/jq/download/)工具，使用方式如下：

```
$ aliyun ecs DescribeRegions | jq
```

## 使用阿里云CLI

阿里云云产品的OpenAPI有RPC和RESTful两种风格，大部分产品使用的是RPC风格。不同风格的API的调用方法也不同。

您可以通过以下特点判断API风格：

- API参数中包含`Action`字段的是RPC风格，需要`PathPattern`参数的是Restful风格。
- 一般情况下，每个产品内，所有API的调用风格是统一的。
- 每个API仅支持特定的一种风格，传入错误的标识，可能会调用到其他API，或收到“ApiNotFound”的错误信息。

####调用RPC风格的API

阿里云CLI中RPC风格的API调用的基本结构如下：

```
$ aliyun <product> <operation> [--parameter1 value1 --parameter2 value2 ...]
```

代码示例：

```
$ aliyun rds DescribeDBInstances --PageSize 50
$ aliyun ecs DescribeRegions
$ aliyun rds DescribeDBInstanceAttribute --DBInstanceId xxxxxx
```


#### 调用RESTful风格的API

部分阿里云产品如容器服务的OpenAPI为Restful风格，调用Restful风格的接口与调用RPC风格的接口方式不同。参考以下代码示例，调用RESTful API。

- GET请求示例：

	```
	$ aliyun cs GET /clusters
	```

- POST请求示例：

	```
	$ aliyun cs POST /clusters --body "$(cat input.json)"
	```

- DELETE请求示例：

	```
	$ aliyun cs DELETE /clusters/ce2cdc26227e09c864d0ca0b2d5671a07
	```

#### 获取帮助信息

阿里云CLI集成了一部分产品的API和参数列表信息, 您可以使用如下命令来获取帮助：

- `$ aliyun help`: 获取产品列表

- `$ aliyun help <product>`: 获取产品的API信息

	如获取ECS的API信息：`$ aliyun help ecs`

- `$ aliyun help <product> <apiName>`: 获取API的调用信息

	如获取ECS的CreateInstance的信息： `aliyun help ecs CreateInstance`

#### 使用`--force`参数

阿里云CLI集成了一部分云产品的元数据，在调用时会对参数的合法性进行检查。如果使用了一个元数据中未包含的API或参数会导致`unknown api`或`unknown parameter`错误。可以使用`--force`参数跳过API和参数检查，强制调用元数据列表外的API和参数，如:

```
$ aliyun newproduct --version 2018-01-01 --endpoint newproduct.aliyuncs.com --param1 ... --force
```

在使用`--force`参数时，必须指定以下两个参数：

- `--version`: 指定API的版本，你可以在API文档中找到版本号，如ECS的版本号是`2014-05-26`。
- `--endpoint`: 指定产品的接入地址，一般产品接入地址是`product.aliyuncs.com`，或`product.en-central-1.aliyuncs.com`，请参考各产品的API文档。

#### 使用`--output`参数

阿里云产品的查询接口会返回json结构化数据，不方便阅读。例如：

```
$ aliyun ecs DescribeInstances
```

执行以上命令将得到以下json结果：

```
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


```
$ aliyun ecs DescribeInstances --output cols=InstanceId,Status
```

执行以上命令将得到以下形式的结果：
```
InstanceId             | Status
----------             | ------
i-12345678912345678123 | Stopped
i-abcdefghijklmnopqrst | Running
```

在使用`--output`参数时，必须指定以下子参数：

- `cols`: 表格的列名，需要与json数据中的字段相对应。如ECS DescribeInstances 接口返回结果中的字段`InstanceId` 以及 `Status`。

可选子参数：

- `rows`: 通过[jmespath](http://jmespath.org/)查询语句来指定表格行在json结果中的数据来源。当查询语句具有`Instances.Instance[]`的形式时，可以省略该参数。

#### 使用`--waiter`参数
该参数用于轮询实例信息直到出现特定状态。

例如使用ECS创建实例后，实例会有启动的过程。我们会不断的查询实例的运行状态，直到状态变为"Running"。

例如：

```
$ aliyun ecs DescribeInstances --InstanceIds '["i-12345678912345678123"]' --waiter expr='Instances.Instance[0].Status' to=Running
```

执行以上命令后,命令行程序将以一定时间间隔进行实例状态轮询，并在实例状态变为`Running`时停止轮询。

在使用`--waiter `参数时，必须指定以下两个子参数：

- `expr`: 通过[jmespath](http://jmespath.org/)查询语句来指定json结果中的被轮询字段。
- `to`: 被轮询字段的目标值。

可选子参数：

- `timeout`: 轮询的超时时间(秒)。
- `interval`: 轮询的间隔时间(秒)。

