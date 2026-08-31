# 安装与升级

[文档索引](../README.md) | [English](../en/installation.md)

阿里云 CLI 支持 macOS、Linux 和 Windows。安装后的可执行文件名为 `aliyun`，Windows 上为 `aliyun.exe`。

## 下载安装包

下载并解压与当前平台匹配的安装包，再把可执行文件放入 `PATH` 中的目录。

| 平台 | 安装包 |
| --- | --- |
| macOS | [图形安装器](https://aliyuncli.alicdn.com/aliyun-cli-latest.pkg) |
| macOS | [Universal 压缩包](https://aliyuncli.alicdn.com/aliyun-cli-macosx-latest-universal.tgz) |
| Linux AMD64 | [压缩包](https://aliyuncli.alicdn.com/aliyun-cli-linux-latest-amd64.tgz) |
| Linux ARM64 | [压缩包](https://aliyuncli.alicdn.com/aliyun-cli-linux-latest-arm64.tgz) |
| Windows x64 | [ZIP 压缩包](https://aliyuncli.alicdn.com/aliyun-cli-windows-latest-amd64.zip) |

所有已发布版本见 [GitHub Releases](https://github.com/aliyun/aliyun-cli/releases)。

在 macOS 或 Linux 上可以这样手动安装：

```sh
tar -xzf <下载的压缩包>.tgz
chmod +x aliyun
sudo mv aliyun /usr/local/bin/aliyun
aliyun version
```

Windows 用户解压 `aliyun.exe` 后，将所在目录加入用户或系统的 `Path` 环境变量，再重新打开终端。

## Homebrew

```sh
brew install aliyun-cli
aliyun version
```

通过 Homebrew 升级：

```sh
brew update
brew upgrade aliyun-cli
```

## 一键安装脚本

一键安装脚本支持 macOS 和 Linux：

```sh
/bin/bash -c "$(curl -fsSL https://aliyuncli.alicdn.com/install.sh)"
```

如果所在环境对软件供应链有严格要求，请先下载并审查脚本，再执行安装。

## 升级已有 CLI

CLI 内置升级命令：

```sh
aliyun upgrade
```

如果通过包管理器安装，优先使用对应包管理器升级。独立二进制安装也可以直接替换为 Releases 页面中的新版本。

## 从源码构建

环境要求：

- 支持 submodule 的 Git
- [`go.mod`](../../go.mod) 声明的 Go 版本
- 所选 Make 目标需要的平台工具

拉取子模块并构建带内置 metadata 的发布形式二进制：

```sh
git clone --recurse-submodules https://github.com/aliyun/aliyun-cli.git
cd aliyun-cli
make build
./out/aliyun version
```

开发构建会直接读取已检出的 metadata 子模块：

```sh
go build -o out/aliyun ./main
```

## 常见问题

- 提示 `command not found`：确认可执行文件目录已加入 `PATH`，然后重新打开 Shell。
- macOS/Linux 提示没有权限：执行 `chmod +x aliyun`。
- CPU 架构不匹配：根据 AMD64/x86_64 或 ARM64 选择安装包。
- 升级后仍运行旧版本：使用 `command -v aliyun`，Windows 使用 `where aliyun`，检查是否存在重复安装。

下一步：[配置凭证](./configuration.md)。
