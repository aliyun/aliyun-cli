[English](./README.md) | [简体中文](./README-CN.md)

<h1 align="center">Alibaba Cloud CLI</h1>

<p align="center">
  通过 OpenAPI 管理阿里云资源的跨平台命令行工具。
</p>

<p align="center">
  <a href="https://github.com/aliyun/aliyun-cli/actions/workflows/go.yml"><img src="https://github.com/aliyun/aliyun-cli/actions/workflows/go.yml/badge.svg" alt="构建状态"></a>
  <a href="https://github.com/aliyun/aliyun-cli/releases/latest"><img src="https://img.shields.io/github/v/release/aliyun/aliyun-cli" alt="最新版本"></a>
  <a href="https://github.com/aliyun/aliyun-cli/blob/master/LICENSE"><img src="https://img.shields.io/github/license/aliyun/aliyun-cli.svg" alt="许可证"></a>
</p>

阿里云命令行工具（`aliyun`）使用 Go 编写，支持 macOS、Linux 和 Windows。它提供交互式与非交互式凭证配置、OpenAPI 调用、本地 API 帮助、结构化输出、状态等待、安全策略和可扩展插件等能力。

> 阿里云 CLI 通过 OpenAPI 调用云服务。使用前请先开通目标服务，并确认当前身份具备所需权限。在可以使用临时凭证或身份凭证的场景中，请避免使用长期 AccessKey。

## 安装

选择以下任一方式：

```sh
# Homebrew
brew install aliyun-cli

# macOS 或 Linux 一键安装脚本
/bin/bash -c "$(curl -fsSL https://aliyuncli.alicdn.com/install.sh)"
```

官方安装包：

- [macOS 图形安装器](https://aliyuncli.alicdn.com/aliyun-cli-latest.pkg)
- [macOS Universal 压缩包](https://aliyuncli.alicdn.com/aliyun-cli-macosx-latest-universal.tgz)
- [Linux AMD64](https://aliyuncli.alicdn.com/aliyun-cli-linux-latest-amd64.tgz)
- [Linux ARM64](https://aliyuncli.alicdn.com/aliyun-cli-linux-latest-arm64.tgz)
- [Windows x64](https://aliyuncli.alicdn.com/aliyun-cli-windows-latest-amd64.zip)
- [GitHub 全部版本](https://github.com/aliyun/aliyun-cli/releases)

验证安装：

```sh
aliyun version
```

升级已有的独立安装：

```sh
aliyun upgrade
```

如果 CLI 通过包管理器安装，请优先使用对应包管理器的升级命令。

PATH 配置、源码构建和升级方法见[安装指南](./docs/zh-CN/installation.md)。

如果暂时不想在本地安装，可以通过[阿里云 Cloud Shell](https://shell.aliyun.com/)试用。

## 快速开始

### 1. 配置 Profile

本地交互环境推荐使用 OAuth，避免保存长期 AccessKey：

```sh
aliyun configure --mode OAuth --profile default
```

CLI 还支持 AK、StsToken、RamRoleArn、EcsRamRole、OIDC、External、CredentialsURI、ChainableRamRoleArn、CloudSSO、BearerToken，以及适用场景下的 Anonymous 模式。

```sh
# 查看并切换 Profile
aliyun configure list
aliyun configure switch --profile default
```

在生产或 CI 环境使用前，请阅读[配置与凭证指南](./docs/zh-CN/configuration.md)。

### 2. 调用 API

在元数据可用时，CLI 同时支持传统大驼峰命名（PascalCase）和元数据驱动的短横线命名（kebab-case）：

```sh
aliyun ecs DescribeRegions
aliyun ecs describe-regions
```

### 3. 查看帮助

```sh
aliyun --help
aliyun ecs DescribeInstances --help
aliyun ecs describe-instances --help
```

## 常用能力

### 筛选和格式化输出

```sh
aliyun ecs DescribeInstances \
  --output cols=InstanceId,Status rows=Instances.Instance[]

aliyun ecs describe-instances \
  --cli-query 'Instances.Instance[].{Id:InstanceId,Status:Status}'
```

### 等待资源进入目标状态

```sh
aliyun ecs DescribeInstances \
  --InstanceIds '["i-example"]' \
  --waiter expr='Instances.Instance[0].Status' to=Running
```

### 使用产品插件

```sh
aliyun plugin search <command-name>
aliyun plugin install --name <plugin-name>
aliyun plugin list
aliyun plugin update --name <plugin-name>
```

完整说明见[命令使用指南](./docs/zh-CN/usage.md)和[插件管理指南](./docs/zh-CN/plugins.md)。

## 文档

| 主题 | 简体中文 | English |
| --- | --- | --- |
| 文档索引 | [中文](./docs/README.md#简体中文) | [English](./docs/README.md#english) |
| 安装与升级 | [指南](./docs/zh-CN/installation.md) | [Guide](./docs/en/installation.md) |
| 配置与凭证 | [指南](./docs/zh-CN/configuration.md) | [Guide](./docs/en/configuration.md) |
| 命令、输出与自动化 | [指南](./docs/zh-CN/usage.md) | [Guide](./docs/en/usage.md) |
| 插件管理 | [指南](./docs/zh-CN/plugins.md) | [Guide](./docs/en/plugins.md) |

其他资源：

- [阿里云 CLI 官方文档](https://help.aliyun.com/zh/cli/)
- [OpenAPI 门户](https://api.aliyun.com/)
- [OpenAPI 问题诊断](https://api.aliyun.com/troubleshoot?source=github_sdk)
- [版本记录](./CHANGELOG.md)
- [安全策略](./SECURITY.md)

## 获取支持

- 搜索已有问题或[提交 GitHub Issue](https://github.com/aliyun/aliyun-cli/issues/new/choose)。
- 如果发现安全漏洞，请按照 [SECURITY.md](./SECURITY.md) 私下报告，不要提交公开 Issue。
- 版本说明和安装包见 [Releases](https://github.com/aliyun/aliyun-cli/releases)。

## 许可证

Alibaba Cloud CLI 使用 [Apache License 2.0](./LICENSE)。
