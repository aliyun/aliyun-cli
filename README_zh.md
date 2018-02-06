
# 阿里云命令行工具 (Aliyun Command Line Interface)

阿里云命令行工具是开源项目，[Github地址](https://github.com/aliyun/aliyun-cli)。

该版本的CLI为Go语言重构版本，目前处于**BETA**发布中，如果您还希望使用原有的Python版本，请切换到[python分支](https://github.com/aliyun/aliyun-cli/tree/python_final)。

## 简介

- 阿里云命令行工具是用Go语言编写的, 是基于阿里云OpenAPI打造的，用于管理阿里云资源的统一工具。通过下载和配置该工具，您可以在一个命令行方式下控制多个阿里云产品。
- 欢迎通过提交[Github Issue](https://github.com/aliyun/aliyun-cli/issues/new)与我们沟通。
- 同时更建议您加入阿里云官方SDK&CLI客户服务群，钉钉群号：11771185。


## 安装

### 下载及使用

CLI工具下载即可使用，支持Mac, Linux, Windows平台(x64版本)。

- [Mac](http://aliyun-cli.oss-cn-hangzhou.aliyuncs.com/aliyun-cli-macosx-0.33-amd64.tgz)
- [Linux](http://aliyun-cli.oss-cn-hangzhou.aliyuncs.com/aliyun-cli-linux-0.33-amd64.tgz)
- [Windows](http://aliyun-cli.oss-cn-hangzhou.aliyuncs.com/aliyun-cli-windows-0.33-amd64.tgz)

解压即可用，您可以将`aliyun`工具挪到`/usr/local/bin`目录下，或添加到`$PATH`中，以获得更便捷的访问。

**或**

## 下载源码后自行编译

当遇到不支持的平台，或其他需求时，请先安装并配置好golang环境，并按照如下步骤下载源码并编译。

```
$ mkdir -p $GOPATH/src/github.com/aliyun
$ cd $GOPATH/src/github.com/aliyun
$ git clone http://github.com/aliyun/aliyun-cli.git
$ git clone http://github.com/aliyun/aliyun-openapi-meta.git
$ cd aliyun-cli
$ make install
```

## 配置

- 在使用`aliyun`命令行工具前，您需要运行`aliuyun configure`命令进行配置。
- 要使用阿里云命令行工具，您需要一个云账号以及一对AccessKeyId和AccessKeySecret。 请在阿里云控制台中的[AccessKey管理控制台](https://ak-console.aliyun.com/#/accesskey)上创建和查看您的AK，或者联系您的系统管理员。
- 命令行工具使用OpenAPI方式访问云产品，您需要事先在阿里云开通这个产品并详细了解产品的OpenAPI。

### 基本配置

```
$ aliyun configure
Configuring profile 'default' ...
Aliyun Access Key ID [None]: <Your aliyun access key id>
Aliyun Access Key Secret [None]: <Your aliyun access key secret>
Default Region Id [None]: cn-hangzhou
Default output format [json]: json
Default Languate [zh]: zh
```

### 多用户

- `aliyun`支持多用户配置, 使用`$ aliyun configure --profile user1`可以配置指定的用户。
- `$ aliyun configure list`命令可以列出当前所有配置, 如下表, 其中在Profile列后面打*的为当前使用的默认配置。
- 在后续的调用中，您可以使用`--profile user1`参数来指定调用时的用户。

```
Profile   | CertificationMode | Valid   | AccessKeyId
--------- | ----------------- | ------- | ----------------
default * | AK                | Valid   | *************ac8
ram       | EcsRamRole        | Invalid |
test      | StsToken          | Invalid | **
leo       | AK                | Valid   | *************bb2
```

### 其他认证方式

- `aliyun`命令行工具，可通过在`configure`命令后增加`--mode ...`的方式来使用不同的认证方式，目前支持的认证方式如下：

| 验证方式  | 说明 |
| --------       | -------- |
| AK             | 使用AccessKeyId/Secret访问 |
| StsToken       | 使用StsToken访问    |
| RamRoleArn     | 使用Ram子账号的AssumeRole方式访问     |
| EcsRamRole     | 在ECS实例上通过EcsRamRole实现免密验证   |
| RsaKeyPair     | 使用Rsa公私钥方式(仅日本站支持)     |

### 使用自动补全功能

- TODO

## 如何使用

### 基本使用方式

```
$ aliyun <product> <operation> --parameter1 value1 --parameter2 value2 ...
```

例如:

```
$ aliyun rds DescribeDBInstances --PageSize 50
$ aliyun ecs DescribeRegions
$ aliyun rds DescribeDBInstanceAttribute --DBInstanceId xxxxxx
```

### 获取帮助信息

`aliyun`集成了一部分产品的API和参数列表的信息, 您可以使用如下命令来获取帮助：

- `$ aliyun help`: 打印产品列表
- `$ aliyun help ecs`: 获取产品的API信息
- `$ aliyuh help ecs CreateInstance`: 获取API的调用信息


### Restful风格的调用

部分阿里云产品的OpenAPI为Restful风格，调用Restful风格的接口与调用RPC风格的接口方式不用，需要使用：

- GET的例子:

```
$ aliyun cs GET /clusters
```

- POST的例子:

```
$ aliyun cs POST /clusters --body "$(cat input.json)"
```

- DELETE的例子

```
$ aliyun cs DELETE /clusters/ce2cdc26227e09c864d0ca0b2d5671a07
```

注：如何区分Rpc风格和Restful风格？

- 简单来说，API参数中，包含Action字段的是RPC风格，需要PathPattern的是Restful风格。
- 一般情况下，每个产品内，所有API的调用风格是统一的。
- 每个API仅支持特定的一种风格调用，传入错误的标识，可能会调用到其他API，或收到“ApiNotFound”的错误信息。

### 使用`--force`参数

`aliyun`命令行工具集成了一部分云产品的元数据，在调用时会进行参数的合法性检查，使用一个元数据中未包含的API或参数会导致`unknown api`或`unknown parameter`的报错。可以使用`--force`参数来跳过API和参数检查，强制调用元数据列表外的API和参数，如:

```
$ aliyun newproduct --version 2018-01-01 --endpoint newproduct.aliyuncs.com --param1 ... --force
```

使用前请阅读阿里云各产品的OpenAPI文档，来了解OpenAPI的使用方式及参数列表，您可以在[阿里云API平台](https://developer.aliyun.com/api)获取产品API文档。在使用`--force`参数时，有两个参数是额外需要指定的:

- `--version`: 指定OpenAPI的版本，你可以在API文档中找到这个版本号，如: ECS的版本号是`2014-05-26`。
- `--endpoint`: 指定产品的接入地址，一般产品接入地址是`product.aliyuncs.com`，或`product.en-central-1.aliyuncs.com`，请参照具体文档。

