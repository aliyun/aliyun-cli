[English](./README.md) | [简体中文](./README-CN.md)

<h1 align="center">Alibaba Cloud CLI</h1>

<p align="center">
  A cross-platform command-line tool for managing Alibaba Cloud resources through OpenAPI.
</p>

<p align="center">
  <a href="https://github.com/aliyun/aliyun-cli/actions/workflows/go.yml"><img src="https://github.com/aliyun/aliyun-cli/actions/workflows/go.yml/badge.svg" alt="Build status"></a>
  <a href="https://github.com/aliyun/aliyun-cli/releases/latest"><img src="https://img.shields.io/github/v/release/aliyun/aliyun-cli" alt="Latest release"></a>
  <a href="https://github.com/aliyun/aliyun-cli/blob/master/LICENSE"><img src="https://img.shields.io/github/license/aliyun/aliyun-cli.svg" alt="License"></a>
</p>

Alibaba Cloud CLI (`aliyun`) is written in Go and works on macOS, Linux, and Windows. It supports interactive and non-interactive credential profiles, OpenAPI calls, local API help, structured output, waiters, safety controls, and an extensible plugin system.

> Alibaba Cloud CLI invokes cloud services through OpenAPI. Activate the target service first and make sure your identity has the required permissions. Avoid long-lived AccessKeys where a temporary or identity-based credential mode is available.

## Install

Choose one of the following methods:

```sh
# Homebrew
brew install aliyun-cli

# macOS or Linux installer script
/bin/bash -c "$(curl -fsSL https://aliyuncli.alicdn.com/install.sh)"
```

Official packages:

- [macOS GUI installer](https://aliyuncli.alicdn.com/aliyun-cli-latest.pkg)
- [macOS universal archive](https://aliyuncli.alicdn.com/aliyun-cli-macosx-latest-universal.tgz)
- [Linux AMD64](https://aliyuncli.alicdn.com/aliyun-cli-linux-latest-amd64.tgz)
- [Linux ARM64](https://aliyuncli.alicdn.com/aliyun-cli-linux-latest-arm64.tgz)
- [Windows x64](https://aliyuncli.alicdn.com/aliyun-cli-windows-latest-amd64.zip)
- [All GitHub releases](https://github.com/aliyun/aliyun-cli/releases)

Verify the installation:

```sh
aliyun version
```

Upgrade an existing standalone installation:

```sh
aliyun upgrade
```

If the CLI was installed through a package manager, prefer that package manager's upgrade command.

See the [installation guide](./docs/en/installation.md) for PATH setup, source builds, and upgrades.

To try the CLI without installing it locally, open [Alibaba Cloud Shell](https://shell.aliyun.com/):

<a href="https://shell.aliyun.com/" target="cloudshell">
  <img src="https://img.alicdn.com/tfs/TB1wt1zq9zqK1RjSZFpXXakSXXa-1066-166.png" width="180" alt="Alibaba Cloud Shell" />
</a>

## Quick start

### 1. Configure a profile

For an interactive local environment, OAuth avoids storing a long-lived AccessKey:

```sh
aliyun configure --mode OAuth --profile default
```

You can also use AK, StsToken, RamRoleArn, EcsRamRole, OIDC, External, CredentialsURI, ChainableRamRoleArn, CloudSSO, BearerToken, or Anonymous mode where applicable.

```sh
# Inspect and select profiles
aliyun configure list
aliyun configure switch --profile default
```

See [configuration and credentials](./docs/en/configuration.md) before using a profile in production or CI.

### 2. Call an API

Both the traditional PascalCase form and the metadata-driven kebab-case form are supported where metadata is available:

```sh
aliyun ecs DescribeRegions
aliyun ecs describe-regions
```

### 3. Explore help

```sh
aliyun --help
aliyun ecs DescribeInstances --help
aliyun ecs describe-instances --help
```

## Common capabilities

### Filter and format output

```sh
aliyun ecs DescribeInstances \
  --output cols=InstanceId,Status rows=Instances.Instance[]

aliyun ecs describe-instances \
  --cli-query 'Instances.Instance[].{Id:InstanceId,Status:Status}'
```

### Wait for a resource state

```sh
aliyun ecs DescribeInstances \
  --InstanceIds '["i-example"]' \
  --waiter expr='Instances.Instance[0].Status' to=Running
```

### Use product plugins

```sh
aliyun plugin search <command-name>
aliyun plugin install --name <plugin-name>
aliyun plugin list
aliyun plugin update --name <plugin-name>
```

See [command usage](./docs/en/usage.md) and [plugin management](./docs/en/plugins.md) for the complete workflows.

## Troubleshoot

[OpenAPI Troubleshoot](https://api.aliyun.com/troubleshoot?source=github_sdk) helps diagnose OpenAPI request failures by `RequestId` or error message.

## Documentation

| Topic | English | 简体中文 |
| --- | --- | --- |
| Documentation index | [English](./docs/README.md#english) | [中文](./docs/README.md#简体中文) |
| Installation and upgrade | [Guide](./docs/en/installation.md) | [指南](./docs/zh-CN/installation.md) |
| Configuration and credentials | [Guide](./docs/en/configuration.md) | [指南](./docs/zh-CN/configuration.md) |
| Commands, output, and automation | [Guide](./docs/en/usage.md) | [指南](./docs/zh-CN/usage.md) |
| Plugin management | [Guide](./docs/en/plugins.md) | [指南](./docs/zh-CN/plugins.md) |

Additional resources:

- [Official Alibaba Cloud CLI documentation](https://www.alibabacloud.com/help/en/cli/)
- [OpenAPI Portal](https://api.aliyun.com/)
- [Changelog](./CHANGELOG.md)
- [Security policy](./SECURITY.md)

## Support

- Search existing or [open a GitHub issue](https://github.com/aliyun/aliyun-cli/issues/new/choose).
- For a suspected security vulnerability, follow [SECURITY.md](./SECURITY.md) instead of opening a public issue.
- Release notes and binaries are available on the [Releases page](https://github.com/aliyun/aliyun-cli/releases).

## License

Alibaba Cloud CLI is licensed under the [Apache License 2.0](./LICENSE).
