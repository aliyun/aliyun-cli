# 插件管理

[文档索引](../README.md) | [English](../en/plugins.md)

阿里云 CLI 可以通过插件扩展产品命令。如果根帮助中某个产品显示“插件可用但未安装”，无需替换主 CLI 二进制即可安装对应插件。

## 查找插件

```sh
# 已安装插件
aliyun plugin list

# 配置的远程索引中可用的插件
aliyun plugin list-remote

# 根据命令或产品名搜索插件
aliyun plugin search <command-name>

# 查看一个已安装插件
aliyun plugin show --name <plugin-name>
```

根帮助也会标记产品插件是否可用：

```sh
aliyun --help
```

## 安装

从远程索引安装一个插件：

```sh
aliyun plugin install --name <plugin-name>
```

安装多个插件：

```sh
aliyun plugin install --names <plugin-one> <plugin-two>
```

指定版本或允许预发布版本：

```sh
aliyun plugin install --name <plugin-name> --version <version>
aliyun plugin install --name <plugin-name> --enable-pre
```

安装本地或 HTTPS 插件包：

```sh
aliyun plugin install --package ./plugin-package.tar.gz
aliyun plugin install --package https://example.com/plugin-package.zip
```

支持 `.zip`、`.tar.gz` 和 `.tgz`。从本地文件或 URL 安装时，请确认插件来源可信。

## 更新和卸载

```sh
# 更新一个插件
aliyun plugin update --name <plugin-name>

# 更新全部已安装插件
aliyun plugin update

# 允许更新到预发布版本
aliyun plugin update --name <plugin-name> --enable-pre

# 卸载一个插件
aliyun plugin uninstall --name <plugin-name>
```

## 插件源

查看全局插件设置：

```sh
aliyun configure plugin-settings show
```

设置或清除自定义插件树根地址：

```sh
aliyun configure plugin-settings set --source-base https://example.com/plugins
aliyun configure plugin-settings clear
```

单次命令可以使用 `--source-base <url>` 覆盖全局设置。环境变量 `ALIBABA_CLOUD_CLI_PLUGIN_SOURCE_BASE` 的优先级高于已保存设置。

## 自动安装产品插件

自动安装可以保存在 Profile 中：

```sh
aliyun configure set --profile default --auto-plugin-install true
```

是否允许预发布包需要单独设置：

```sh
aliyun configure set \
  --profile default \
  --auto-plugin-install true \
  --auto-plugin-install-enable-pre true
```

在需要可复现的 CI 环境中，应显式安装并固定所需插件，不建议依赖自动安装。

## 存储与问题排查

插件默认保存在 `~/.aliyun/plugins`。可以使用 `ALIBABA_CLOUD_CLI_PLUGINS_DIR` 指定其他目录。

常用检查命令：

```sh
aliyun plugin list
aliyun plugin show --name <plugin-name>
aliyun plugin update --name <plugin-name>
```

安装失败时检查：

- 插件名是否出现在 `plugin list-remote`；
- 是否可以访问配置的插件源；
- 插件包是否支持当前操作系统和 CPU 架构；
- 插件要求的最低 CLI 版本是否高于当前版本；
- 除非正在排查本地安装损坏，否则不要手动修改插件 manifest。

隐藏的内部批量安装命令不属于公开工作流，也不提供兼容性承诺。
